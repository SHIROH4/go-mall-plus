package mq

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "ecommerce-demo/app/order/internal/config"
    "ecommerce-demo/app/order/internal/repo"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    amqp "github.com/rabbitmq/amqp091-go"
)

/*
  完整手动ACK消费者实现

  可靠性保障：
  1. QoS 预取控制 - 控制消费速率，防止内存溢出
  2. 手动ACK机制 - 确保消息处理完成后才确认
  3. 重试次数限制 - 防止无限重试
  4. 死信队列(DLQ) - 永久失败消息存储
  5. 优雅退出 - 服务停止时处理完正在处理的消息
  6. 消息追踪ID - 问题排查

  ACK/NACK 策略：
  ┌─────────────────────────────────────────────────────────────────┐
  │ 消息状态                    │ 处理方式                          │
  ├─────────────────────────────────────────────────────────────────┤
  │ 解析成功 + 业务处理成功      │ ACK 确认                          │
  │ 解析失败                    │ ACK 丢弃（无效消息）               │
  │ 业务处理失败 + 重试次数 < 3  │ Nack + Requeue（重新入队）         │
  │ 业务处理失败 + 重试次数 >= 3 │ Nack + 发送到DLQ（死信队列）       │
  │ Redis/数据库故障             │ Nack + Requeue（稍后重试）         │
  └─────────────────────────────────────────────────────────────────┘
*/

// MessageMetadata 消息元数据（用于追踪）
type MessageMetadata struct {
    MessageID  string    `json:"messageId"`  // 消息唯一ID
    TraceID    string    `json:"traceId"`    // 链路追踪ID
    RetryCount int       `json:"retryCount"` // 当前重试次数
    FirstTry   time.Time `json:"firstTry"`   // 首次尝试时间
}

// Consumer 消费者接口
type Consumer interface {
    Start()
    Stop()
    IsRunning() bool
}

// ReliableConsumer 可靠消息消费者
type ReliableConsumer struct {
    conn      *amqp.Connection
    channel   *amqp.Channel
    queueName string
    orderRepo repo.OrderRepo
    rdb       *redis.Client
    config    config.MQConsumerConfig

    // 运行时状态
    stopCh    chan struct{}
    doneCh    chan struct{}
    isRunning bool
}

// NewReliableConsumer 创建可靠消费者
func NewReliableConsumer(
    cfg config.Config,
    orderRepo repo.OrderRepo,
    rdb *redis.Client,
) (Consumer, error) {
    // 1. 连接 RabbitMQ
    conn, err := amqp.Dial(cfg.RabbitMQ.Url)
    if err != nil {
        return nil, fmt.Errorf("连接 RabbitMQ 失败: %w", err)
    }

    // 2. 创建 Channel
    ch, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("创建 Channel 失败: %w", err)
    }

    // 3. 设置 QoS（预取控制）
    // prefetchCount: 每次预取的消息数量
    // prefetchSize: 预取的消息大小（0表示不限制）
    // global: false 表示每个消费者独立预取
    prefetchCount := cfg.MQConsumer.PrefetchCount
    if prefetchCount <= 0 {
        prefetchCount = 10
    }
    err = ch.Qos(prefetchCount, 0, false)
    if err != nil {
        ch.Close()
        conn.Close()
        return nil, fmt.Errorf("设置 QoS 失败: %w", err)
    }

    // 4. 声明主队列
    _, err = ch.QueueDeclare(
        cfg.RabbitMQ.QueueName,
        true,  // durable: 持久化
        false, // delete when unused
        false, // exclusive
        false, // no-wait
        nil,   // arguments
    )
    if err != nil {
        ch.Close()
        conn.Close()
        return nil, fmt.Errorf("声明主队列失败: %w", err)
    }

    // 5. 如果启用死信队列，声明 DLQ 相关
    if cfg.MQConsumer.EnableDLQ {
        if err := declareDLQ(ch, cfg.MQConsumer); err != nil {
            ch.Close()
            conn.Close()
            return nil, fmt.Errorf("声明死信队列失败: %w", err)
        }
    }

    return &ReliableConsumer{
        conn:      conn,
        channel:   ch,
        queueName: cfg.RabbitMQ.QueueName,
        orderRepo: orderRepo,
        rdb:       rdb,
        config:    cfg.MQConsumer,
        stopCh:    make(chan struct{}),
        doneCh:    make(chan struct{}),
        isRunning: false,
    }, nil
}

