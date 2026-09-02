package repo

import (
	"context"
	"errors"
	"time"
)

// OutboxMessageType 消息类型
const (
	OutboxTypeOrderCreated        = "order.created"
	OutboxTypeOrderDelay          = "order.delay.check"
	DefaultOutboxMaxRetries int32 = 5
)

// OutboxStatus 出站消息状态
const (
	OutboxStatusPending    int8 = 0 // 待发送
	OutboxStatusProcessing int8 = 1 // 发送中
	OutboxStatusCompleted  int8 = 2 // 已发送
	OutboxStatusFailed     int8 = 3 // 超过最大重试次数
)

var ErrOutboxClaimLost = errors.New("outbox claim lost or expired")

// OutboxRecord 本地消息表实体
type OutboxRecord struct {
	ID           int64      `gorm:"column:id;primaryKey;autoIncrement"`
	MessageType  string     `gorm:"column:message_type"`
	Payload      string     `gorm:"column:payload"`
	Status       int8       `gorm:"column:status"`
	RetryCount   int32      `gorm:"column:retry_count"`
	MaxRetries   int32      `gorm:"column:max_retries;default:5"`
	NextRetryAt  time.Time  `gorm:"column:next_retry_at"`
	ErrorMessage string     `gorm:"column:error_message"`
	LockToken    string     `gorm:"column:lock_token"`
	LockedAt     *time.Time `gorm:"column:locked_at"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (OutboxRecord) TableName() string { return "outbox" }

// OutboxRepo 本地消息表仓储接口
type OutboxRepo interface {
	// Insert 插入一条出站消息（在业务事务内调用）
	Insert(ctx context.Context, record *OutboxRecord) error
	// InsertBatch 批量插入出站消息（在业务事务内调用）
	InsertBatch(ctx context.Context, records []*OutboxRecord) error
	// CountPendingMessages 返回当前待投递消息总数，用于积压监控。
	CountPendingMessages(ctx context.Context) (int64, error)
	// ClaimPendingMessages 原子抢占待发送消息；超时租约可由其他实例接管。
	ClaimPendingMessages(ctx context.Context, limit int, leaseDuration time.Duration) ([]*OutboxRecord, error)
	// MarkCompleted 标记消息为已发送
	MarkCompleted(ctx context.Context, id int64, lockToken string) error
	// MarkFailed 标记消息发送失败，更新重试次数和下次重试时间
	MarkFailed(ctx context.Context, id int64, lockToken string, errMsg string) error
}
