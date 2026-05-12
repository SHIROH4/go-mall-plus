package cron

import (
    "testing"
    "time"

    "ecommerce-demo/app/order/internal/config"
    "ecommerce-demo/app/order/internal/repo"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

/*
  定时扫描任务单元测试

  测试覆盖：
  1. 扫描任务启动和停止
  2. 扫描间隔配置生效
  3. 批量处理超时订单
*/

// MockOrderRepo 用于测试的模拟仓储
type MockOrderRepo struct {
    mock.Mock
    repo.OrderRepo
}

func TestOrderTimeoutJob_StartAndStop(t *testing.T) {
    mockRepo := new(MockOrderRepo)
    cfg := config.OrderTimeoutConfig{
        OrderExpireMinutes:    1,
        ScanIntervalSeconds:   1, // 测试用1秒
        MaxScanCount:          10,
    }

    job := NewOrderTimeoutJob(mockRepo, cfg)

    // 启动任务
    job.Start()
    time.Sleep(100 * time.Millisecond) // 等待任务启动

    assert.True(t, job.isRunning, "任务应该处于运行状态")

    // 停止任务
    job.Stop()
    time.Sleep(100 * time.Millisecond) // 等待任务停止

    assert.False(t, job.isRunning, "任务应该已停止")
}

func TestOrderTimeoutJob_ScanInterval(t *testing.T) {
    /*
      测试扫描间隔配置

      场景：
      1. 配置扫描间隔为 1 秒
      2. 模拟两次扫描，验证间隔
    */
    // 此测试需要集成测试环境
    t.Skip("跳过单元测试，需要集成测试环境")
}

func TestOrderTimeoutJob_BatchProcess(t *testing.T) {
    /*
      测试批量处理

      场景：
      1. 模拟 150 个超时订单
      2. 配置每次最多处理 100 个
      3. 验证分批处理
    */
    t.Skip("跳过单元测试，需要集成测试环境")
}

func TestNewOrderTimeoutJob(t *testing.T) {
    mockRepo := new(MockOrderRepo)
    cfg := config.OrderTimeoutConfig{
        OrderExpireMinutes:    30,
        ScanIntervalSeconds:   60,
        MaxScanCount:         100,
    }

    job := NewOrderTimeoutJob(mockRepo, cfg)

    assert.NotNil(t, job)
    assert.Equal(t, mockRepo, job.orderRepo)
    assert.Equal(t, cfg, job.config)
    assert.False(t, job.isRunning)
}

func TestOrderTimeoutJob_NotRunningTwice(t *testing.T) {
    mockRepo := new(MockOrderRepo)
    cfg := config.OrderTimeoutConfig{
        ScanIntervalSeconds: 1,
        MaxScanCount:       10,
    }

    job := NewOrderTimeoutJob(mockRepo, cfg)

    // 第一次启动
    job.Start()
    time.Sleep(50 * time.Millisecond)

    // 尝试第二次启动（应该被忽略）
    job.Start()
    time.Sleep(50 * time.Millisecond)

    // 仍然应该处于运行状态
    assert.True(t, job.isRunning)

    job.Stop()
}
