package outbox

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/mq"
	"ecommerce-demo/app/order/internal/repo"
	"ecommerce-demo/common/metrics"
)

/*
  Outbox Worker（本地消息表轮询投递器）

  职责：
  1. 原子抢占 outbox 表中的待发送消息，支持多副本和超时接管
  2. 根据消息类型投递到对应的 MQ 队列
  3. 投递成功 → 标记 completed
  4. 投递失败 → 指数退避重试，达到上限后标记最终失败

  消息类型 → MQ 路由：
  - order.created      → 订单创建领域事件队列
  - order.delay.check  → 延迟队列（超时检查，带 TTL）
*/

type Worker struct {
	outboxRepo   repo.OutboxRepo
	producer     mq.Producer
	pollInterval time.Duration
	claimTimeout time.Duration
	batchSize    int
	stopCh       chan struct{}
	doneCh       chan struct{}
	stopOnce     sync.Once
	isRunning    bool
	runningMu    sync.Mutex
}

func NewWorker(outboxRepo repo.OutboxRepo, producer mq.Producer, cfg config.Config) *Worker {
	pollInterval := time.Duration(cfg.Outbox.PollIntervalSeconds) * time.Second
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	batchSize := cfg.Outbox.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}
	claimTimeout := time.Duration(cfg.Outbox.ClaimTimeoutSeconds) * time.Second
	if claimTimeout <= 0 {
		claimTimeout = time.Minute
	}

	return &Worker{
		outboxRepo:   outboxRepo,
		producer:     producer,
		pollInterval: pollInterval,
		claimTimeout: claimTimeout,
		batchSize:    batchSize,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}
}

func (w *Worker) Start() {
	w.runningMu.Lock()
	if w.isRunning {
		w.runningMu.Unlock()
		return
	}
	w.isRunning = true
	w.runningMu.Unlock()

	go w.run()
	log.Printf("Outbox Worker 已启动，轮询间隔=%v，批量大小=%d", w.pollInterval, w.batchSize)
}

func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopCh)
		w.runningMu.Lock()
		w.isRunning = false
		w.runningMu.Unlock()
		log.Println("Outbox Worker 已停止")
	})
}

func (w *Worker) run() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	defer close(w.doneCh)

	// 启动后立即执行一次
	w.pollAndPublish()

	for {
		select {
		case <-w.stopCh:
			// 退出前最后一次处理
			w.pollAndPublish()
			return
		case <-ticker.C:
			w.pollAndPublish()
		}
	}
}

func (w *Worker) pollAndPublish() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pendingCount, err := w.outboxRepo.CountPendingMessages(ctx)
	if err != nil {
		log.Printf("Outbox Worker 统计待投递消息失败: %v", err)
	} else {
		metrics.OutboxPendingGauge.Set(float64(pendingCount))
	}

	records, err := w.outboxRepo.ClaimPendingMessages(ctx, w.batchSize, w.claimTimeout)
	if err != nil {
		log.Printf("Outbox Worker 拉取消息失败: %v", err)
		return
	}

	if len(records) == 0 {
		return
	}

	log.Printf("Outbox Worker 拉取到 %d 条待投递消息", len(records))

	for _, record := range records {
		w.processRecord(ctx, record)
	}
}

func (w *Worker) processRecord(ctx context.Context, record *repo.OutboxRecord) {
	switch record.MessageType {
	case repo.OutboxTypeOrderCreated:
		w.publishOrderCreated(ctx, record)
	case repo.OutboxTypeOrderDelay:
		w.publishOrderDelay(ctx, record)
	default:
		log.Printf("Outbox Worker 未知消息类型: %s, ID=%d", record.MessageType, record.ID)
		w.markFailed(ctx, record, "未知消息类型: "+record.MessageType)
	}
}

// publishOrderCreated 投递订单创建消息到普通队列
func (w *Worker) publishOrderCreated(ctx context.Context, record *repo.OutboxRecord) {
	var msg mq.OrderMsg
	if err := json.Unmarshal([]byte(record.Payload), &msg); err != nil {
		log.Printf("Outbox Worker 解析订单创建消息失败: ID=%d, Err=%v", record.ID, err)
		w.markFailed(ctx, record, err.Error())
		return
	}

	if err := w.producer.PublishOrder(ctx, &msg); err != nil {
		metrics.MQPublishTotal.WithLabelValues("order.created", "fail").Inc()
		log.Printf("Outbox Worker 投递订单创建消息失败: ID=%d, OrderNo=%s, Err=%v",
			record.ID, msg.OrderNo, err)
		w.markFailed(ctx, record, err.Error())
		return
	}

	metrics.MQPublishTotal.WithLabelValues("order.created", "success").Inc()
	w.markCompleted(ctx, record)
	log.Printf("Outbox Worker 投递成功(order.created): OrderNo=%s", msg.OrderNo)
}

// publishOrderDelay 投递延迟超时检查消息
func (w *Worker) publishOrderDelay(ctx context.Context, record *repo.OutboxRecord) {
	var msg mq.DelayOrderMsg
	if err := json.Unmarshal([]byte(record.Payload), &msg); err != nil {
		log.Printf("Outbox Worker 解析延迟消息失败: ID=%d, Err=%v", record.ID, err)
		w.markFailed(ctx, record, err.Error())
		return
	}

	// 计算距离过期还有多久（分钟）
	remaining := time.Until(msg.ExpireTime)
	if remaining <= 0 {
		// 已经过期，直接标记完成，让定时扫描兜底
		log.Printf("Outbox Worker 延迟消息已过期，跳过: OrderNo=%s", msg.OrderNo)
		w.markCompleted(ctx, record)
		return
	}

	expireMinutes := int(remaining.Minutes()) + 1 // 向上取整，确保在过期后触发

	if err := w.producer.PublishDelayOrder(ctx, &msg, expireMinutes); err != nil {
		metrics.MQPublishTotal.WithLabelValues("order.delay.check", "fail").Inc()
		log.Printf("Outbox Worker 投递延迟消息失败: ID=%d, OrderNo=%s, Err=%v",
			record.ID, msg.OrderNo, err)
		w.markFailed(ctx, record, err.Error())
		return
	}

	metrics.MQPublishTotal.WithLabelValues("order.delay.check", "success").Inc()
	w.markCompleted(ctx, record)
	log.Printf("Outbox Worker 投递成功(order.delay.check): OrderNo=%s, TTL=%dm", msg.OrderNo, expireMinutes)
}

func (w *Worker) markCompleted(ctx context.Context, record *repo.OutboxRecord) {
	if err := w.outboxRepo.MarkCompleted(ctx, record.ID, record.LockToken); err != nil {
		log.Printf("Outbox Worker 完成状态更新失败: ID=%d, Err=%v", record.ID, err)
	}
}

func (w *Worker) markFailed(ctx context.Context, record *repo.OutboxRecord, errMsg string) {
	if err := w.outboxRepo.MarkFailed(ctx, record.ID, record.LockToken, errMsg); err != nil {
		log.Printf("Outbox Worker 失败状态更新失败: ID=%d, Err=%v", record.ID, err)
	}
}