// declareDLQ 声明死信队列
func declareDLQ(ch *amqp.Channel, cfg config.MQConsumerConfig) error {
    // 声明死信交换机
    err := ch.ExchangeDeclare(
        cfg.DLQExchangeName,
        "direct",
        true,  // durable
        false, // auto-deleted
        false, // internal
        false, // no-wait
        nil,
    )
    if err != nil {
        return err
    }

    // 声明死信队列
    _, err = ch.QueueDeclare(
        cfg.DLQQueueName,
        true,  // durable
        false, // delete when unused
        false, // exclusive
        false, // no-wait
        nil,
    )
    if err != nil {
        return err
    }

    // 绑定死信队列到死信交换机
    err = ch.QueueBind(
        cfg.DLQQueueName,
        "dlq", // routing key
        cfg.DLQExchangeName,
        false,
        nil,
    )
    return err
}

// Start 启动消费者
func (c *ReliableConsumer) Start() {
    if c.isRunning {
        log.Println("⚠️ 消费者已在运行，忽略重复启动")
        return
    }
    c.isRunning = true

    // 启动消费协程
    go c.consume()
    log.Printf("🚀 [可靠消费者] 已启动，队列: %s，预取数: %d，最大重试: %d",
        c.queueName, c.config.PrefetchCount, c.config.MaxRetryTimes)
}

// Stop 停止消费者（优雅退出）
func (c *ReliableConsumer) Stop() {
    if !c.isRunning {
        return
    }

    log.Println("🛑 收到停止信号，正在优雅关闭消费者...")

    // 1. 发送停止信号
    close(c.stopCh)

    // 2. 等待正在处理的消息完成
    select {
    case <-c.doneCh:
        log.Println("✅ 所有正在处理的消息已完成")
    case <-time.After(time.Duration(c.config.ShutdownTimeout) * time.Second):
        log.Printf("⚠️ 优雅关闭超时（%d秒），强制退出", c.config.ShutdownTimeout)
    }

    // 3. 关闭连接
    if c.channel != nil {
        c.channel.Close()
    }
    if c.conn != nil {
        c.conn.Close()
    }

    c.isRunning = false
    log.Println("👋 消费者已完全停止")
}

// IsRunning 检查消费者是否在运行
func (c *ReliableConsumer) IsRunning() bool {
    return c.isRunning
}

// consume 消费消息
func (c *ReliableConsumer) consume() {
    defer close(c.doneCh)

    msgs, err := c.channel.Consume(
        c.queueName,
        c.config.ConsumerTag, // consumer tag
        false,                // auto-ack（必须为false，启用手动ACK）
        false,                // exclusive
        false,                // no-local
        false,                // no-wait
        nil,                  // args
    )
    if err != nil {
        log.Printf("❌ 注册消费者失败: %v", err)
        return
    }

    workerNum := 20
    log.Printf(" [*] 启动 %d 个 Worker 处理消息", workerNum)

    for i := 0; i < workerNum; i++ {
        go c.worker(i, msgs)
    }

    // 阻塞直到收到停止信号
    <-c.stopCh

    // 停止时，不再接受新消息
    log.Println(" [*] 停止接收新消息，等待处理中...")
}

// worker 消息处理Worker
func (c *ReliableConsumer) worker(workerID int, msgs <-chan amqp.Delivery) {
    log.Printf("⚡ [Worker-%d] 已就绪", workerID)

    for {
        select {
        case <-c.stopCh:
            log.Printf("🛑 [Worker-%d] 收到停止信号，退出", workerID)
            return
        case d, ok := <-msgs:
            if !ok {
                log.Printf("⚠️ [Worker-%d] 消息通道已关闭", workerID)
                return
            }
            c.processMessage(workerID, d)
        }
    }
}

