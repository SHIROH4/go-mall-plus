package repo

import (
	"context"
	"time"
)

// OutboxMessageType 消息类型
const (
	OutboxTypeOrderCreated  = "order.created"
	OutboxTypeOrderDelay    = "order.delay.check"
)

// OutboxStatus 出站消息状态
const (
	OutboxStatusPending    int8 = 0 // 待发送
	OutboxStatusProcessing int8 = 1 // 发送中
	OutboxStatusCompleted  int8 = 2 // 已发送
)

// OutboxRecord 本地消息表实体
type OutboxRecord struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	MessageType  string    `gorm:"column:message_type"`
	Payload      string    `gorm:"column:payload"`
	Status       int8      `gorm:"column:status"`
	RetryCount   int32     `gorm:"column:retry_count"`
	MaxRetries   int32     `gorm:"column:max_retries"`
	NextRetryAt  time.Time `gorm:"column:next_retry_at"`
	ErrorMessage string    `gorm:"column:error_message"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (OutboxRecord) TableName() string { return "outbox" }

// OutboxRepo 本地消息表仓储接口
type OutboxRepo interface {
	// Insert 插入一条出站消息（在业务事务内调用）
	Insert(ctx context.Context, record *OutboxRecord) error
	// InsertBatch 批量插入出站消息（在业务事务内调用）
	InsertBatch(ctx context.Context, records []*OutboxRecord) error
	// FetchPendingMessages 拉取待发送消息（按 next_retry_at 升序）
	FetchPendingMessages(ctx context.Context, limit int) ([]*OutboxRecord, error)
	// MarkCompleted 标记消息为已发送
	MarkCompleted(ctx context.Context, id int64) error
	// MarkFailed 标记消息发送失败，更新重试次数和下次重试时间
	MarkFailed(ctx context.Context, id int64, errMsg string) error
}
