package mq

import (
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestRetryExpirationUsesExponentialBackoffAndCap(t *testing.T) {
	assert.Equal(t, "5000", retryExpiration(5, 1))
	assert.Equal(t, "10000", retryExpiration(5, 2))
	assert.Equal(t, "20000", retryExpiration(5, 3))
	assert.Equal(t, "300000", retryExpiration(5, 20))
}

func TestDeliveryRetryCountSupportsRabbitMQNumericTypes(t *testing.T) {
	assert.Equal(t, 2, deliveryRetryCount(amqp.Delivery{Headers: amqp.Table{
		headerRetryCount: int32(2),
	}}))
	assert.Equal(t, 3, deliveryRetryCount(amqp.Delivery{Headers: amqp.Table{
		headerRetryCount: int64(3),
	}}))
}

func TestDeliveryMessageIDIsStableWithoutHeaders(t *testing.T) {
	delivery := amqp.Delivery{Body: []byte(`{"orderNo":"ORD100"}`)}

	first := deliveryMessageID(delivery, "")
	second := deliveryMessageID(delivery, "")

	assert.Equal(t, first, second)
	assert.Contains(t, first, "sha256:")
}

func TestRetryHeadersPreserveTraceAndIncrementMetadata(t *testing.T) {
	firstTry := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
	delivery := amqp.Delivery{Headers: amqp.Table{"custom": "value"}}

	headers := retryHeaders(delivery, "evt-1", "trace-1", 2, firstTry)

	assert.Equal(t, "value", headers["custom"])
	assert.Equal(t, "evt-1", headers[headerMessageID])
	assert.Equal(t, "trace-1", headers[headerTraceID])
	assert.Equal(t, int32(2), headers[headerRetryCount])
	assert.Equal(t, firstTry.UnixMilli(), headers[headerFirstTry])
}
