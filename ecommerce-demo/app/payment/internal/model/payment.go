package model

import (
	"time"
)

type PaymentStatus int32

const (
	PaymentStatus_Pending   PaymentStatus = 0 // 待支付
	PaymentStatus_Paid     PaymentStatus = 1 // 已支付
	PaymentStatus_Cancelled PaymentStatus = 2 // 已取消
	PaymentStatus_Expired   PaymentStatus = 3 // 已超时
)

func (s PaymentStatus) String() string {
	switch s {
	case PaymentStatus_Pending:
		return "待支付"
	case PaymentStatus_Paid:
		return "已支付"
	case PaymentStatus_Cancelled:
		return "已取消"
	case PaymentStatus_Expired:
		return "已超时"
	default:
		return "未知状态"
	}
}

type Payment struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PaymentNo   string    `gorm:"type:varchar(32);uniqueIndex;not null" json:"payment_no"`
	OrderNo     string    `gorm:"type:varchar(32);index;not null" json:"order_no"`
	UserID      int64     `gorm:"index;not null" json:"user_id"`
	Amount      int64     `gorm:"not null" json:"amount"` // 单位：分
	Status      PaymentStatus `gorm:"default:0;index" json:"status"`
	PayChannel  string    `gorm:"type:varchar(20)" json:"pay_channel"`
	PayTime     *time.Time `json:"pay_time"`
	ExpireTime  time.Time `gorm:"not null" json:"expire_time"`
	CallbackData string   `gorm:"type:text" json:"callback_data"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Payment) TableName() string {
	return "payment"
}

func (p *Payment) IsExpired() bool {
	return time.Now().After(p.ExpireTime)
}

func (p *Payment) CanCancel() bool {
	return p.Status == PaymentStatus_Pending
}

func (p *Payment) CanPay() bool {
	return p.Status == PaymentStatus_Pending && !p.IsExpired()
}
