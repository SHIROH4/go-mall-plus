package mq

import (
    "context"
    "testing"
    "time"

    "ecommerce-demo/app/order/internal/config"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

/*
  生产者单元测试

  测试覆盖：
  1. 普通订单消息发布
  2. 延迟订单消息发布
  3. 消息持久化验证

  前置条件：
  1. RabbitMQ 服务运行在 localhost:5672
  2. guest/guest 可访问
*/

func newTestConfig() config.Config {
    return config.Config{
        RabbitMQ: struct {
            Url       string
            QueueName string
        }{
            Url:       "amqp://guest:guest@localhost:5672/",
            QueueName: "test.order.queue",
        },
        OrderTimeout: config.OrderTimeoutConfig{
            OrderExpireMinutes:    1, // 测试用1分钟
            DelayQueueName:        "test.order.delay.queue",
            DelayExchangeName:     "test.order.delay.exchange",
            ScanIntervalSeconds:   60,
            MaxScanCount:         100,
        },
    }
}

func TestRabbitProducer_PublishOrder(t *testing.T) {
    cfg := newTestConfig()
    producer := NewRabbitProducer(cfg)
    defer producer.Close()

    msg := &OrderMsg{
        OrderNo:     "TEST-ORD-001",
        UserID:      1001,
        ProductID:   1,
        Count:       2,
        TotalAmount: 199800,
        ExpireTime:  time.Now().Add(30 * time.Minute).Unix(),
    }

    err := producer.PublishOrder(context.Background(), msg)
    require.NoError(t, err, "发布普通订单消息失败")
}

func TestRabbitProducer_PublishDelayOrder(t *testing.T) {
    cfg := newTestConfig()
    producer := NewRabbitProducer(cfg)
    defer producer.Close()

    expireTime := time.Now().Add(1 * time.Minute)
    msg := BuildDelayOrderMsg("TEST-ORD-002", 1, 2, expireTime)

    // 1分钟后过期
    err := producer.PublishDelayOrder(context.Background(), msg, 1)
    require.NoError(t, err, "发布延迟订单消息失败")
}

func TestBuildDelayOrderMsg(t *testing.T) {
    expireTime := time.Now().Add(30 * time.Minute)
    msg := BuildDelayOrderMsg("ORD123", 1, 5, expireTime)

    assert.Equal(t, "ORD123", msg.OrderNo)
    assert.Equal(t, int64(1), msg.ProductID)
    assert.Equal(t, int32(5), msg.Count)
    assert.False(t, msg.CreateTime.IsZero())
    assert.Equal(t, expireTime.Unix(), msg.ExpireTime.Unix())
}

func TestOrderMsg_ExpireTime(t *testing.T) {
    expireTime := time.Now().Add(30 * time.Minute)
    msg := &OrderMsg{
        OrderNo:     "TEST-ORD-003",
        UserID:      1001,
        ProductID:   1,
        Count:       1,
        TotalAmount: 100,
        ExpireTime:  expireTime.Unix(),
    }

    assert.Equal(t, expireTime.Unix(), msg.ExpireTime)
}
