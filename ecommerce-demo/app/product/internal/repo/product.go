package repo

import (
	"context"
	"time"
)

// Product 商品实体映射 (对应 MySQL 表)
type Product struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	Name       string    `gorm:"column:name"`
	Desc       string    `gorm:"column:desc"`
	Price      int64     `gorm:"column:price"` // 注意：单位为分
	ImageUrl   string    `gorm:"column:image_url"`
	CategoryID int64     `gorm:"column:category_id"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

func (Product) TableName() string {
	return "product"
}

// Category 商品分类实体
type Category struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	Icon      string    `gorm:"column:icon"`
	Sort      int32     `gorm:"column:sort"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

func (Category) TableName() string {
	return "category"
}

// Stock 库存实体映射
type Stock struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	ProductID int64     `gorm:"column:product_id;uniqueIndex"`
	StockNum  int32     `gorm:"column:stock_num"`
	Version   int32     `gorm:"column:version"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

func (Stock) TableName() string {
	return "stock"
}

// ProductRepo 暴露给业务层的商品仓储接口
// 业务层根本不需要知道底层用了 Redis 还是 MySQL
type ProductRepo interface {
	GetProductByID(ctx context.Context, id int64) (*Product, error)
	ListProducts(ctx context.Context) ([]*Product, error)
	AddProduct(ctx context.Context, p *Product, stock int32) (int64, error)
	UpdateProduct(ctx context.Context, p *Product) error
	DeleteProduct(ctx context.Context, id int64) error
	ListProductsByPage(ctx context.Context, categoryID int64, keyword string, page, pageSize int32) ([]*Product, int32, error)
	GetStock(ctx context.Context, productID int64) (int32, error)
	GetStockFromCache(ctx context.Context, productID int64) (int32, error)
	SetStockCache(ctx context.Context, productID int64, stock int32) error
	InitStockCache(ctx context.Context) error
}

// CategoryRepo 商品分类仓储接口
type CategoryRepo interface {
	ListCategories(ctx context.Context) ([]*Category, error)
	AddCategory(ctx context.Context, c *Category) (int64, error)
	GetCategoryByID(ctx context.Context, id int64) (*Category, error)
}
