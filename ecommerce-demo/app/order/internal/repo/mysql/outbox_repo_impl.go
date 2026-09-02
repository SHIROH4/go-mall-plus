package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ecommerce-demo/app/order/internal/repo"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type outboxRepoImpl struct {
	db *gorm.DB
}

func NewOutboxRepo(db *gorm.DB) repo.OutboxRepo {
	return &outboxRepoImpl{db: db}
}

// Insert 在已存在的 GORM 事务中插入出站消息
func (r *outboxRepoImpl) Insert(ctx context.Context, record *repo.OutboxRecord) error {
	if record.MaxRetries == 0 {
		record.MaxRetries = repo.DefaultOutboxMaxRetries
	}
	if record.NextRetryAt.IsZero() {
		record.NextRetryAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(record).Error
}

// InsertBatch 批量插入出站消息
func (r *outboxRepoImpl) InsertBatch(ctx context.Context, records []*repo.OutboxRecord) error {
	if len(records) == 0 {
		return nil
	}
	for _, record := range records {
		if record.MaxRetries == 0 {
			record.MaxRetries = repo.DefaultOutboxMaxRetries
		}
		if record.NextRetryAt.IsZero() {
			record.NextRetryAt = time.Now()
		}
	}
	return r.db.WithContext(ctx).Create(&records).Error
}

func (r *outboxRepoImpl) CountPendingMessages(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&repo.OutboxRecord{}).
		Where("status = ?", repo.OutboxStatusPending).
		Count(&count).Error
	return count, err
}

// ClaimPendingMessages 使用条件更新原子抢占消息，避免多副本重复投递。
// processing 消息超过租约后允许被其他 Worker 接管。
func (r *outboxRepoImpl) ClaimPendingMessages(ctx context.Context, limit int, leaseDuration time.Duration) ([]*repo.OutboxRecord, error) {
	if limit <= 0 {
		return nil, nil
	}
	if leaseDuration <= 0 {
		leaseDuration = time.Minute
	}

	now := time.Now()
	staleBefore := now.Add(-leaseDuration)
	claimable := `(status = ? AND next_retry_at <= ?)
		OR (status = ? AND locked_at IS NOT NULL AND locked_at <= ?)`

	var candidates []repo.OutboxRecord
	if err := r.db.WithContext(ctx).
		Select("id").
		Where("retry_count < max_retries").
		Where(claimable,
			repo.OutboxStatusPending, now,
			repo.OutboxStatusProcessing, staleBefore,
		).
		Order("next_retry_at ASC, id ASC").
		Limit(limit * 2).
		Find(&candidates).Error; err != nil {
		return nil, err
	}

	claimed := make([]*repo.OutboxRecord, 0, limit)
	for _, candidate := range candidates {
		if len(claimed) >= limit {
			break
		}

		lockToken := uuid.NewString()
		result := r.db.WithContext(ctx).
			Model(&repo.OutboxRecord{}).
			Where("id = ? AND retry_count < max_retries", candidate.ID).
			Where(claimable,
				repo.OutboxStatusPending, now,
				repo.OutboxStatusProcessing, staleBefore,
			).
			Updates(map[string]interface{}{
				"status":     repo.OutboxStatusProcessing,
				"lock_token": lockToken,
				"locked_at":  now,
			})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}

		var record repo.OutboxRecord
		if err := r.db.WithContext(ctx).
			Where("id = ? AND lock_token = ?", candidate.ID, lockToken).
			First(&record).Error; err != nil {
			return nil, err
		}
		claimed = append(claimed, &record)
	}

	return claimed, nil
}

// MarkCompleted 标记消息为已发送
func (r *outboxRepoImpl) MarkCompleted(ctx context.Context, id int64, lockToken string) error {
	result := r.db.WithContext(ctx).
		Model(&repo.OutboxRecord{}).
		Where("id = ? AND status = ? AND lock_token = ?", id, repo.OutboxStatusProcessing, lockToken).
		Updates(completionUpdates())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrOutboxClaimLost
	}
	return nil
}

func completionUpdates() map[string]interface{} {
	return map[string]interface{}{
		"status":        repo.OutboxStatusCompleted,
		"error_message": "",
		"lock_token":    "",
		"locked_at":     nil,
	}
}

// MarkFailed 标记失败并计算下次重试时间（指数退避）
func (r *outboxRepoImpl) MarkFailed(ctx context.Context, id int64, lockToken string, errMsg string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record repo.OutboxRecord
		if err := tx.Where("id = ? AND status = ? AND lock_token = ?", id, repo.OutboxStatusProcessing, lockToken).
			First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repo.ErrOutboxClaimLost
			}
			return err
		}

		nextRetries := record.RetryCount + 1
		updates := failureUpdates(time.Now(), record.MaxRetries, nextRetries, errMsg)

		return tx.Model(&repo.OutboxRecord{}).
			Where("id = ? AND status = ? AND lock_token = ?", id, repo.OutboxStatusProcessing, lockToken).
			Updates(updates).Error
	})
}

func failureUpdates(now time.Time, maxRetries, nextRetries int32, errMsg string) map[string]interface{} {
	updates := map[string]interface{}{
		"retry_count":   nextRetries,
		"error_message": fmt.Sprintf("%.200s", errMsg),
		"lock_token":    "",
		"locked_at":     nil,
	}

	if nextRetries >= maxRetries {
		updates["status"] = repo.OutboxStatusFailed
		return updates
	}

	backoff := time.Duration(1<<nextRetries) * time.Second
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute
	}
	updates["status"] = repo.OutboxStatusPending
	updates["next_retry_at"] = now.Add(backoff)
	return updates
}
