package repo

import (
	"context"
	"errors"
	"time"
)

// Stock 库存实体映射
type Stock struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	ProductID  int64     `gorm:"column:product_id;uniqueIndex"`
	StockNum   int32     `gorm:"column:stock_num"`
	Version    int32     `gorm:"column:version"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

func (Stock) TableName() string { return "stock" }

var (
	ErrStockNotEnough    = errors.New("库存不足")
	ErrStockNotInitiated = errors.New("库存缓存未初始化")
	ErrStockNotFound     = errors.New("库存记录不存在")
)

// StockRepo 库存仓储接口
type StockRepo interface {
	// Redis Lua 原子扣减
	DeductStockLua(ctx context.Context, productID int64, count int32) (bool, error)
	// Redis 库存回滚
	RollbackStockLua(ctx context.Context, productID int64, count int32) error
	// 从 Redis 获取库存
	GetStockFromCache(ctx context.Context, productID int64) (int32, error)
	// 批量从 Redis 获取库存
	BatchGetStockFromCache(ctx context.Context, productIDs []int64) (map[int64]int32, error)
	// 从 MySQL 获取库存
	GetStock(ctx context.Context, productID int64) (int32, error)
	// 同步库存到 Redis
	SetStockCache(ctx context.Context, productID int64, stock int32) error
	// MySQL 更新库存（乐观锁）
	UpdateStock(ctx context.Context, productID int64, delta int32) error
	// 启动时初始化所有库存缓存（MySQL → Redis）
	InitStockCache(ctx context.Context) error
	// 对账：获取所有 MySQL 库存，用于与 Redis 对比
	GetAllStocks(ctx context.Context) ([]*Stock, error)
}
