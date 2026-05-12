package cron

import (
    "context"
    "log"
    "sync"
    "time"

    "ecommerce-demo/app/order/internal/config"
    "ecommerce-demo/app/order/internal/repo"
)

/*
  订单超时扫描定时任务

  职责：作为延迟队列的兜底机制，处理以下情况：
  1. 延迟消息丢失或未被消费
  2. 延迟消息处理失败但 Nack 次数超限被丢弃
  3. 系统重启期间的超时订单

  工作机制：
  1. 每 N 秒扫描一次数据库
  2. 找出所有已过期但仍为待支付状态的订单
  3. 批量处理超时订单

  设计考量：
  - 轻量级：只查询和更新，不做复杂计算
  - 安全：限制每次处理数量，避免长时间锁表
  - 可观测：详细日志记录处理过程
  - 容错：任何异常不阻塞后续扫描
*/

type OrderTimeoutJob struct {
    orderRepo repo.OrderRepo
    config    config.OrderTimeoutConfig
    stopCh    chan struct{}
    stopOnce  sync.Once
    isRunning bool
    runningMu sync.Mutex
}

func NewOrderTimeoutJob(orderRepo repo.OrderRepo, cfg config.OrderTimeoutConfig) *OrderTimeoutJob {
    return &OrderTimeoutJob{
        orderRepo: orderRepo,
        config:    cfg,
        stopCh:    make(chan struct{}),
    }
}

/*
  Start 启动定时扫描任务

  在独立 goroutine 中运行，不阻塞主流程

  扫描间隔由配置项 scanIntervalSeconds 控制，默认60秒
  每次最多处理 maxScanCount 条订单，默认100条
*/
func (j *OrderTimeoutJob) Start() {
    j.runningMu.Lock()
    if j.isRunning {
        j.runningMu.Unlock()
        log.Println("⚠️ 定时扫描任务已在运行，忽略重复启动")
        return
    }
    j.isRunning = true
    j.runningMu.Unlock()

    go j.run()
    log.Printf("⏰ 订单超时扫描定时任务已启动，扫描间隔: %d秒，每次最多处理: %d条",
        j.config.ScanIntervalSeconds, j.config.MaxScanCount)
}

func (j *OrderTimeoutJob) Stop() {
    j.stopOnce.Do(func() {
        close(j.stopCh)
        j.runningMu.Lock()
        j.isRunning = false
        j.runningMu.Unlock()
        log.Println("🛑 订单超时扫描定时任务已停止")
    })
}

func (j *OrderTimeoutJob) run() {
    // 先等待一个扫描间隔，避免服务刚启动就立即扫描
    interval := time.Duration(j.config.ScanIntervalSeconds) * time.Second
    if interval <= 0 {
        interval = 60 * time.Second // 默认60秒
    }

    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-j.stopCh:
            return
        case <-ticker.C:
            j.scanAndProcess()
        }
    }
}

func (j *OrderTimeoutJob) scanAndProcess() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 1. 查询超时订单
    orders, err := j.orderRepo.ListTimeoutOrders(ctx, int32(j.config.MaxScanCount))
    if err != nil {
        log.Printf("❌ 扫描超时订单失败: %v", err)
        return
    }

    if len(orders) == 0 {
        return
    }

    log.Printf("📋 扫描到 %d 个超时订单，开始处理...", len(orders))

    // 2. 逐个处理超时订单
    successCount := 0
    failCount := 0

    for _, order := range orders {
        err := j.orderRepo.TimeoutOrderTx(ctx, order.OrderNo, order.ProductID, order.Count)
        if err != nil {
            failCount++
            log.Printf("❌ 处理超时订单失败: OrderNo=%s, Err=%v", order.OrderNo, err)
            continue
        }
        successCount++
        log.Printf("✅ 超时订单处理成功: OrderNo=%s, ProductID=%d, Count=%d",
            order.OrderNo, order.ProductID, order.Count)
    }

    log.Printf("📊 超时订单批量处理完成: 成功=%d, 失败=%d", successCount, failCount)
}
