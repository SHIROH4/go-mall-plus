package mysql

import (
	"strings"
	"testing"
	"time"

	"ecommerce-demo/app/order/internal/repo"

	"github.com/stretchr/testify/assert"
)

func TestFailureUpdatesSchedulesRetryWithExponentialBackoff(t *testing.T) {
	now := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

	updates := failureUpdates(now, 5, 2, "temporary failure")

	assert.Equal(t, repo.OutboxStatusPending, updates["status"])
	assert.Equal(t, int32(2), updates["retry_count"])
	assert.Equal(t, now.Add(4*time.Second), updates["next_retry_at"])
	assert.Equal(t, "", updates["lock_token"])
	assert.Nil(t, updates["locked_at"])
}

func TestFailureUpdatesStopsAtMaximumRetries(t *testing.T) {
	updates := failureUpdates(time.Now(), 5, 5, "permanent failure")

	assert.Equal(t, repo.OutboxStatusFailed, updates["status"])
	assert.NotContains(t, updates, "next_retry_at")
}

func TestFailureUpdatesLimitsStoredErrorMessage(t *testing.T) {
	updates := failureUpdates(time.Now(), 5, 1, strings.Repeat("x", 300))

	assert.Len(t, updates["error_message"], 200)
}
