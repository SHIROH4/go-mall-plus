package mq

import (
	"context"
	"errors"
	"testing"
	"time"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/repo"

	"github.com/stretchr/testify/assert"
)

type fakeOrderLookup struct {
	order *repo.Order
	err   error
	calls int
}

func (f *fakeOrderLookup) GetOrderByNo(_ context.Context, _ string) (*repo.Order, error) {
	f.calls++
	return f.order, f.err
}

/*
  MQ消费者可靠性配置单元测试

  测试覆盖：
  1. 默认配置生成
  2. 配置字段验证
*/

func TestDefaultMQConsumerConfig(t *testing.T) {
	cfg := config.DefaultMQConsumerConfig()

	assert.Equal(t, 10, cfg.PrefetchCount)
	assert.Equal(t, 3, cfg.MaxRetryTimes)
	assert.Equal(t, 5, cfg.RetryDelay)
	assert.Equal(t, "order.retry.exchange", cfg.RetryExchangeName)
	assert.Equal(t, "order.retry.queue", cfg.RetryQueueName)
	assert.True(t, cfg.EnableDLQ)
	assert.Equal(t, "order.dlx", cfg.DLQExchangeName)
	assert.Equal(t, "order.dlq", cfg.DLQQueueName)
	assert.Equal(t, "order-consumer", cfg.ConsumerTag)
	assert.Equal(t, 30, cfg.ShutdownTimeout)
}

func TestMessageMetadata(t *testing.T) {
	metadata := MessageMetadata{
		MessageID:  "test-msg-id",
		TraceID:    "test-trace-id",
		RetryCount: 2,
		FirstTry:   time.Now(),
	}

	assert.Equal(t, "test-msg-id", metadata.MessageID)
	assert.Equal(t, "test-trace-id", metadata.TraceID)
	assert.Equal(t, 2, metadata.RetryCount)
	assert.False(t, metadata.FirstTry.IsZero())
}

func TestDelayMessageMetadata(t *testing.T) {
	metadata := MessageMetadata{
		MessageID:  "delay-msg-id",
		TraceID:    "delay-trace-id",
		RetryCount: 1,
		FirstTry:   time.Now(),
	}

	assert.Equal(t, "delay-msg-id", metadata.MessageID)
	assert.Equal(t, "delay-trace-id", metadata.TraceID)
	assert.Equal(t, 1, metadata.RetryCount)
}

func TestReliableConsumerConfig(t *testing.T) {
	// 测试自定义配置
	cfg := config.MQConsumerConfig{
		PrefetchCount:     20,
		MaxRetryTimes:     5,
		RetryDelay:        10,
		RetryExchangeName: "custom.retry.exchange",
		RetryQueueName:    "custom.retry.queue",
		EnableDLQ:         true,
		DLQExchangeName:   "custom.dlx",
		DLQQueueName:      "custom.dlq",
		ConsumerTag:       "custom-consumer",
		ShutdownTimeout:   60,
	}

	assert.Equal(t, 20, cfg.PrefetchCount)
	assert.Equal(t, 5, cfg.MaxRetryTimes)
	assert.Equal(t, 10, cfg.RetryDelay)
	assert.Equal(t, "custom.retry.exchange", cfg.RetryExchangeName)
	assert.Equal(t, "custom.retry.queue", cfg.RetryQueueName)
	assert.True(t, cfg.EnableDLQ)
	assert.Equal(t, "custom.dlx", cfg.DLQExchangeName)
	assert.Equal(t, "custom.dlq", cfg.DLQQueueName)
	assert.Equal(t, "custom-consumer", cfg.ConsumerTag)
	assert.Equal(t, 60, cfg.ShutdownTimeout)
}

func TestReliableConsumerConfigZeroValues(t *testing.T) {
	// 测试零值配置（应该使用默认值）
	cfg := config.MQConsumerConfig{}

	// 零值应该按预期处理
	assert.Equal(t, 0, cfg.PrefetchCount)
	assert.Equal(t, 0, cfg.MaxRetryTimes)
	assert.Equal(t, 0, cfg.RetryDelay)
	assert.False(t, cfg.EnableDLQ)
}

func TestVerifyOrderCreatedReadsPersistedOrderWithoutRecreatingIt(t *testing.T) {
	expireTime := time.Unix(1_800_000_000, 0)
	lookup := &fakeOrderLookup{order: &repo.Order{
		OrderNo:     "ORD1001",
		UserID:      7,
		ProductID:   11,
		Count:       2,
		TotalAmount: 19900,
		ExpireTime:  &expireTime,
	}}
	consumer := &ReliableConsumer{orderRepo: lookup}

	err := consumer.verifyOrderCreated(context.Background(), OrderMsg{
		OrderNo:     "ORD1001",
		UserID:      7,
		ProductID:   11,
		Count:       2,
		TotalAmount: 19900,
		ExpireTime:  expireTime.Unix(),
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, lookup.calls)
}

func TestVerifyOrderCreatedRejectsMismatchedEvent(t *testing.T) {
	lookup := &fakeOrderLookup{order: &repo.Order{
		OrderNo:     "ORD1002",
		UserID:      7,
		ProductID:   11,
		Count:       1,
		TotalAmount: 9900,
	}}
	consumer := &ReliableConsumer{orderRepo: lookup}

	err := consumer.verifyOrderCreated(context.Background(), OrderMsg{
		OrderNo:     "ORD1002",
		UserID:      7,
		ProductID:   11,
		Count:       2,
		TotalAmount: 19800,
	})

	assert.ErrorContains(t, err, "事件与订单数据不一致")
}

func TestVerifyOrderCreatedPropagatesLookupFailure(t *testing.T) {
	lookup := &fakeOrderLookup{err: errors.New("database unavailable")}
	consumer := &ReliableConsumer{orderRepo: lookup}

	err := consumer.verifyOrderCreated(context.Background(), OrderMsg{OrderNo: "ORD1003"})

	assert.ErrorContains(t, err, "database unavailable")
}
