package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/repo"
	"ecommerce-demo/common/metrics"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

/*
  可靠消费者实现（自动重连版）

  可靠性保障：
  1. 手动ACK — 处理完成后确认
  2. QoS 预取控制 — 控制并发，防止过载
  3. 死信队列(DLQ) — 永久失败消息存储
  4. 连接自动重连 — 断线后自动恢复，无需重启服务
  5. 优雅退出 — 处理完进行中的消息后退出
  6. 幂等消费 — Redis SetNX 防止重复处理
*/

type MessageMetadata struct {
	MessageID  string    `json:"messageId"`
	TraceID    string    `json:"traceId"`
	RetryCount int       `json:"retryCount"`
	FirstTry   time.Time `json:"firstTry"`
}

type Consumer interface {
	Start()
	Stop()
	IsRunning() bool
}

type ReliableConsumer struct {
	url       string
	cfg       config.MQConsumerConfig
	queueName string
	orderRepo repo.OrderRepo
	rdb       *redis.ClusterClient

	mu        sync.RWMutex
	conn      *amqp.Connection
	channel   *amqp.Channel
	closeCh   chan struct{}
	doneCh    chan struct{}
	isRunning bool
	stopping  bool
}

func NewReliableConsumer(
	cfg config.Config,
	orderRepo repo.OrderRepo,
	rdb *redis.ClusterClient,
) (Consumer, error) {
	c := &ReliableConsumer{
		url:       cfg.RabbitMQ.Url,
		cfg:       cfg.MQConsumer,
		queueName: cfg.RabbitMQ.QueueName,
		orderRepo: orderRepo,
		rdb:       rdb,
		closeCh:   make(chan struct{}),
		doneCh:    make(chan struct{}),
	}

	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}

	return c, nil
}

// connect 建立连接和通道，声明队列
func (c *ReliableConsumer) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	// 设置 QoS
	prefetchCount := c.cfg.PrefetchCount
	if prefetchCount <= 0 {
		prefetchCount = 10
	}
	if err = ch.Qos(prefetchCount, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("设置 QoS 失败: %w", err)
	}

	// 声明主队列
	_, err = ch.QueueDeclare(
		c.queueName,
		true, false, false, false, nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("声明主队列失败: %w", err)
	}

	// 声明死信队列相关
	if c.cfg.EnableDLQ {
		if err := declareDLQ(ch, c.cfg); err != nil {
			log.Printf("声明死信队列失败（非致命）: %v", err)
		}
	}

	c.conn = conn
	c.channel = ch

	return nil
}

func declareDLQ(ch *amqp.Channel, cfg config.MQConsumerConfig) error {
	if err := ch.ExchangeDeclare(cfg.DLQExchangeName, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	_, err := ch.QueueDeclare(cfg.DLQQueueName, true, false, false, false, nil)
	if err != nil {
		return err
	}
	return ch.QueueBind(cfg.DLQQueueName, "dlq", cfg.DLQExchangeName, false, nil)
}

func (c *ReliableConsumer) Start() {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return
	}
	c.isRunning = true
	c.mu.Unlock()

	log.Printf("可靠消费者已启动，队列: %s，预取数: %d，最大重试: %d",
		c.queueName, c.cfg.PrefetchCount, c.cfg.MaxRetryTimes)

	go c.runLoop()
}

func (c *ReliableConsumer) Stop() {
	c.mu.Lock()
	if c.stopping {
		c.mu.Unlock()
		return
	}
	c.stopping = true
	c.isRunning = false
	c.mu.Unlock()

	log.Println("正在优雅关闭消费者...")
	close(c.closeCh)

	timeout := c.cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = 30
	}
	select {
	case <-c.doneCh:
		log.Println("消费者已完全停止")
	case <-time.After(time.Duration(timeout) * time.Second):
		log.Printf("优雅关闭超时（%d秒），强制退出", timeout)
	}

	c.mu.Lock()
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()
}

func (c *ReliableConsumer) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

// runLoop 主循环：消费消息 + 断线重连
func (c *ReliableConsumer) runLoop() {
	defer close(c.doneCh)

	for {
		select {
		case <-c.closeCh:
			return
		default:
		}

		c.consumeLoop()

		// 退出前检查
		select {
		case <-c.closeCh:
			return
		default:
		}

		log.Println("消费者连接断开，3秒后重连...")
		time.Sleep(3 * time.Second)

		if err := c.connect(); err != nil {
			log.Printf("重连失败: %v，继续重试...", err)
			continue
		}
		log.Println("消费者重连成功")
	}
}

// consumeLoop 单次消费循环（持续消费直到连接断开）
func (c *ReliableConsumer) consumeLoop() {
	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()

	if ch == nil || ch.IsClosed() {
		return
	}

	// 监听连接关闭通知
	connClose := c.conn.NotifyClose(make(chan *amqp.Error, 1))
	chClose := ch.NotifyClose(make(chan *amqp.Error, 1))

	msgs, err := ch.Consume(
		c.queueName,
		c.cfg.ConsumerTag,
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		log.Printf("注册消费者失败: %v", err)
		return
	}

	workerNum := 20
	var wg sync.WaitGroup
	for i := 0; i < workerNum; i++ {
		wg.Add(1)
		go c.worker(i, msgs, &wg)
	}
	log.Printf("启动 %d 个 Worker 处理消息", workerNum)

	// 阻塞直到连接断开或收到关闭信号
	select {
	case <-c.closeCh:
	case <-connClose:
		log.Println("RabbitMQ 连接断开")
	case <-chClose:
		log.Println("RabbitMQ 通道断开")
	}

	wg.Wait()
}

