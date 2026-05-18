package mysql

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"ecommerce-demo/app/stock/internal/repo"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

var deductStockScript = redis.NewScript(`
local stockKey = KEYS[1]
local deductCount = tonumber(ARGV[1])
local currentStock = redis.call('GET', stockKey)
if currentStock == false then return -1 end
currentStock = tonumber(currentStock)
if currentStock >= deductCount then
    redis.call('DECRBY', stockKey, deductCount)
    return 1
else
    return 0
end
`)

type stockRepoImpl struct {
	db  *gorm.DB
	rdb *redis.ClusterClient
	sfg singleflight.Group
}

func NewStockRepo(db *gorm.DB, rdb *redis.ClusterClient) repo.StockRepo {
	return &stockRepoImpl{db: db, rdb: rdb}
}

// DeductStockLua 执行 Lua 脚本原子扣减 Redis 库存
func (r *stockRepoImpl) DeductStockLua(ctx context.Context, productID int64, count int32) (bool, error) {
	stockKey := fmt.Sprintf("{stock}:%d", productID)
	res, err := deductStockScript.Run(ctx, r.rdb, []string{stockKey}, count).Int()
	if err != nil {
		return false, err
	}
	switch res {
	case -1:
		return false, repo.ErrStockNotInitiated
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, nil
	}
}

// RollbackStockLua 回滚 Redis 库存
func (r *stockRepoImpl) RollbackStockLua(ctx context.Context, productID int64, count int32) error {
	stockKey := fmt.Sprintf("{stock}:%d", productID)
	return r.rdb.IncrBy(ctx, stockKey, int64(count)).Err()
}

// GetStockFromCache 从 Redis 缓存查库存（带 singleflight 防击穿）
func (r *stockRepoImpl) GetStockFromCache(ctx context.Context, productID int64) (int32, error) {
	stockKey := fmt.Sprintf("{stock}:%d", productID)
	val, err := r.rdb.Get(ctx, stockKey).Int()
	if err == nil {
		return int32(val), nil
	}
	if err != redis.Nil {
		return 0, err
	}

	// 缓存未命中，singleflight + double check 查 DB 回填
	cacheKey := fmt.Sprintf("stock:sf:%d", productID)
	v, err, _ := r.sfg.Do(cacheKey, func() (interface{}, error) {
		// Double check
		if val, err := r.rdb.Get(ctx, stockKey).Int(); err == nil {
			return int32(val), nil
		}
		stock, err := r.GetStock(ctx, productID)
		if err != nil {
			return int32(0), err
		}
		// 回填缓存（带随机抖动防雪崩）
		baseExp := 10 * time.Minute
		jitter := time.Duration(rand.Intn(120)) * time.Second
		r.rdb.Set(ctx, stockKey, stock, baseExp+jitter)
		return stock, nil
	})
	if err != nil {
		return 0, err
	}
	return v.(int32), nil
}

// BatchGetStockFromCache 批量从 Redis 获取库存
func (r *stockRepoImpl) BatchGetStockFromCache(ctx context.Context, productIDs []int64) (map[int64]int32, error) {
	if len(productIDs) == 0 {
		return map[int64]int32{}, nil
	}

	keys := make([]string, len(productIDs))
	for i, pid := range productIDs {
		keys[i] = fmt.Sprintf("{stock}:%d", pid)
	}

	vals, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int32, len(productIDs))
	for i, v := range vals {
		if v == nil {
			// 缓存未命中，从 MySQL 回填
			stock, err := r.GetStockFromCache(ctx, productIDs[i])
			if err != nil {
				result[productIDs[i]] = 0
			} else {
				result[productIDs[i]] = stock
			}
		} else {
			if num, ok := v.(string); ok {
				var n int64
				fmt.Sscanf(num, "%d", &n)
				result[productIDs[i]] = int32(n)
			}
		}
	}
	return result, nil
}

// GetStock 从 MySQL 查库存
func (r *stockRepoImpl) GetStock(ctx context.Context, productID int64) (int32, error) {
	var stock repo.Stock
	err := r.db.WithContext(ctx).Where("product_id = ?", productID).First(&stock).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return stock.StockNum, nil
}

// SetStockCache 设置库存到 Redis
func (r *stockRepoImpl) SetStockCache(ctx context.Context, productID int64, stock int32) error {
	stockKey := fmt.Sprintf("{stock}:%d", productID)
	jitter := time.Duration(rand.Intn(120)) * time.Second
	expiration := 10*time.Minute + jitter
	return r.rdb.Set(ctx, stockKey, stock, expiration).Err()
}

// UpdateStock 更新 MySQL 库存（使用 version 乐观锁）
func (r *stockRepoImpl) UpdateStock(ctx context.Context, productID int64, delta int32) error {
	result := r.db.WithContext(ctx).Exec(
		"UPDATE stock SET stock_num = stock_num + ?, version = version + 1 WHERE product_id = ? AND stock_num + ? >= 0",
		delta, productID, delta,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrStockNotEnough
	}
	return nil
}

// InitStockCache 启动时将 MySQL 所有库存同步到 Redis
func (r *stockRepoImpl) InitStockCache(ctx context.Context) error {
	var stocks []repo.Stock
	if err := r.db.WithContext(ctx).Find(&stocks).Error; err != nil {
		return fmt.Errorf("查询库存失败: %w", err)
	}

	successCount, failCount := 0, 0
	for _, s := range stocks {
		stockKey := fmt.Sprintf("{stock}:%d", s.ProductID)
		if err := r.rdb.SetNX(ctx, stockKey, s.StockNum, 0).Err(); err != nil {
			if setErr := r.rdb.Set(ctx, stockKey, s.StockNum, 0).Err(); setErr != nil {
				fmt.Printf("库存同步失败: ProductID=%d, Err=%v\n", s.ProductID, setErr)
				failCount++
				continue
			}
		}
		successCount++
	}
	fmt.Printf("库存缓存初始化完成: 成功=%d, 失败=%d\n", successCount, failCount)
	return nil
}

// GetAllStocks 获取所有 MySQL 库存（用于对账）
func (r *stockRepoImpl) GetAllStocks(ctx context.Context) ([]*repo.Stock, error) {
	var stocks []*repo.Stock
	err := r.db.WithContext(ctx).Find(&stocks).Error
	return stocks, err
}
