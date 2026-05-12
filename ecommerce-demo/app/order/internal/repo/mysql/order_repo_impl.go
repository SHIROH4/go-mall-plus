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

type orderRepoImpl struct {
    db  *gorm.DB
    rdb *redis.Client
}

func NewOrderRepo(db *gorm.DB, rdb *redis.Client) repo.OrderRepo {
    return &orderRepoImpl{db: db, rdb: rdb}
}

// DeductStockCache 执行 Lua 脚本原子扣减库存
func (r *orderRepoImpl) DeductStockCache(ctx context.Context, productID int64, count int32) (bool, error) {
    stockKey := fmt.Sprintf("stock:%d", productID)
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
    stockKey := fmt.Sprintf("stock:%d", productID)
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
func (r *orderRepoImpl) TimeoutOrderTx(ctx context.Context, orderNo string, productID int64, count int32) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 1. 查询订单状态
        var order repo.Order
        if err := tx.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                // 订单不存在，可能是用户已取消或已支付，跳过
                return nil
            }
            return err
        }

        // 2. 幂等检查：只有待支付订单才能超时取消
        // 已支付/已取消/已超时的订单直接跳过
        if order.Status != 0 {
            return nil
        }

        // 3. 更新状态为已超时
        if err := tx.Model(&repo.Order{}).Where("order_no = ?", orderNo).
            Update("status", repo.OrderStatusTimeout).Error; err != nil {
            return err
        }

        // 4. 回滚库存（Redis 和 MySQL 都回滚）
        // 4.1 回滚 Redis 库存（同步执行，失败则记录日志但不阻塞事务）
        stockKey := fmt.Sprintf("stock:%d", productID)
        if err := r.rdb.IncrBy(ctx, stockKey, int64(count)).Err(); err != nil {
            // Redis 回滚失败不影响主流程，后续有补偿机制
            fmt.Printf("⚠️ Redis 库存回滚失败, OrderNo: %s, ProductID: %d, Err: %v\n",
                orderNo, productID, err)
        }

        // 4.2 回滚 MySQL 库存
        res := tx.Exec(`
            UPDATE stock
            SET stock_num = stock_num + ?,
                version = version + 1
            WHERE product_id = ?`,
            count, productID)
        if res.Error != nil {
            return res.Error
        }

        // 5. 可选：记录超时日志到单独表（用于运营分析）
        // tx.Exec("INSERT INTO order_timeout_log (...) VALUES (...)", orderNo, ...)

        return nil
    })
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
        Where("status = ? AND expire_time IS NOT NULL AND expire_time < ?",
            repo.OrderStatusPending, time.Now()).
        Order("expire_time ASC").
        Limit(int(limit)).
        Find(&orders).Error

    if err != nil {
        return nil, err
    }

    return orders, nil
}
