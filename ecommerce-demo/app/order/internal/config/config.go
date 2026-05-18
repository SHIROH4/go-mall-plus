package config

import "github.com/zeromicro/go-zero/zrpc"

// OrderTimeoutConfig 订单超时自动取消配置
type OrderTimeoutConfig struct {
    // OrderExpireMinutes 订单超时时间（分钟），默认30分钟
    OrderExpireMinutes int
    // DelayQueueName 延迟队列名称
    DelayQueueName string
    // DelayExchangeName 延迟交换机名称
    DelayExchangeName string
    // ScanIntervalSeconds 定时扫描间隔（秒），默认60秒
    ScanIntervalSeconds int
    // MaxScanCount 每次最多扫描处理数量，防止锁表
    MaxScanCount int
}

// DefaultOrderTimeoutConfig 默认超时配置
func DefaultOrderTimeoutConfig() OrderTimeoutConfig {
    return OrderTimeoutConfig{
        OrderExpireMinutes:    30,
        DelayQueueName:        "order.delay.queue",
        DelayExchangeName:     "order.delay.exchange",
        ScanIntervalSeconds:   60,
        MaxScanCount:          100,
    }
}

// MQConsumerConfig MQ消费者可靠性配置
type MQConsumerConfig struct {
    // PrefetchCount 每次预取消息数量，默认10
    PrefetchCount int
    // MaxRetryTimes 最大重试次数，默认3次
    MaxRetryTimes int
    // RetryDelay 重试间隔（秒），默认5秒
    RetryDelay int
    // EnableDLQ 是否启用死信队列，默认true
    EnableDLQ bool
    // DLQExchangeName 死信交换机名称
    DLQExchangeName string
    // DLQQueueName 死信队列名称
    DLQQueueName string
    // ConsumerTag 消费者标签（用于区分不同消费者实例）
    ConsumerTag string
    // ShutdownTimeout 服务关闭时等待处理完成的最大时间（秒）
    ShutdownTimeout int
}

// DefaultMQConsumerConfig 默认MQ消费者配置
func DefaultMQConsumerConfig() MQConsumerConfig {
    return MQConsumerConfig{
        PrefetchCount:    10,
        MaxRetryTimes:    3,
        RetryDelay:       5,
        EnableDLQ:        true,
        DLQExchangeName:  "order.dlx",
        DLQQueueName:     "order.dlq",
        ConsumerTag:      "order-consumer",
        ShutdownTimeout:  30,
    }
}

type Config struct {
    zrpc.RpcServerConf
    DataSource string
    RedisConf  struct {
        Host string
        Type string
        Pass string
    }
    // 依赖 Product RPC
    ProductRpcConf zrpc.RpcClientConf
    // 依赖 Stock RPC（库存服务）
    StockRpcConf    zrpc.RpcClientConf
    // Outbox Worker 配置
    Outbox struct {
        PollIntervalSeconds int // 轮询间隔，默认 2 秒
        BatchSize           int // 每次拉取消息数，默认 50
    }
    RabbitMQ struct {
        Url       string
        QueueName string
    }
    // [新增] 订单超时配置
    OrderTimeout OrderTimeoutConfig
    // [新增] MQ消费者可靠性配置
    MQConsumer MQConsumerConfig
}
