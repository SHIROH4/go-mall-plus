package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/repo"
	"ecommerce-demo/common/metrics"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

/*
  延迟订单消费者（可靠性增强版）

  职责：
  监听延迟队列，接收超时订单消息，检查并处理超时订单

  可靠性增强：
  1. 手动ACK确认 - 确保消息处理后才确认
  2. 重试次数限制 - 防止无限重试
  3. 死信队列(DLQ) - 永久失败消息存储
  4. 优雅退出 - 服务停止时处理完正在处理的消息
  5. 消息追踪 - MessageID用于问题排查
  6. 幂等处理 - 即使收到重复消息，也只处理一次
*/

// DelayMessageMetadata 延迟消息元数据
type DelayMessageMetadata struct {
	MessageID  string    `json:"messageId"`
	TraceID    string    `json:"traceId"`
	RetryCount int       `json:"retryCount"`
	FirstTry   time.Time `json:"firstTry"`
}

// DelayConsumer 延迟消费者接口
type DelayConsumer interface {
	Start()
	Stop()
	IsRunning() bool
}

// ReliableDelayConsumer 可靠延迟消费者
type ReliableDelayConsumer struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string // order.timeout.check.queue
	orderRepo repo.OrderRepo
	rdb       *redis.ClusterClient
	config    config.MQConsumerConfig
	stopCh    chan struct{}
	doneCh    chan struct{}
	isRunning bool
}

// NewReliableDelayConsumer 创建可靠延迟消费者
func NewReliableDelayConsumer(
	cfg config.Config,
	orderRepo repo.OrderRepo,
	rdb *redis.ClusterClient,
) (DelayConsumer, error) {
	conn, err := amqp.Dial(cfg.RabbitMQ.Url)
	if err != nil {
		return nil, fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("创建 Channel 失败: %w", err)
	}

	// 设置 QoS
	prefetchCount := cfg.MQConsumer.PrefetchCount
	if prefetchCount <= 0 {
		prefetchCount = 5 // 延迟消费者预取数可以小一些
	}
	err = ch.Qos(prefetchCount, 0, false)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("设置 QoS 失败: %w", err)
	}

	// 声明超时检查队列
	queueName := "order.timeout.check.queue"
	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("声明超时检查队列失败: %w", err)
	}

	// 声明死信队列
	if cfg.MQConsumer.EnableDLQ {
		if err := declareDelayDLQ(ch, cfg.MQConsumer); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("声明延迟死信队列失败: %w", err)
		}
	}

	return &ReliableDelayConsumer{
		conn:      conn,
		channel:   ch,
		queueName: queueName,
		orderRepo: orderRepo,
		rdb:       rdb,
		config:    cfg.MQConsumer,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
		isRunning: false,
	}, nil
}

// declareDelayDLQ 声明延迟队列的DLQ
func declareDelayDLQ(ch *amqp.Channel, cfg config.MQConsumerConfig) error {
	// 延迟队列专用死信交换机
	dlxName := cfg.DLQExchangeName + ".delay"

	err := ch.ExchangeDeclare(
		dlxName,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	dlqName := cfg.DLQQueueName + ".delay"
	_, err = ch.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		dlqName,
		"delay.dlq",
		dlxName,
		false,
		nil,
	)
	return err
}

// Start 启动消费者
func (c *ReliableDelayConsumer) Start() {
	if c.isRunning {
		log.Println("⚠️ 延迟消费者已在运行，忽略重复启动")
		return
	}
	c.isRunning = true

	go c.consume()
	log.Printf("🚀 [可靠延迟消费者] 已启动，队列: %s，预取数: %d，最大重试: %d",
		c.queueName, c.config.PrefetchCount, c.config.MaxRetryTimes)
}

