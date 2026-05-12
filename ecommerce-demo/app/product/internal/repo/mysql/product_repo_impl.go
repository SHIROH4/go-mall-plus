package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"ecommerce-demo/app/product/internal/repo"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

const (
	// 用于缓存穿透的空对象标记
	emptyCacheToken = "{}"
)

// Stock 库存结构
type Stock struct {
	ID        int64 `gorm:"column:id;primaryKey"`
	ProductID int64 `gorm:"column:product_id;uniqueIndex"`
	StockNum  int32 `gorm:"column:stock_num"`
	Version   int32 `gorm:"column:version"`
}

func (Stock) TableName() string {
	return "stock"
}

type productRepoImpl struct {
	db  *gorm.DB
	rdb *redis.Client
	sfg singleflight.Group // [新增] 防缓存击穿的核心并发控制组
}

func NewProductRepo(db *gorm.DB, rdb *redis.Client) repo.ProductRepo {
	return &productRepoImpl{
		db:  db,
		rdb: rdb,
	}
}

func (r *productRepoImpl) GetProductByID(ctx context.Context, id int64) (*repo.Product, error) {
	cacheKey := fmt.Sprintf("product:info:%d", id)

	// ==========================================
	// 1. 第一层查询：直接查 Redis (常规流程)
	// ==========================================
	val, err := r.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		// 【防穿透拦截】如果读到的是特殊空标记，说明数据库里真没这数据，直接返回 nil，阻断恶意流量！
		if val == emptyCacheToken {
			return nil, nil
		}
		// 正常命中缓存
		var p repo.Product
		if err := json.Unmarshal([]byte(val), &p); err == nil {
			return &p, nil
		}
	} else if err != redis.Nil {
		fmt.Printf("Redis 异常，降级处理: %v\n", err)
	}

	// ==========================================
	// 2. 缓存未命中，准备查 DB。【防击穿核心：Singleflight】
	// ==========================================
	// 假设有 10000 个高并发请求发现缓存失效，冲到了这里
	// sfg.Do 保证针对同一个 cacheKey，只有一个 Goroutine 会真正执行内部的匿名函数，其他的全部阻塞等待！
	v, err, _ := r.sfg.Do(cacheKey, func() (interface{}, error) {

		// 【极其致命的细节：Double Check 双重检查】
		// 获得执行权的这 1 个协程，需要再查一次 Redis！
		// 因为有可能上一个夺得执行权的协程刚刚处理完把数据放进了 Redis。
		val, err := r.rdb.Get(ctx, cacheKey).Result()
		if err == nil {
			if val == emptyCacheToken {
				return nil, nil
			}
			var p repo.Product
			if err := json.Unmarshal([]byte(val), &p); err == nil {
				return &p, nil
			}
		}

		// 真正去 MySQL 查询
		var p repo.Product
		err = r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 【防穿透核心】数据库里真没有这个商品！
				// 我们在 Redis 里存入一个空标记 "{}"，过期时间设短一点（比如 1 分钟）。
				// 这样黑客接下来 1 分钟内的十万次恶意重试，都会被最上面的 if val == emptyCacheToken 挡住，MySQL 安然无恙！
				r.rdb.Set(ctx, cacheKey, emptyCacheToken, 1*time.Minute)
				return nil, nil
			}
			return nil, err
		}

		// 【防雪崩核心】给缓存时间加上随机抖动
		// 如果双11我们同时上架了 1000 个商品，绝不能让它们在同一秒过期，否则那一秒会引发雪崩
		// 基础过期时间 10 分钟 + 随机 0~5 分钟的抖动 (Jitter)
		baseExpiration := 10 * time.Minute
		jitter := time.Duration(rand.Intn(300)) * time.Second
		finalExpiration := baseExpiration + jitter

		// 将查询到的真实数据写入 Redis
		if pBytes, err := json.Marshal(&p); err == nil {
			r.rdb.Set(ctx, cacheKey, pBytes, finalExpiration)
		}

		// 将数据通过 Singleflight 共享给另外等待的 9999 个协程
		return &p, nil
	})

	if err != nil {
		return nil, err
	}

	// 从 Singleflight 返回的是 interface{}，且如果触发了防穿透机制，v 会是 nil
	if v == nil {
		return nil, nil
	}

	// 类型断言并返回
	return v.(*repo.Product), nil
}

