package main

import (
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"

    "ecommerce-demo/app/order/internal/config"
    "ecommerce-demo/app/order/internal/mq"
    "ecommerce-demo/app/order/internal/repo/mysql"

    "github.com/redis/go-redis/v9"
    "github.com/zeromicro/go-zero/core/conf"
    gormMysql "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

/*
  延迟消费者独立启动入口（可靠性增强版）

  用途：独立部署延迟消费者，专门处理超时订单

  启动方式：
  go run main.go -f etc/order.yaml

  可靠性保障：
  1. 手动ACK确认 - 确保消息处理完成后才确认
  2. 重试次数限制 - 超过3次进入死信队列
  3. 死信队列(DLQ) - 永久失败消息存储，供人工处理
  4. 优雅退出 - 服务停止时等待处理完成
  5. 消息追踪 - MessageID用于问题排查
*/

var configFile = flag.String("f", "etc/order.yaml", "the config file")

func main() {
    flag.Parse()

    var c config.Config
    conf.MustLoad(*configFile, &c)

    // 初始化 MySQL
    db, err := gorm.Open(gormMysql.Open(c.DataSource), &gorm.Config{})
    if err != nil {
        log.Fatalf("MySQL 初始化失败: %v", err)
    }

    // 初始化 Redis
    rdb := redis.NewClusterClient(&redis.ClusterOptions{
        Addrs:    []string{c.RedisConf.Host},
        Password: c.RedisConf.Pass,
    })

    // 初始化 Order Repo
    orderRepo := mysql.NewOrderRepo(db, rdb)

    // 初始化可靠延迟消费者
    consumer, err := mq.NewReliableDelayConsumer(c, orderRepo, rdb)
    if err != nil {
        log.Fatalf("延迟消费者初始化失败: %v", err)
    }

    // 启动延迟消费者
    consumer.Start()

    log.Printf("🚀 延迟消费者服务已启动，PID: %d", os.Getpid())
    log.Printf("📋 配置信息: 预取数=%d，最大重试=%d，DLQ启用=%v",
        c.MQConsumer.PrefetchCount, c.MQConsumer.MaxRetryTimes, c.MQConsumer.EnableDLQ)

    // 等待退出信号
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    log.Println("🛑 收到退出信号，正在停止...")
    consumer.Stop()
    log.Println("👋 延迟消费者服务已退出")
}
