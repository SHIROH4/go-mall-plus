package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ecommerce-demo/app/order/internal/repo"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

/*
  订单仓储 MySQL 实现

  核心改造：
  1. CreateOrderTx 新增 expireTime 参数
  2. 新增 TimeoutOrderTx 超时取消事务
  3. 新增 ListTimeoutOrders 批量查询超时订单
*/

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

// restoreTimeoutStockScript 用订单号作为幂等键，保证 Redis 库存最多回补一次。
// 若库存键不存在，只记录补偿标记；后续缓存回源会直接读取已回补的 MySQL 库存。
var restoreTimeoutStockScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 then
    return 0
end
local currentStock = redis.call('GET', KEYS[2])
if currentStock ~= false and tonumber(currentStock) == nil then
    return -1
end
if currentStock ~= false then
    redis.call('INCRBY', KEYS[2], ARGV[1])
end
redis.call('SET', KEYS[1], '1', 'EX', ARGV[2])
return 1
`)

const timeoutStockDedupTTL = 30 * 24 * time.Hour

type orderRepoImpl struct {
	db  *gorm.DB
	rdb *redis.ClusterClient
}

func NewOrderRepo(db *gorm.DB, rdb *redis.ClusterClient) repo.OrderRepo {
	return &orderRepoImpl{db: db, rdb: rdb}
}

// DeductStockCache 执行 Lua 脚本原子扣减库存
func (r *orderRepoImpl) DeductStockCache(ctx context.Context, productID int64, count int32) (bool, error) {
	stockKey := fmt.Sprintf("{stock}:%d", productID)
	res, err := deductStockScript.Run(ctx, r.rdb, []string{stockKey}, count).Int()
	if err != nil {
		return false, err
	}
	if res == -1 {
		return false, errors.New("Redis 中未初始化商品库存")
	}
	if res == 0 {
		return false, nil
	}
	return true, nil
}

// RollbackStockCache 补偿机制：如果数据库宕机，要把 Redis 扣掉的库存加回来
func (r *orderRepoImpl) RollbackStockCache(ctx context.Context, productID int64, count int32) error {
	stockKey := fmt.Sprintf("{stock}:%d", productID)
	return r.rdb.IncrBy(ctx, stockKey, int64(count)).Err()
}

/*
CreateOrderTx 创建订单事务

事务流程：
1. 扣减 MySQL 真实库存（乐观锁兜底）
2. 插入订单记录（含过期时间）
*/
func (r *orderRepoImpl) CreateOrderTx(ctx context.Context, order *repo.Order, expireTime time.Time, count int32) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 扣减库存（乐观锁兜底）
		// 使用 UPDATE ... WHERE stock_num >= ? 确保不会超卖
		res := tx.Exec(`
            UPDATE stock
            SET stock_num = stock_num - ?,
                version = version + 1
            WHERE product_id = ? AND stock_num >= ?`,
			count, order.ProductID, count)

		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("数据库落库失败: 库存不足或发生并发冲突")
		}

		// 2. 写入订单（含过期时间）
		order.ExpireTime = &expireTime
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		return nil
	})
}

/*
CreateOrderWithOutboxTx 创建订单事务（含 Outbox 本地消息表）

在单个 MySQL 事务中完成：
1. 扣减 MySQL 库存（乐观锁兜底）
2. 插入订单记录
3. 插入 Outbox 出站消息（用于后续 MQ 投递）

原子性保证：订单创建和消息写入是原子的，绝不会出现"订单创建了但消息没写"的情况。
Outbox Worker 会异步读取并投递消息到 MQ。
*/
func (r *orderRepoImpl) CreateOrderWithOutboxTx(ctx context.Context, order *repo.Order, expireTime time.Time, count int32, outboxRecords []*repo.OutboxRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 扣减库存（乐观锁兜底）
		res := tx.Exec(`
            UPDATE stock
            SET stock_num = stock_num - ?,
                version = version + 1
            WHERE product_id = ? AND stock_num >= ?`,
			count, order.ProductID, count)

		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("数据库落库失败: 库存不足或发生并发冲突")
		}

		// 2. 写入订单
		order.ExpireTime = &expireTime
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// 3. 写入 Outbox 出站消息
		if len(outboxRecords) > 0 {
			if err := tx.Create(&outboxRecords).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetOrderByNo 根据订单号查询订单
func (r *orderRepoImpl) GetOrderByNo(ctx context.Context, orderNo string) (*repo.Order, error) {
	var order repo.Order
	err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repo.ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

// UpdateOrderStatus 更新订单状态
func (r *orderRepoImpl) UpdateOrderStatus(ctx context.Context, orderNo string, status int8) error {
	result := r.db.WithContext(ctx).Model(&repo.Order{}).Where("order_no = ?", orderNo).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrOrderNotFound
	}
	return nil
}

// ListOrdersByUser 分页查询用户的订单列表
func (r *orderRepoImpl) ListOrdersByUser(ctx context.Context, userID int64, page, pageSize int32, status int8) ([]*repo.Order, int32, error) {
	var orders []*repo.Order
	var total int64

	query := r.db.WithContext(ctx).Model(&repo.Order{}).Where("user_id = ?", userID)
	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(int(offset)).Limit(int(pageSize)).Order("create_time DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, int32(total), nil
}

/*
CancelOrderTx 取消订单事务

适用场景：用户主动取消订单
事务流程：
1. 查询并校验订单归属和状态
2. 更新订单状态为已取消
3. 回滚库存
*/
func (r *orderRepoImpl) CancelOrderTx(ctx context.Context, orderNo string, userID int64, productID int64, count int32) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 查询订单确认归属
		var order repo.Order
		if err := tx.Where("order_no = ? AND user_id = ?", orderNo, userID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repo.ErrOrderNotFound
			}
			return err
		}

		// 2. 检查订单状态，只有待支付才能取消
		if order.Status != 0 {
			return repo.ErrOrderStatusInvalid
		}

		// 3. 更新订单状态为已取消
		if err := tx.Model(&repo.Order{}).Where("order_no = ?", orderNo).
			Update("status", repo.OrderStatusCancelled).Error; err != nil {
			return err
		}

		// 4. 回滚库存
		res := tx.Exec(`
            UPDATE stock
            SET stock_num = stock_num + ?,
                version = version + 1
            WHERE product_id = ?`,
			count, productID)
		if res.Error != nil {
			return res.Error
		}

		return nil
	})
}

/*
TimeoutOrderTx 超时取消订单事务

适用场景：
1. 延迟队列消息触发（30分钟后）
2. 定时扫描兜底任务触发

事务流程：
1. 查询订单当前状态
2. 仅对待支付订单执行取消（防止重复处理）
3. 更新状态为已超时
4. 回滚库存
5. 记录超时日志（可选）

注意：此方法设计为幂等，多次执行结果一致
*/
func (r *orderRepoImpl) TimeoutOrderTx(ctx context.Context, orderNo string) (bool, error) {
	var order repo.Order
	transitioned := false

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repo.ErrOrderNotFound
			}
			return err
		}

		if repo.OrderStatus(order.Status) != repo.OrderStatusPending {
			return nil
		}

		result := tx.Model(&repo.Order{}).
			Where("order_no = ? AND status = ?", orderNo, repo.OrderStatusPending).
			Update("status", repo.OrderStatusTimeout)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		result = tx.Exec(`
            UPDATE stock
            SET stock_num = stock_num + ?,
                version = version + 1
            WHERE product_id = ?`,
			order.Count, order.ProductID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("restore MySQL stock: product %d not found", order.ProductID)
		}

		transitioned = true
		return nil
	})
	if err != nil {
		return false, err
	}

	// 条件更新的败者重读最终状态，仍可以幂等补做 Redis 回补。
	if !transitioned && repo.OrderStatus(order.Status) == repo.OrderStatusPending {
		if err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, repo.ErrOrderNotFound
			}
			return false, err
		}
	}
	if repo.OrderStatus(order.Status) != repo.OrderStatusTimeout && !transitioned {
		return false, nil
	}
	if order.StockCacheRestored {
		return false, nil
	}

	// 两个 key 使用相同 Redis Cluster hash tag，保证 Lua 可在集群模式执行。
	markerKey := fmt.Sprintf("{stock}:timeout-restored:%s", orderNo)
	stockKey := fmt.Sprintf("{stock}:%d", order.ProductID)
	result, err := restoreTimeoutStockScript.Run(
		ctx,
		r.rdb,
		[]string{markerKey, stockKey},
		order.Count,
		int64(timeoutStockDedupTTL/time.Second),
	).Int()
	if err != nil {
		return transitioned, fmt.Errorf("restore Redis stock: %w", err)
	}
	if result < 0 {
		return transitioned, fmt.Errorf("restore Redis stock: key %s contains a non-integer value", stockKey)
	}

	// Redis 成功后再持久化完成标记。若此处失败，下次扫描会重试，Lua 标记会防止重复加库存。
	markResult := r.db.WithContext(ctx).Model(&repo.Order{}).
		Where("order_no = ? AND status = ?", orderNo, repo.OrderStatusTimeout).
		Update("stock_cache_restored", true)
	if markResult.Error != nil {
		return transitioned, fmt.Errorf("mark Redis stock compensation completed: %w", markResult.Error)
	}
	if markResult.RowsAffected == 0 {
		return transitioned, fmt.Errorf("mark Redis stock compensation completed: order %s is no longer timed out", orderNo)
	}

	return transitioned, nil
}

/*
ListTimeoutOrders 批量查询待超时订单

用于定时扫描兜底机制

查询条件：
1. status = 0 (待支付)
2. expire_time < NOW() (已过过期时间)

排序：按过期时间升序（先超时的先处理）
限制：每次最多处理 N 条（防止锁表）
*/
func (r *orderRepoImpl) ListTimeoutOrders(ctx context.Context, limit int32) ([]*repo.Order, error) {
	var orders []*repo.Order

	err := r.db.WithContext(ctx).Model(&repo.Order{}).
		Where("(status = ? AND expire_time IS NOT NULL AND expire_time < ?) OR (status = ? AND stock_cache_restored = ?)",
			repo.OrderStatusPending, time.Now(), repo.OrderStatusTimeout, false).
		Order("expire_time ASC").
		Limit(int(limit)).
		Find(&orders).Error

	if err != nil {
		return nil, err
	}

	return orders, nil
}