// Stop 停止消费者
func (c *ReliableDelayConsumer) Stop() {
	if !c.isRunning {
		return
	}

	log.Println("🛑 收到停止信号，正在优雅关闭延迟消费者...")

	close(c.stopCh)

	select {
	case <-c.doneCh:
		log.Println("✅ 所有正在处理的延迟消息已完成")
	case <-time.After(time.Duration(c.config.ShutdownTimeout) * time.Second):
		log.Printf("⚠️ 延迟消费者优雅关闭超时（%d秒）", c.config.ShutdownTimeout)
	}

	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}

	c.isRunning = false
	log.Println("👋 延迟消费者已完全停止")
}

// IsRunning 检查是否在运行
func (c *ReliableDelayConsumer) IsRunning() bool {
	return c.isRunning
}

func (c *ReliableDelayConsumer) consume() {
	defer close(c.doneCh)

	msgs, err := c.channel.Consume(
		c.queueName,
		c.config.ConsumerTag+".delay",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("❌ 注册延迟消费者失败: %v", err)
		return
	}

	workerNum := 5 // 延迟消费者Worker数可以少一些
	log.Printf(" [*] 启动 %d 个延迟 Worker 处理消息", workerNum)

	for i := 0; i < workerNum; i++ {
		go c.worker(i, msgs)
	}

	<-c.stopCh
	log.Println(" [*] 延迟消费者停止接收新消息...")
}

func (c *ReliableDelayConsumer) worker(workerID int, msgs <-chan amqp.Delivery) {
	log.Printf("⚡ [延迟Worker-%d] 已就绪", workerID)

	for {
		select {
		case <-c.stopCh:
			log.Printf("🛑 [延迟Worker-%d] 收到停止信号，退出", workerID)
			return
		case d, ok := <-msgs:
			if !ok {
				log.Printf("⚠️ [延迟Worker-%d] 消息通道已关闭", workerID)
				return
			}
			c.processMessage(workerID, d)
		}
	}
}

func (c *ReliableDelayConsumer) processMessage(workerID int, d amqp.Delivery) {
	// 1. 提取元数据
	metadata := c.extractMetadata(d)

	// 2. 解析消息
	var msg DelayOrderMsg
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		log.Printf("❌ [延迟Worker-%d] 消息解析失败: %v，MessageID=%s",
			workerID, err, metadata.MessageID)
		d.Ack(false) // 解析失败直接丢弃
		return
	}

	log.Printf("📨 [延迟Worker-%d] 收到超时检查消息: OrderNo=%s，MessageID=%s，重试次数=%d",
		workerID, msg.OrderNo, metadata.MessageID, metadata.RetryCount)

	// 3. 获取重试次数
	retryCount := c.getRetryCount(d)
	maxRetries := c.config.MaxRetryTimes
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// 4. 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 5. 查询订单状态
	order, err := c.orderRepo.GetOrderByNo(ctx, msg.OrderNo)
	if err != nil {
		if err == repo.ErrOrderNotFound {
			log.Printf("ℹ️ [延迟Worker-%d] 订单不存在，跳过: OrderNo=%s", workerID, msg.OrderNo)
			d.Ack(false)
			return
		}

		// 数据库查询失败，判断是否重试
		log.Printf("⚠️ [延迟Worker-%d] 查询订单失败: OrderNo=%s，Err=%v，重试次数=%d/%d",
			workerID, msg.OrderNo, err, retryCount, maxRetries)

		if retryCount >= maxRetries {
			metrics.MQConsumeTotal.WithLabelValues("order.delay.check", "dql").Inc()
			c.handleDeadLetter(workerID, d, msg, err, metadata, "delay.dlq")
		} else {
			metrics.MQConsumeTotal.WithLabelValues("order.delay.check", "fail").Inc()
			d.Nack(false, true)
		}
		return
	}

	// 6. 幂等检查：只有待支付订单才处理
	if order.Status != 0 {
		log.Printf("ℹ️ [延迟Worker-%d] 订单状态非待支付，跳过: OrderNo=%s，Status=%d",
			workerID, msg.OrderNo, order.Status)
		d.Ack(false)
		return
	}

	// 7. 执行超时取消
	err = c.orderRepo.TimeoutOrderTx(ctx, msg.OrderNo, msg.ProductID, msg.Count)
	if err != nil {
		log.Printf("❌ [延迟Worker-%d] 超时取消失败: OrderNo=%s，Err=%v，重试次数=%d/%d",
			workerID, msg.OrderNo, err, retryCount, maxRetries)

		if retryCount >= maxRetries {
			metrics.MQConsumeTotal.WithLabelValues("order.delay.check", "dql").Inc()
			c.handleDeadLetter(workerID, d, msg, err, metadata, "delay.dlq")
		} else {
			metrics.MQConsumeTotal.WithLabelValues("order.delay.check", "fail").Inc()
			d.Nack(false, true)
		}
		return
	}

	log.Printf("✅ [延迟Worker-%d] 超时订单处理成功: OrderNo=%s，MessageID=%s",
		workerID, msg.OrderNo, metadata.MessageID)
	metrics.MQConsumeTotal.WithLabelValues("order.delay.check", "success").Inc()
	metrics.OrderTimeoutTotal.Inc()
	d.Ack(false)
}

