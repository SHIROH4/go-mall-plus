package mysql

import (
	"context"
	"fmt"
	"time"

	"ecommerce-demo/app/order/internal/repo"

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
		record.MaxRetries = 5
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
			record.MaxRetries = 5
		}
		if record.NextRetryAt.IsZero() {
			record.NextRetryAt = time.Now()
		}
	}
	return r.db.WithContext(ctx).Create(&records).Error
}

// FetchPendingMessages 拉取待发送消息
func (r *outboxRepoImpl) FetchPendingMessages(ctx context.Context, limit int) ([]*repo.OutboxRecord, error) {
	var records []*repo.OutboxRecord
	err := r.db.WithContext(ctx).
		Where("status IN ? AND next_retry_at <= ?", []int8{repo.OutboxStatusPending, repo.OutboxStatusProcessing}, time.Now()).
		Order("next_retry_at ASC").
		Limit(limit).
		Find(&records).Error
	return records, err
}

// MarkCompleted 标记消息为已发送
func (r *outboxRepoImpl) MarkCompleted(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&repo.OutboxRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status": repo.OutboxStatusCompleted,
		}).Error
}

// MarkFailed 标记失败并计算下次重试时间（指数退避）
func (r *outboxRepoImpl) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record repo.OutboxRecord
		if err := tx.Where("id = ?", id).First(&record).Error; err != nil {
			return err
		}
		nextRetries := record.RetryCount + 1
		backoff := time.Duration(1<<nextRetries) * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
		nextRetryAt := time.Now().Add(backoff)
		return tx.Model(&repo.OutboxRecord{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":        repo.OutboxStatusPending,
				"retry_count":   nextRetries,
				"next_retry_at": nextRetryAt,
				"error_message": fmt.Sprintf("%.200s", errMsg),
			}).Error
	})
}