// ListProducts 获取商品列表
func (r *productRepoImpl) ListProducts(ctx context.Context) ([]*repo.Product, error) {
	cacheKey := "product:list"

	// 1. 先查 Redis 缓存
	val, err := r.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var products []*repo.Product
		if err := json.Unmarshal([]byte(val), &products); err == nil {
			return products, nil
		}
	} else if err != redis.Nil {
		fmt.Printf("Redis 异常，降级处理: %v\n", err)
	}

	// 2. 缓存未命中，查 MySQL
	var products []*repo.Product
	err = r.db.WithContext(ctx).Find(&products).Error
	if err != nil {
		return nil, err
	}

	// 3. 写入缓存（加随机抖动防雪崩）
	if pBytes, err := json.Marshal(&products); err == nil {
		baseExpiration := 5 * time.Minute
		jitter := time.Duration(rand.Intn(120)) * time.Second
		r.rdb.Set(ctx, cacheKey, pBytes, baseExpiration+jitter)
	}

	return products, nil
}

// AddProduct 添加商品
func (r *productRepoImpl) AddProduct(ctx context.Context, p *repo.Product, stock int32) (int64, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 创建商品记录
		if err := tx.Create(p).Error; err != nil {
			return err
		}

		// 2. 初始化库存记录
		stockRecord := &Stock{
			ProductID: p.ID,
			StockNum:  stock,
			Version:   0,
		}
		if err := tx.Create(stockRecord).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	// 3. 同步库存到 Redis（用于订单服务扣减）
	stockKey := fmt.Sprintf("stock:%d", p.ID)
	if err := r.rdb.SetNX(ctx, stockKey, stock, 0).Err(); err != nil {
		fmt.Printf("⚠️ 库存同步到Redis失败: ProductID=%d, Stock=%d, Err=%v\n", p.ID, stock, err)
	}

	// 4. 清除列表相关缓存
	r.clearListCache(ctx)

	return p.ID, nil
}

// clearListCache 清除商品列表相关缓存
func (r *productRepoImpl) clearListCache(ctx context.Context) {
	// 清除全量列表缓存
	r.rdb.Del(ctx, "product:list")
	// 清除分页缓存（使用 SCAN 匹配前缀）
	iter := r.rdb.Scan(ctx, 0, "product:list:page:*", 100).Iterator()
	for iter.Next(ctx) {
		r.rdb.Del(ctx, iter.Val())
	}
}

// UpdateProduct 更新商品信息
func (r *productRepoImpl) UpdateProduct(ctx context.Context, p *repo.Product) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新商品信息
		if err := tx.Model(&repo.Product{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
			"name": p.Name,
			"desc": p.Desc,
			"price": p.Price,
		}).Error; err != nil {
			return err
		}

		// 清除缓存
		cacheKey := fmt.Sprintf("product:info:%d", p.ID)
		r.rdb.Del(ctx, cacheKey)

		return nil
	})

	if err != nil {
		return err
	}

	// 清除列表缓存
	r.clearListCache(ctx)
	return nil
}

// DeleteProduct 删除商品
func (r *productRepoImpl) DeleteProduct(ctx context.Context, id int64) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 删除商品记录
		if err := tx.Where("id = ?", id).Delete(&repo.Product{}).Error; err != nil {
			return err
		}

		// 2. 删除库存记录
		if err := tx.Where("product_id = ?", id).Delete(&Stock{}).Error; err != nil {
			return err
		}

		// 3. 清除缓存
		cacheKey := fmt.Sprintf("product:info:%d", id)
		r.rdb.Del(ctx, cacheKey)

		return nil
	})

	if err != nil {
		return err
	}

	// 4. 清除列表缓存
	r.clearListCache(ctx)
	return nil
}