func (c *ReliableConsumer) worker(workerID int, msgs <-chan amqp.Delivery, wg *sync.WaitGroup) {
	defer wg.Done()

	for d := range msgs {
		c.processMessage(workerID, d)
	}
}

func (c *ReliableConsumer) processMessage(workerID int, d amqp.Delivery) {
	var msg OrderMsg
	metadata := c.extractMetadata(d)

	if err := json.Unmarshal(d.Body, &msg); err != nil {
		log.Printf("[Worker-%d] 消息解析失败: %v", workerID, err)
		d.Ack(false)
		return
	}

	log.Printf("[Worker-%d] 收到消息: OrderNo=%s，重试=%d",
		workerID, msg.OrderNo, metadata.RetryCount)

	retryCount := c.getRetryCount(d)
	maxRetries := c.cfg.MaxRetryTimes
	if maxRetries <= 0 {
		maxRetries = 3
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 幂等检查
	idempotentKey := fmt.Sprintf("{idempotent}:order:%s", msg.OrderNo)
	acquired, err := c.rdb.SetNX(ctx, idempotentKey, metadata.MessageID, 24*time.Hour).Result()
	if err != nil {
		d.Nack(false, true)
		return
	}
	if !acquired {
		d.Ack(false)
		return
	}

	// 执行落库
	expireTime := time.Unix(msg.ExpireTime, 0)
	newOrder := &repo.Order{
		OrderNo:     msg.OrderNo,
		UserID:      msg.UserID,
		ProductID:   msg.ProductID,
		Count:       msg.Count,
		TotalAmount: msg.TotalAmount,
		Status:      0,
		ExpireTime:  &expireTime,
	}

	err = c.orderRepo.CreateOrderTx(ctx, newOrder, expireTime, msg.Count)
	if err != nil {
		log.Printf("[Worker-%d] 落库失败: OrderNo=%s, Err=%v, 重试=%d/%d",
			workerID, msg.OrderNo, err, retryCount, maxRetries)
		c.rdb.Del(ctx, idempotentKey)

		if retryCount >= maxRetries {
			metrics.MQConsumeTotal.WithLabelValues("order.created", "dql").Inc()
			c.handleDeadLetter(workerID, d, msg, err, metadata)
		} else {
			metrics.MQConsumeTotal.WithLabelValues("order.created", "fail").Inc()
			d.Nack(false, true)
		}
		return
	}

	log.Printf("[Worker-%d] 落库成功: OrderNo=%s", workerID, msg.OrderNo)
	metrics.MQConsumeTotal.WithLabelValues("order.created", "success").Inc()
	d.Ack(false)
}

func (c *ReliableConsumer) handleDeadLetter(workerID int, d amqp.Delivery, msg OrderMsg, err error, metadata MessageMetadata) {
	if !c.cfg.EnableDLQ {
		d.Ack(false)
		return
	}

	deadLetter := map[string]interface{}{
		"originalMessage": msg,
		"error":           err.Error(),
		"failedAt":        time.Now().Format(time.RFC3339),
		"retryCount":      metadata.RetryCount,
	}
	body, _ := json.Marshal(deadLetter)

	pubErr := c.channel.PublishWithContext(context.Background(),
		c.cfg.DLQExchangeName, "dlq", false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})

	if pubErr != nil {
		log.Printf("[Worker-%d] 发送死信失败: OrderNo=%s, Err=%v", workerID, msg.OrderNo, pubErr)
	} else {
		log.Printf("[Worker-%d] 消息已入死信队列: OrderNo=%s", workerID, msg.OrderNo)
	}
	d.Ack(false)
}

func (c *ReliableConsumer) extractMetadata(d amqp.Delivery) MessageMetadata {
	metadata := MessageMetadata{
		MessageID: uuid.New().String(),
		TraceID:   uuid.New().String(),
		FirstTry:  time.Now(),
	}
	if d.Headers != nil {
		if msgID, ok := d.Headers["x-message-id"].(string); ok {
			metadata.MessageID = msgID
		}
	}
	if metadata.MessageID == "" {
		metadata.MessageID = fmt.Sprintf("%d", d.DeliveryTag)
	}
	return metadata
}

func (c *ReliableConsumer) getRetryCount(d amqp.Delivery) int {
	if d.Headers == nil {
		return 0
	}
	if xDeath, ok := d.Headers["x-death"].([]interface{}); ok && len(xDeath) > 0 {
		if deathMap, ok := xDeath[0].(amqp.Table); ok {
			if count, ok := deathMap["count"].(int64); ok {
				return int(count)
			}
		}
	}
	return 0
}
