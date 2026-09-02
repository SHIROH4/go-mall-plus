package mq

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	headerMessageID  = "x-message-id"
	headerTraceID    = "x-trace-id"
	headerRetryCount = "x-retry-count"
	headerFirstTry   = "x-first-try"
)

type confirmedPublisher struct {
	channel  *amqp.Channel
	confirms chan amqp.Confirmation
	mu       sync.Mutex
}

func newConfirmedPublisher(channel *amqp.Channel) (*confirmedPublisher, error) {
	if err := channel.Confirm(false); err != nil {
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}
	return &confirmedPublisher{
		channel:  channel,
		confirms: channel.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}, nil
}

func (p *confirmedPublisher) publish(
	ctx context.Context,
	exchange string,
	routingKey string,
	message amqp.Publishing,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.channel.PublishWithContext(ctx, exchange, routingKey, false, false, message); err != nil {
		return err
	}

	select {
	case confirmation, ok := <-p.confirms:
		if !ok || !confirmation.Ack {
			return fmt.Errorf("rabbitmq did not confirm publish")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func declareRetryTopology(
	channel *amqp.Channel,
	exchangeName string,
	queueName string,
	routingKey string,
	targetQueue string,
) error {
	if err := channel.ExchangeDeclare(exchangeName, "direct", true, false, false, false, nil); err != nil {
		return err
	}

	if _, err := channel.QueueDeclare(queueName, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": targetQueue,
	}); err != nil {
		return err
	}

	return channel.QueueBind(queueName, routingKey, exchangeName, false, nil)
}

func retryExpiration(baseDelaySeconds, retryCount int) string {
	if baseDelaySeconds <= 0 {
		baseDelaySeconds = 5
	}
	if retryCount < 1 {
		retryCount = 1
	}

	delay := time.Duration(baseDelaySeconds) * time.Second
	for i := 1; i < retryCount; i++ {
		delay *= 2
		if delay >= 5*time.Minute {
			delay = 5 * time.Minute
			break
		}
	}
	return strconv.FormatInt(delay.Milliseconds(), 10)
}

func deliveryRetryCount(delivery amqp.Delivery) int {
	if count, ok := tableInt(delivery.Headers, headerRetryCount); ok {
		return count
	}

	if delivery.Headers != nil {
		if deaths, ok := delivery.Headers["x-death"].([]interface{}); ok {
			for _, death := range deaths {
				if table, ok := death.(amqp.Table); ok {
					if count, ok := tableInt(table, "count"); ok {
						return count
					}
				}
			}
		}
	}
	return 0
}

func deliveryMessageID(delivery amqp.Delivery, eventID string) string {
	if delivery.MessageId != "" {
		return delivery.MessageId
	}
	if delivery.Headers != nil {
		if messageID, ok := delivery.Headers[headerMessageID].(string); ok && messageID != "" {
			return messageID
		}
	}
	if eventID != "" {
		return eventID
	}

	digest := sha256.Sum256(delivery.Body)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func deliveryFirstTry(delivery amqp.Delivery) time.Time {
	if millis, ok := tableInt64(delivery.Headers, headerFirstTry); ok && millis > 0 {
		return time.UnixMilli(millis)
	}
	if !delivery.Timestamp.IsZero() {
		return delivery.Timestamp
	}
	return time.Now()
}

func retryHeaders(delivery amqp.Delivery, messageID, traceID string, retryCount int, firstTry time.Time) amqp.Table {
	headers := amqp.Table{}
	for key, value := range delivery.Headers {
		headers[key] = value
	}
	headers[headerMessageID] = messageID
	headers[headerTraceID] = traceID
	headers[headerRetryCount] = int32(retryCount)
	headers[headerFirstTry] = firstTry.UnixMilli()
	return headers
}

func tableInt(table amqp.Table, key string) (int, bool) {
	value, ok := table[key]
	if !ok {
		return 0, false
	}
	switch number := value.(type) {
	case int:
		return number, true
	case int32:
		return int(number), true
	case int64:
		return int(number), true
	case uint32:
		return int(number), true
	case uint64:
		return int(number), true
	default:
		return 0, false
	}
}

func tableInt64(table amqp.Table, key string) (int64, bool) {
	value, ok := table[key]
	if !ok {
		return 0, false
	}
	switch number := value.(type) {
	case int:
		return int64(number), true
	case int32:
		return int64(number), true
	case int64:
		return number, true
	case uint32:
		return int64(number), true
	case uint64:
		return int64(number), true
	default:
		return 0, false
	}
}