// processMessage 处理单条消息
func (c *ReliableConsumer) processMessage(workerID int, d amqp.Delivery) {
    // 1. 解析消息元数据
    var msg OrderMsg
    metadata := c.extractMetadata(d)

    // 2. 解析消息体
    if err := json.Unmarshal(d.Body, &msg); err != nil {
        log.Printf("❌ [Worker-%d] 消息解析失败: %v，MessageID=%s，Body=%s",
            workerID, err, metadata.MessageID, string(d.Body))
        // 解析失败的消息直接丢弃
        d.Ack(false)
        return
    }

    log.Printf("📨 [Worker-%d] 收到消息: OrderNo=%s，MessageID=%s，重试次数=%d",
        workerID, msg.OrderNo, metadata.MessageID, metadata.RetryCount)

    // 3. 获取当前重试次数
    retryCount := c.getRetryCount(d)
    maxRetries := c.config.MaxRetryTimes
    if maxRetries <= 0 {
        maxRetries = 3
    }

    // 4. 执行业务处理
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 4.1 Redis 幂等检查
    idempotentKey := fmt.Sprintf("idempotent:order:%s", msg.OrderNo)
    acquired, err := c.rdb.SetNX(ctx, idempotentKey, metadata.MessageID, 24*time.Hour).Result()
    if err != nil {
        log.Printf("⚠️ [Worker-%d] Redis 连接异常，暂缓消费: %v", workerID, err)
        d.Nack(false, true) // 重回队列，稍后重试
        return
    }

    if !acquired {
        // 幂等命中，说明已处理过
        log.Printf("♻️ [Worker-%d] 幂等拦截，重复消息: OrderNo=%s", workerID, msg.OrderNo)
        d.Ack(false)
        return
    }

    // 4.2 执行落库
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

    // 5. 处理结果
    if err != nil {
        log.Printf("❌ [Worker-%d] 落库失败: OrderNo=%s，Err=%v，重试次数=%d/%d",
            workerID, msg.OrderNo, err, retryCount, maxRetries)

        // 删除幂等锁，允许重试
        c.rdb.Del(ctx, idempotentKey)

        if retryCount >= maxRetries {
            // 超过最大重试次数，发送到死信队列
            c.handleDeadLetter(workerID, d, msg, err, metadata)
        } else {
            // 还在重试次数内，重回队列
            d.Nack(false, true)
        }
        return
    }

    // 6. 成功，发送 ACK
    log.Printf("✅ [Worker-%d] 落库成功: OrderNo=%s，MessageID=%s",
        workerID, msg.OrderNo, metadata.MessageID)
    d.Ack(false)
}

// handleDeadLetter 处理死信
func (c *ReliableConsumer) handleDeadLetter(workerID int, d amqp.Delivery, msg OrderMsg, err error, metadata MessageMetadata) {
    if !c.config.EnableDLQ {
        log.Printf("⚠️ [Worker-%d] DLQ未启用，消息将被丢弃: OrderNo=%s", workerID, msg.OrderNo)
        d.Ack(false) // 不启用DLQ时，直接丢弃
        return
    }

    // 构建死信消息体
    deadLetter := map[string]interface{}{
        "originalMessage":  msg,
        "metadata":        metadata,
        "error":           err.Error(),
        "failedAt":        time.Now().Format(time.RFC3339),
        "retryCount":      metadata.RetryCount,
    }

    body, _ := json.Marshal(deadLetter)

    // 发布到死信交换机
    err = c.channel.PublishWithContext(context.Background(),
        c.config.DLQExchangeName,
        "dlq", // routing key
        false, // mandatory
        false, // immediate
        amqp.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp.Persistent,
            Body:         body,
            Headers: amqp.Table{
                "x-first-death-reason": "rejected",
                "x-death-time":         time.Now().Unix(),
            },
        })

    if err != nil {
        log.Printf("❌ [Worker-%d] 发送死信失败: OrderNo=%s，Err=%v，消息将被丢弃",
            workerID, msg.OrderNo, err)
    } else {
        log.Printf("☠️ [Worker-%d] 消息已发送到死信队列: OrderNo=%s，DLQ=%s",
            workerID, msg.OrderNo, c.config.DLQQueueName)
    }

    // 确认原消息
    d.Ack(false)
}

// extractMetadata 从消息头提取元数据
func (c *ReliableConsumer) extractMetadata(d amqp.Delivery) MessageMetadata {
    metadata := MessageMetadata{
        MessageID: uuid.New().String(),
        TraceID:   uuid.New().String(),
        RetryCount: 0,
        FirstTry:   time.Now(),
    }

    // 从 headers 中提取 MessageID
    if d.Headers != nil {
        if msgID, ok := d.Headers["x-message-id"].(string); ok {
            metadata.MessageID = msgID
        }
        if retryCount, ok := d.Headers["x-retry-count"].(int32); ok {
            metadata.RetryCount = int(retryCount)
        }
    }

    // 如果没有 MessageID，使用 DeliveryTag
    if metadata.MessageID == "" {
        metadata.MessageID = fmt.Sprintf("%d", d.DeliveryTag)
    }

    return metadata
}

// getRetryCount 获取消息重试次数
func (c *ReliableConsumer) getRetryCount(d amqp.Delivery) int {
    if d.Headers == nil {
        return 0
    }

    // 从 x-death 头中获取重试次数
    if xDeath, ok := d.Headers["x-death"].([]interface{}); ok && len(xDeath) > 0 {
        for _, death := range xDeath {
            if deathMap, ok := death.(amqp.Table); ok {
                if count, ok := deathMap["count"].(int64); ok {
                    return int(count)
                }
            }
        }
    }

    // 从自定义头获取
    if retryCount, ok := d.Headers["x-retry-count"].(int32); ok {
        return int(retryCount)
    }

    return 0
}
