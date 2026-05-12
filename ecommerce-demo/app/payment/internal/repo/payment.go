package repo

import (
	"context"
	"ecommerce-demo/app/payment/internal/model"
)

var (
	ErrPaymentNotFound = ErrCode("payment_not_found")
	ErrPaymentExpired  = ErrCode("payment_expired")
	ErrPaymentCancelled = ErrCode("payment_cancelled")
	ErrPaymentPaid     = ErrCode("payment_already_paid")
	ErrInvalidStatus   = ErrCode("invalid_payment_status")
)

type ErrCode string

func (e ErrCode) Error() string {
	return string(e)
}

type PaymentRepo interface {
	// Create 创建支付单
	Create(ctx context.Context, payment *model.Payment) error

	// GetByPaymentNo 根据支付单号查询
	GetByPaymentNo(ctx context.Context, paymentNo string) (*model.Payment, error)

	// GetByOrderNo 根据订单号查询
	GetByOrderNo(ctx context.Context, orderNo string) (*model.Payment, error)

	// UpdateStatus 更新支付状态
	UpdateStatus(ctx context.Context, paymentNo string, status model.PaymentStatus, callbackData string) error

	// UpdateStatusWithTx 带事务更新状态
	UpdateStatusWithTx(ctx context.Context, paymentNo string, status model.PaymentStatus, callbackData string) error

	// ListByUserID 分页查询用户支付记录
	ListByUserID(ctx context.Context, userID int64, page, pageSize int32) ([]*model.Payment, int32, error)

	// GetExpiredPayments 查询过期的待支付单
	GetExpiredPayments(ctx context.Context, limit int) ([]*model.Payment, error)

	// CloseExpiredPayments 关闭过期支付单
	CloseExpiredPayments(ctx context.Context, paymentNos []string) (int64, error)
}
