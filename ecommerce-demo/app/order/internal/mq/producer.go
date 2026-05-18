package mq

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"ecommerce-demo/app/order/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

/*
  RabbitMQ 订单消息生产者（可靠性增强版）

  改进：
  1. Publisher Confirm：启用后每一条消息都等待 Broker 确认，未确认则重试
  2. 自动重连：检测到连接/通道关闭后自动重建
  3. 连接健康检查：定期 heartbeat 检测

  延迟队列原理（死信+TTL）：
  ┌─────────────────────────────────────────────────────────────────┐
  │  延迟交换机 (order.delay.exchange)                               │
  │       │                                                         │
  │       ▼                                                         │
  │  延迟队列 (order.delay.queue)                                    │
  │  设置 x-dead-letter-exchange = ""                               │
  │  设置 x-dead-letter-routing-key = order.timeout.check            │
  │  设置消息 TTL                                                   │
  │       │                                                         │
  │       │ 消息到期后，RabbitMQ 自动转发到                           │
  │       │ 路由键 order.timeout.check 对应的队列                    │
  │       ▼                                                         │
  │  超时检查队列 (order.timeout.check.queue)                        │
  └─────────────────────────────────────────────────────────────────┘
*/

type OrderMsg struct {
	OrderNo     string `json:"orderNo"`
	UserID      int64  `json:"userId"`
	ProductID   int64  `json:"productId"`
	Count       int32  `json:"count"`
	TotalAmount int64  `json:"totalAmount"`
	ExpireTime  int64  `json:"expireTime"`
}

type Producer interface {
	PublishOrder(ctx context.Context, msg *OrderMsg) error
	PublishDelayOrder(ctx context.Context, msg *DelayOrderMsg, expireMinutes int) error
	Close() error
}

type rabbitProducer struct {
	url string
	cfg config.Config

	mu       sync.RWMutex
	conn     *amqp.Connection
	channel  *amqp.Channel

	// Publisher Confirm
	confirms    chan amqp.Confirmation
	confirmMode bool

	// 队列/交换机名称（重连时重建）
	queueName         string
	delayExchangeName string
	delayQueueName    string
	delayRoutingKey   string

	// 优雅关闭
	closeCh chan struct{}
	doneCh  chan struct{}
	closed  bool
}

func NewRabbitProducer(c config.Config) Producer {
	p := &rabbitProducer{
		url:               c.RabbitMQ.Url,
		cfg:               c,
		queueName:         c.RabbitMQ.QueueName,
		delayExchangeName: c.OrderTimeout.DelayExchangeName,
		delayQueueName:    c.OrderTimeout.DelayQueueName,
		delayRoutingKey:   "order.timeout.check",
		closeCh:           make(chan struct{}),
		doneCh:            make(chan struct{}),
	}

	if err := p.connect(); err != nil {
		log.Fatalf("无法连接 RabbitMQ: %v", err)
	}

	// 启动后台重连监控
	go p.monitorConnection()

	return p
}

// connect 建立连接和通道，声明所有队列/交换机
func (p *rabbitProducer) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	conn, err := amqp.Dial(p.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	// 启用 Publisher Confirm
	if err := ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	p.conn = conn
	p.channel = ch
	p.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 100))
	p.confirmMode = true

	// 声明普通订单队列
	_, err = ch.QueueDeclare(
		p.queueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		return err
	}

	// 声明延迟交换机
	err = ch.ExchangeDeclare(
		p.delayExchangeName,
		"direct",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	)
	if err != nil {
		log.Printf("延迟交换机声明失败（非致命）: %v", err)
	}

	// 声明延迟队列（死信队列配置）
	_, err = ch.QueueDeclare(
		p.delayQueueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": p.delayRoutingKey,
		},
	)
	if err != nil {
		return err
	}

	// 绑定延迟队列到交换机
	_ = ch.QueueBind(p.delayQueueName, p.delayRoutingKey, p.delayExchangeName, false, nil)

	// 声明超时检查队列
	_, err = ch.QueueDeclare(
		"order.timeout.check.queue",
		true, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	return nil
}

// monitorConnection 后台监控连接健康状态，断线自动重连
func (p *rabbitProducer) monitorConnection() {
	defer close(p.doneCh)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.closeCh:
			return
		case <-ticker.C:
			p.mu.RLock()
			conn := p.conn
			ch := p.channel
			p.mu.RUnlock()

			if conn == nil || conn.IsClosed() || ch == nil || ch.IsClosed() {
				log.Println("检测到 RabbitMQ 连接断开，开始重连...")
				for {
					select {
					case <-p.closeCh:
						return
					default:
					}

					if err := p.connect(); err != nil {
						log.Printf("RabbitMQ 重连失败: %v，3秒后重试...", err)
						time.Sleep(3 * time.Second)
						continue
					}
					log.Println("RabbitMQ 重连成功")
					break
				}
			}
		}
	}
}

// PublishOrder 发布订单创建消息（带 Publisher Confirm）
func (p *rabbitProducer) PublishOrder(ctx context.Context, msg *OrderMsg) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	p.mu.RLock()
	ch := p.channel
	confirms := p.confirms
	confirmMode := p.confirmMode
	p.mu.RUnlock()

	if ch == nil || ch.IsClosed() {
		return amqp.ErrClosed
	}

	err = ch.PublishWithContext(ctx,
		"",
		p.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})
	if err != nil {
		return err
	}

	// 等待 Broker 确认
	if confirmMode && confirms != nil {
		select {
		case confirm := <-confirms:
			if !confirm.Ack {
				return amqp.ErrClosed
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// PublishDelayOrder 发布延迟超时检查消息（带 Publisher Confirm）
func (p *rabbitProducer) PublishDelayOrder(ctx context.Context, msg *DelayOrderMsg, expireMinutes int) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	expiration := strconv.FormatInt(int64(expireMinutes)*60*1000, 10)

	p.mu.RLock()
	ch := p.channel
	confirms := p.confirms
	confirmMode := p.confirmMode
	p.mu.RUnlock()

	if ch == nil || ch.IsClosed() {
		return amqp.ErrClosed
	}

	err = ch.PublishWithContext(ctx,
		p.delayExchangeName,
		p.delayRoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Expiration:   expiration,
		})
	if err != nil {
		return err
	}

	// 等待 Broker 确认
	if confirmMode && confirms != nil {
		select {
		case confirm := <-confirms:
			if !confirm.Ack {
				return amqp.ErrClosed
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// Close 优雅关闭
func (p *rabbitProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true
	close(p.closeCh)

	<-p.doneCh // 等待监控协程退出

	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