// handleDeadLetter 处理死信
func (c *ReliableDelayConsumer) handleDeadLetter(workerID int, d amqp.Delivery, msg DelayOrderMsg, err error, metadata DelayMessageMetadata, routingKey string) {
	if !c.config.EnableDLQ {
		log.Printf("⚠️ [延迟Worker-%d] DLQ未启用，消息将被丢弃: OrderNo=%s", workerID, msg.OrderNo)
		d.Ack(false)
		return
	}

	dlxName := c.config.DLQExchangeName + ".delay"
	dlqName := c.config.DLQQueueName + ".delay"

	deadLetter := map[string]interface{}{
		"originalMessage": msg,
		"metadata":        metadata,
		"error":           err.Error(),
		"failedAt":        time.Now().Format(time.RFC3339),
		"retryCount":      metadata.RetryCount,
		"source":          "delay-consumer",
	}

	body, _ := json.Marshal(deadLetter)

	err = c.channel.PublishWithContext(context.Background(),
		dlxName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Headers: amqp.Table{
				"x-first-death-reason": "rejected",
				"x-death-time":         time.Now().Unix(),
				"x-original-queue":     c.queueName,
			},
		})

	if err != nil {
		log.Printf("❌ [延迟Worker-%d] 发送死信失败: OrderNo=%s，Err=%v",
			workerID, msg.OrderNo, err)
	} else {
		log.Printf("☠️ [延迟Worker-%d] 消息已发送至死信队列: OrderNo=%s，DLQ=%s",
			workerID, msg.OrderNo, dlqName)
	}

	d.Ack(false)
}

// extractMetadata 提取元数据
func (c *ReliableDelayConsumer) extractMetadata(d amqp.Delivery) DelayMessageMetadata {
	metadata := DelayMessageMetadata{
		MessageID:  uuid.New().String(),
		TraceID:    uuid.New().String(),
		RetryCount: 0,
		FirstTry:   time.Now(),
	}

	if d.Headers != nil {
		if msgID, ok := d.Headers["x-message-id"].(string); ok {
			metadata.MessageID = msgID
		}
		if retryCount, ok := d.Headers["x-retry-count"].(int32); ok {
			metadata.RetryCount = int(retryCount)
		}
	}

	if metadata.MessageID == "" {
		metadata.MessageID = fmt.Sprintf("%d", d.DeliveryTag)
	}

	return metadata
}

// getRetryCount 获取重试次数
func (c *ReliableDelayConsumer) getRetryCount(d amqp.Delivery) int {
	if d.Headers == nil {
		return 0
	}

	if xDeath, ok := d.Headers["x-death"].([]interface{}); ok && len(xDeath) > 0 {
		for _, death := range xDeath {
			if deathMap, ok := death.(amqp.Table); ok {
				if count, ok := deathMap["count"].(int64); ok {
					return int(count)
				}
			}
		}
	}

	if retryCount, ok := d.Headers["x-retry-count"].(int32); ok {
		return int(retryCount)
	}

	return 0
}
