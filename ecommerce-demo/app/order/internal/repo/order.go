package repo

import (
    "context"
    "errors"
    "time"
)

// Order 订单表实体
type Order struct {
    ID          int64     `gorm:"column:id;primaryKey;autoIncrement"`
    OrderNo     string    `gorm:"column:order_no"`
    UserID      int64     `gorm:"column:user_id"`
    ProductID   int64     `gorm:"column:product_id"`
    Count       int32     `gorm:"column:count"`
    TotalAmount int64     `gorm:"column:total_amount"`
    Status      int8      `gorm:"column:status"`
    // [新增] 订单超时时间
    ExpireTime  *time.Time `gorm:"column:expire_time"`
    CreateTime  time.Time  `gorm:"column:create_time;autoCreateTime"`
    UpdateTime  time.Time  `gorm:"column:update_time;autoUpdateTime"`
}

func (Order) TableName() string { return "order" }

// OrderStatus 订单状态常量
type OrderStatus int8

const (
    OrderStatusPending   OrderStatus = 0 // 待支付
    OrderStatusPaid      OrderStatus = 1 // 已支付
    OrderStatusCancelled OrderStatus = 2 // 已取消
    OrderStatusTimeout   OrderStatus = 3 // 已超时
)

// StatusText 返回状态文本
func (s OrderStatus) StatusText() string {
    switch s {
    case OrderStatusPending:
        return "待支付"
    case OrderStatusPaid:
        return "已支付"
    case OrderStatusCancelled:
        return "已取消"
    case OrderStatusTimeout:
        return "已超时"
    default:
        return "未知状态"
    }
}

// OrderRepo 订单仓储接口
type OrderRepo interface {
    // Redis Lua 原子扣减 (保留兼容，新流程通过 StockRPC)
    DeductStockCache(ctx context.Context, productID int64, count int32) (bool, error)
    // Redis 库存回滚 (保留兼容)
    RollbackStockCache(ctx context.Context, productID int64, count int32) error
    // 创建订单 + 扣减 MySQL 库存 (原有)
    CreateOrderTx(ctx context.Context, order *Order, expireTime time.Time, count int32) error
    // 【新】创建订单 + 扣减 MySQL 库存 + 写入 Outbox 消息（单事务）
    CreateOrderWithOutboxTx(ctx context.Context, order *Order, expireTime time.Time, count int32, outboxRecords []*OutboxRecord) error
    GetOrderByNo(ctx context.Context, orderNo string) (*Order, error)
    UpdateOrderStatus(ctx context.Context, orderNo string, status int8) error
    ListOrdersByUser(ctx context.Context, userID int64, page, pageSize int32, status int8) ([]*Order, int32, error)
    CancelOrderTx(ctx context.Context, orderNo string, userID int64, productID int64, count int32) error
    // 超时取消订单事务（状态更新 + 库存回滚）
    TimeoutOrderTx(ctx context.Context, orderNo string, productID int64, count int32) error
    // 批量查询超时订单（用于定时扫描兜底）
    ListTimeoutOrders(ctx context.Context, limit int32) ([]*Order, error)
}

var (
    ErrOrderNotFound       = errors.New("订单不存在")
    ErrOrderStatusInvalid  = errors.New("订单状态不允许此操作")
)
