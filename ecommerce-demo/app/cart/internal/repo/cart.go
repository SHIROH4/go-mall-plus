package repo

import (
	"context"
	"ecommerce-demo/app/cart/pb"
	"errors"
	"time"
)

const (
	MaxItemCount  = 99       // 单商品最大数量
	MaxCartSize   = 50       // 单用户购物车最大商品数
	CartExpire    = 720 * time.Hour // 购物车 30 天过期
	CartKeyPrefix = "cart:"
)

var (
	ErrCartEmpty      = errors.New("购物车为空")
	ErrItemNotFound   = errors.New("商品不在购物车中")
	ErrItemCountLimit = errors.New("单商品数量已达上限")
	ErrCartFull       = errors.New("购物车已满")
)

type CartRepo interface {
	// AddItem 添加商品到购物车
	// 如果商品已存在，则累加数量
	AddItem(ctx context.Context, userId int64, item *pb.CartItem) error

	// GetCart 获取用户购物车所有商品
	GetCart(ctx context.Context, userId int64) ([]*pb.CartItem, error)

	// UpdateItem 更新购物车商品数量
	UpdateItem(ctx context.Context, userId int64, productId int64, count int32) error

	// RemoveItem 从购物车删除商品
	RemoveItem(ctx context.Context, userId int64, productId int64) error

	// ClearCart 清空用户购物车
	ClearCart(ctx context.Context, userId int64) (int32, error)

	// SelectItem 勾选/取消勾选商品
	SelectItem(ctx context.Context, userId int64, productId int64, selected bool) error

	// GetSelectedItems 获取已选中的商品（用于结算）
	GetSelectedItems(ctx context.Context, userId int64) ([]*pb.CartItem, error)

	// GetCartSize 获取购物车商品数量
	GetCartSize(ctx context.Context, userId int64) (int32, error)

	// CartKey 获取用户购物车的 Redis Key
	CartKey(userId int64) string

	// ItemKey 获取购物车商品的 Field Key
	ItemKey(productId int64) string
}