// ListProductsByPage 分页查询商品
func (r *productRepoImpl) ListProductsByPage(ctx context.Context, categoryID int64, keyword string, page, pageSize int32) ([]*repo.Product, int32, error) {
	// 构建缓存 key
	cacheKey := fmt.Sprintf("product:list:page:%d:%d:cat:%d:kw:%s", page, pageSize, categoryID, keyword)

	// 1. 先查缓存
	val, err := r.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var cached struct {
			Products []*repo.Product `json:"products"`
			Total    int32           `json:"total"`
		}
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			return cached.Products, cached.Total, nil
		}
	} else if err != redis.Nil {
		fmt.Printf("Redis 异常，降级处理: %v\n", err)
	}

	var products []*repo.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&repo.Product{})

	// 分类筛选
	if categoryID > 0 {
		query = query.Where("category_id = ?", categoryID)
	}

	// 关键词搜索（支持名称和描述）
	if keyword != "" {
		search := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR `desc` LIKE ?", search, search)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Order("create_time DESC").Find(&products).Error; err != nil {
		return nil, 0, err
	}

	// 写入缓存
	cached := struct {
		Products []*repo.Product `json:"products"`
		Total    int32           `json:"total"`
	}{products, int32(total)}
	if pBytes, err := json.Marshal(&cached); err == nil {
		baseExpiration := 3 * time.Minute
		jitter := time.Duration(rand.Intn(60)) * time.Second
		r.rdb.Set(ctx, cacheKey, pBytes, baseExpiration+jitter)
	}

	return products, int32(total), nil
}

// ListCategories 获取所有分类
func (r *productRepoImpl) ListCategories(ctx context.Context) ([]*repo.Category, error) {
	var categories []*repo.Category
	err := r.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

// AddCategory 添加分类
func (r *productRepoImpl) AddCategory(ctx context.Context, c *repo.Category) (int64, error) {
	err := r.db.WithContext(ctx).Create(c).Error
	if err != nil {
		return 0, err
	}
	return c.ID, nil
}

// GetCategoryByID 获取分类详情
func (r *productRepoImpl) GetCategoryByID(ctx context.Context, id int64) (*repo.Category, error) {
	var category repo.Category
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

// GetStock 获取商品库存
func (r *productRepoImpl) GetStock(ctx context.Context, productID int64) (int32, error) {
	var stock Stock
	err := r.db.WithContext(ctx).Where("product_id = ?", productID).First(&stock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return stock.StockNum, nil
}

// GetStockFromCache 从 Redis 获取库存
func (r *productRepoImpl) GetStockFromCache(ctx context.Context, productID int64) (int32, error) {
	stockKey := fmt.Sprintf("stock:%d", productID)
	val, err := r.rdb.Get(ctx, stockKey).Int()
	if err != nil {
		return 0, err
	}
	return int32(val), nil
}

// SetStockCache 设置库存到 Redis（缓存未命中时调用）
func (r *productRepoImpl) SetStockCache(ctx context.Context, productID int64, stock int32) error {
	stockKey := fmt.Sprintf("stock:%d", productID)
	// 设置 10 分钟过期，带随机抖动防雪崩
	jitter := time.Duration(rand.Intn(120)) * time.Second
	expiration := 10*time.Minute + jitter
	return r.rdb.Set(ctx, stockKey, stock, expiration).Err()
}

// InitStockCache 初始化库存缓存（服务启动时调用）
// 将数据库中的所有库存同步到 Redis，供订单服务使用
func (r *productRepoImpl) InitStockCache(ctx context.Context) error {
	var stocks []Stock
	if err := r.db.WithContext(ctx).Find(&stocks).Error; err != nil {
		return fmt.Errorf("查询库存失败: %w", err)
	}

	successCount := 0
	failCount := 0
	for _, s := range stocks {
		stockKey := fmt.Sprintf("stock:%d", s.ProductID)
		// 使用 SetNX 只在 key 不存在时设置，避免覆盖已有的正确数据
		// 如果失败，使用 Set 强制覆盖
		if err := r.rdb.SetNX(ctx, stockKey, s.StockNum, 0).Err(); err != nil {
			// 尝试强制设置
			if setErr := r.rdb.Set(ctx, stockKey, s.StockNum, 0).Err(); setErr != nil {
				fmt.Printf("⚠️ 库存同步失败: ProductID=%d, Err=%v\n", s.ProductID, setErr)
				failCount++
				continue
			}
		}
		successCount++
	}

	fmt.Printf("📦 库存缓存初始化完成: 成功=%d, 失败=%d\n", successCount, failCount)
	return nil
}
