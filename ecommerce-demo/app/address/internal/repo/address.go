package repo

import (
	"context"
	"time"
)

// Address 收货地址实体
type Address struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID        int64     `gorm:"column:user_id;index"`
	ReceiverName  string    `gorm:"column:receiver_name;size:50"`
	Phone         string    `gorm:"column:phone;size:20"`
	Province      string    `gorm:"column:province;size:50"`
	City          string    `gorm:"column:city;size:50"`
	District      string    `gorm:"column:district;size:50"`
	DetailAddress string    `gorm:"column:detail_address;size:200"`
	PostalCode    string    `gorm:"column:postal_code;size:10"`
	IsDefault     bool      `gorm:"column:is_default;default:false"`
	CreateTime    time.Time `gorm:"column:create_time;autoCreateTime;<-:create"`
	UpdateTime    time.Time `gorm:"column:update_time;autoUpdateTime"`
}

func (Address) TableName() string {
	return "address"
}

// AddressRepo 收货地址仓储层接口
type AddressRepo interface {
	// GetAddressByID 根据ID获取地址
	GetAddressByID(ctx context.Context, id int64) (*Address, error)
	// ListAddressesByUserID 获取用户的所有地址
	ListAddressesByUserID(ctx context.Context, userID int64) ([]*Address, error)
	// GetDefaultAddress 获取用户默认地址
	GetDefaultAddress(ctx context.Context, userID int64) (*Address, error)
	// CreateAddress 创建地址
	CreateAddress(ctx context.Context, addr *Address) (int64, error)
	// UpdateAddress 更新地址
	UpdateAddress(ctx context.Context, addr *Address) error
	// DeleteAddress 删除地址
	DeleteAddress(ctx context.Context, id int64, userID int64) error
	// ClearDefaultAddress 清除用户的所有默认地址标记
	ClearDefaultAddress(ctx context.Context, userID int64) error
	// SetDefaultAddress 设置默认地址
	SetDefaultAddress(ctx context.Context, id int64, userID int64) error
}
