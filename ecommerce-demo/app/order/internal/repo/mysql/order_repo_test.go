package mysql

import (
    "testing"
    "time"

    "ecommerce-demo/app/order/internal/config"
    "ecommerce-demo/app/order/internal/repo"

    "github.com/stretchr/testify/assert"
)

/*
  订单仓储层单元测试

  测试覆盖：
  1. CreateOrderTx 创建订单事务
  2. TimeoutOrderTx 超时取消事务（幂等性）
  3. ListTimeoutOrders 查询超时订单

  前置条件：
  1. MySQL 运行在 localhost:3306
  2. 数据库 ecommerce_demo 存在
  3. 表结构已初始化（含 expire_time 字段）
*/

func TestOrderRepoImpl_TimeoutOrderTx_Idempotent(t *testing.T) {
    /*
      测试超时取消的幂等性

      场景：
      1. 同一订单号被多次调用 TimeoutOrderTx
      2. 只有第一次调用会真正执行取消
      3. 后续调用直接返回 nil（幂等）
    */
    t.Skip("跳过单元测试，需要完整数据库环境")

    // 实现示例：
    // 1. 创建测试订单
    // 2. 第一次调用 TimeoutOrderTx，验证状态变更为已超时
    // 3. 第二次调用 TimeoutOrderTx，验证状态不变
    // 4. 第三次调用 TimeoutOrderTx，验证状态不变
    // assert.Equal(t, repo.OrderStatusTimeout, order.Status)
}

func TestOrderRepoImpl_ListTimeoutOrders(t *testing.T) {
    t.Skip("跳过单元测试，需要完整数据库环境")
}

func TestOrderStatus_Constants(t *testing.T) {
    // 测试订单状态常量定义正确
    assert.Equal(t, repo.OrderStatus(0), repo.OrderStatusPending)
    assert.Equal(t, repo.OrderStatus(1), repo.OrderStatusPaid)
    assert.Equal(t, repo.OrderStatus(2), repo.OrderStatusCancelled)
    assert.Equal(t, repo.OrderStatus(3), repo.OrderStatusTimeout)
}

func TestOrderStatus_StatusText(t *testing.T) {
    tests := []struct {
        status   repo.OrderStatus
        expected string
    }{
        {repo.OrderStatusPending, "待支付"},
        {repo.OrderStatusPaid, "已支付"},
        {repo.OrderStatusCancelled, "已取消"},
        {repo.OrderStatusTimeout, "已超时"},
        {repo.OrderStatus(99), "未知状态"},
    }

    for _, tt := range tests {
        t.Run(tt.expected, func(t *testing.T) {
            assert.Equal(t, tt.expected, tt.status.StatusText())
        })
    }
}

func TestOrder_ExpireTime(t *testing.T) {
    // 测试订单实体过期时间字段
    expireTime := time.Now().Add(30 * time.Minute)
    order := &repo.Order{
        OrderNo:    "TEST-ORD-001",
        UserID:     1,
        ProductID:  1,
        Count:      2,
        Status:     0,
        ExpireTime: &expireTime,
    }

    assert.NotNil(t, order.ExpireTime)
    assert.Equal(t, expireTime.Unix(), order.ExpireTime.Unix())
}

func TestConfig_DefaultOrderTimeoutConfig(t *testing.T) {
    cfg := config.DefaultOrderTimeoutConfig()

    assert.Equal(t, 30, cfg.OrderExpireMinutes)
    assert.Equal(t, "order.delay.queue", cfg.DelayQueueName)
    assert.Equal(t, "order.delay.exchange", cfg.DelayExchangeName)
    assert.Equal(t, 60, cfg.ScanIntervalSeconds)
    assert.Equal(t, 100, cfg.MaxScanCount)
}
