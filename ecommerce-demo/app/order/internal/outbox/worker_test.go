package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"ecommerce-demo/app/order/internal/mq"
	"ecommerce-demo/app/order/internal/repo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOutboxRepo struct {
	records        []*repo.OutboxRecord
	claimLimit     int
	claimLease     time.Duration
	completedID    int64
	completedToken string
	failedID       int64
	failedToken    string
	failedMessage  string
}

func (f *fakeOutboxRepo) Insert(context.Context, *repo.OutboxRecord) error { return nil }

func (f *fakeOutboxRepo) InsertBatch(context.Context, []*repo.OutboxRecord) error { return nil }

func (f *fakeOutboxRepo) ClaimPendingMessages(_ context.Context, limit int, lease time.Duration) ([]*repo.OutboxRecord, error) {
	f.claimLimit = limit
	f.claimLease = lease
	return f.records, nil
}

func (f *fakeOutboxRepo) MarkCompleted(_ context.Context, id int64, lockToken string) error {
	f.completedID = id
	f.completedToken = lockToken
	return nil
}

func (f *fakeOutboxRepo) MarkFailed(_ context.Context, id int64, lockToken, errMsg string) error {
	f.failedID = id
	f.failedToken = lockToken
	f.failedMessage = errMsg
	return nil
}

type fakeProducer struct {
	orderErr       error
	publishedOrder *mq.OrderMsg
}

func (f *fakeProducer) PublishOrder(_ context.Context, msg *mq.OrderMsg) error {
	f.publishedOrder = msg
	return f.orderErr
}

func (f *fakeProducer) PublishDelayOrder(context.Context, *mq.DelayOrderMsg, int) error { return nil }

func (f *fakeProducer) Close() error { return nil }

func TestWorkerClaimsAndCompletesOrderCreatedEvent(t *testing.T) {
	repository := &fakeOutboxRepo{records: []*repo.OutboxRecord{{
		ID:          42,
		MessageType: repo.OutboxTypeOrderCreated,
		Payload:     `{"orderNo":"ORD42","userId":7,"productId":11,"count":2,"totalAmount":19900}`,
		LockToken:   "claim-42",
	}}}
	producer := &fakeProducer{}
	worker := &Worker{
		outboxRepo:   repository,
		producer:     producer,
		batchSize:    10,
		claimTimeout: 45 * time.Second,
	}

	worker.pollAndPublish()

	require.NotNil(t, producer.publishedOrder)
	assert.Equal(t, "ORD42", producer.publishedOrder.OrderNo)
	assert.Equal(t, 10, repository.claimLimit)
	assert.Equal(t, 45*time.Second, repository.claimLease)
	assert.Equal(t, int64(42), repository.completedID)
	assert.Equal(t, "claim-42", repository.completedToken)
	assert.Zero(t, repository.failedID)
}

func TestWorkerReturnsFailedPublishToRetryStateWithClaimToken(t *testing.T) {
	repository := &fakeOutboxRepo{records: []*repo.OutboxRecord{{
		ID:          43,
		MessageType: repo.OutboxTypeOrderCreated,
		Payload:     `{"orderNo":"ORD43"}`,
		LockToken:   "claim-43",
	}}}
	producer := &fakeProducer{orderErr: errors.New("broker unavailable")}
	worker := &Worker{
		outboxRepo:   repository,
		producer:     producer,
		batchSize:    10,
		claimTimeout: time.Minute,
	}

	worker.pollAndPublish()

	assert.Equal(t, int64(43), repository.failedID)
	assert.Equal(t, "claim-43", repository.failedToken)
	assert.Equal(t, "broker unavailable", repository.failedMessage)
	assert.Zero(t, repository.completedID)
}

func TestWorkerRejectsMalformedPayloadWithoutSilentlyCompletingIt(t *testing.T) {
	repository := &fakeOutboxRepo{records: []*repo.OutboxRecord{{
		ID:          44,
		MessageType: repo.OutboxTypeOrderCreated,
		Payload:     `{not-json}`,
		LockToken:   "claim-44",
	}}}
	worker := &Worker{
		outboxRepo:   repository,
		producer:     &fakeProducer{},
		batchSize:    10,
		claimTimeout: time.Minute,
	}

	worker.pollAndPublish()

	assert.Equal(t, int64(44), repository.failedID)
	assert.Equal(t, "claim-44", repository.failedToken)
	assert.NotEmpty(t, repository.failedMessage)
	assert.Zero(t, repository.completedID)
}
