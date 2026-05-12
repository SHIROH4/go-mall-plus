package mq

import (
    "context"
    "encoding/json"
    "log"
    "strconv"

    "ecommerce-demo/app/order/internal/config"

    amqp "github.com/rabbitmq/amqp091-go"
)

/*
  RabbitMQ 订单消息生产者

  职责：
  1. PublishOrder - 发布普通订单消息（异步落库）
  2. PublishDelayOrder - 发布延迟订单消息（超时检查）

  延迟队列实现原理（死信队列+TTL）：
  ┌─────────────────────────────────────────────────────────────────┐
  │                                                                 │
  │  延迟交换机                                                     │
  │  (order.delay.exchange)                                         │
  │       │                                                         │
  │       │ x-delayed-message 路由                                  │
  │       ▼                                                         │
  │  延迟队列 (order.delay.queue)                                   │
  │  设置 x-dead-letter-exchange = ""                               │
  │  设置 x-dead-letter-routing-key = order.timeout.check           │
  │  设置消息 TTL = 订单超时时间（毫秒）                            │
  │       │                                                         │
  │       │ 消息到期后，RabbitMQ 自动将消息转发到                     │
  │       │ 路由键 order.timeout.check 对应的队列                    │
  │       ▼                                                         │
  │  真正的超时检查消费者                                            │
  │                                                                 │
  └─────────────────────────────────────────────────────────────────┘
*/

// OrderMsg 投递到 MQ 的异步订单消息体
type OrderMsg struct {
    OrderNo    string `json:"orderNo"`
    UserID     int64  `json:"userId"`
    ProductID  int64  `json:"productId"`
    Count      int32  `json:"count"`
    TotalAmount int64 `json:"totalAmount"`
    // [新增] 过期时间（Unix时间戳）
    ExpireTime int64 `json:"expireTime"`
}

// Producer 生产者接口
type Producer interface {
    // PublishOrder 发布普通订单消息（异步落库）
    PublishOrder(ctx context.Context, msg *OrderMsg) error
    // PublishDelayOrder 发布延迟订单消息（超时检查）
    PublishDelayOrder(ctx context.Context, msg *DelayOrderMsg, expireMinutes int) error
    // Close 关闭连接
    Close() error
}

type rabbitProducer struct {
    conn              *amqp.Connection
    channel           *amqp.Channel
    queueName         string
    delayExchangeName string
    delayQueueName    string
    delayRoutingKey   string
}

func NewRabbitProducer(c config.Config) Producer {
    // 1. 连接 RabbitMQ
    conn, err := amqp.Dial(c.RabbitMQ.Url)
    if err != nil {
        log.Fatalf("无法连接 RabbitMQ: %v", err)
    }

    // 2. 创建 Channel
    ch, err := conn.Channel()
    if err != nil {
        log.Fatalf("无法创建 RabbitMQ Channel: %v", err)
    }

    // 3. 声明普通订单队列（持久化）
    _, err = ch.QueueDeclare(
        c.RabbitMQ.QueueName, // 队列名
        true,                 // durable: 重启不丢失
        false,                // delete when unused
        false,                // exclusive
        false,                // no-wait
        nil,                  // arguments
    )
    if err != nil {
        log.Fatalf("无法声明普通 Queue: %v", err)
    }

    // 4. 声明延迟交换机（使用 direct 类型）
    delayExchangeName := c.OrderTimeout.DelayExchangeName
    delayQueueName := c.OrderTimeout.DelayQueueName
    delayRoutingKey := "order.timeout.check"

    // 尝试声明延迟交换机（先用普通 direct 交换机，TTL 由消息属性控制）
    err = ch.ExchangeDeclare(
        delayExchangeName, // name
        "direct",          // type
        true,              // durable
        false,             // auto-deleted
        false,             // internal
        false,             // no-wait
        nil,               // arguments
    )
    if err != nil {
        log.Printf("⚠️ 延迟交换机声明失败（非致命，将使用备选方案）: %v", err)
    }

    // 5. 声明延迟队列（死信队列）
    // x-dead-letter-exchange 和 x-dead-letter-routing-key 用于消息过期后转发
    _, err = ch.QueueDeclare(
        delayQueueName,
        true, // durable
        false, // delete when unused
        false, // exclusive
        false, // no-wait
        amqp.Table{
            // 消息过期后转发的交换机
            "x-dead-letter-exchange":    "",
            // 消息过期后转发的路由键
            "x-dead-letter-routing-key": delayRoutingKey,
        },
    )
    if err != nil {
        log.Fatalf("无法声明延迟队列: %v", err)
    }

    // 6. 绑定延迟队列到交换机
    err = ch.QueueBind(
        delayQueueName,
        delayRoutingKey,
        delayExchangeName,
        false,
        nil,
    )
    if err != nil {
        log.Fatalf("无法绑定延迟队列: %v", err)
    }

    // 7. 声明真正接收超时消息的队列
    timeoutQueueName := "order.timeout.check.queue"
    _, err = ch.QueueDeclare(
        timeoutQueueName,
        true,  // durable
        false, // delete when unused
        false, // exclusive
        false, // no-wait
        nil,   // arguments
    )
    if err != nil {
        log.Fatalf("无法声明超时检查队列: %v", err)
    }

    return &rabbitProducer{
        conn:              conn,
        channel:           ch,
        queueName:         c.RabbitMQ.QueueName,
        delayExchangeName: delayExchangeName,
        delayQueueName:    delayQueueName,
        delayRoutingKey:   delayRoutingKey,
    }
}

func (p *rabbitProducer) PublishOrder(ctx context.Context, msg *OrderMsg) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    return p.channel.PublishWithContext(ctx,
        "",          // exchange（默认交换机）
        p.queueName, // routing key
        false,       // mandatory
        false,       // immediate
        amqp.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp.Persistent, // 消息持久化
            Body:         body,
        })
}

func (p *rabbitProducer) PublishDelayOrder(ctx context.Context, msg *DelayOrderMsg, expireMinutes int) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    // 将过期时间转换为毫秒
    expiration := strconv.FormatInt(int64(expireMinutes)*60*1000, 10)

    return p.channel.PublishWithContext(ctx,
        p.delayExchangeName, // 交换机
        p.delayQueueName,    // 路由到延迟队列
        false,              // mandatory
        false,              // immediate
        amqp.Publishing{
            ContentType:  "application/json",
            DeliveryMode: amqp.Persistent, // 消息持久化
            Body:         body,
            // 关键：设置消息过期时间（毫秒）
            // RabbitMQ 会在消息过期后，将其转发到 x-dead-letter-exchange
            // 本例中转发到空交换机 + order.timeout.check 路由键
            Expiration: expiration,
        })
}

func (p *rabbitProducer) Close() error {
    if p.channel != nil {
        p.channel.Close()
    }
    if p.conn != nil {
        return p.conn.Close()
    }
    return nil
}
