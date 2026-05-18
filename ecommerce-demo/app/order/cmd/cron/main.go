package main

import (
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"

    "ecommerce-demo/app/order/internal/config"
    "ecommerce-demo/app/order/internal/cron"
    "ecommerce-demo/app/order/internal/repo/mysql"

    "github.com/redis/go-redis/v9"
    "github.com/zeromicro/go-zero/core/conf"
    gormMysql "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

/*
  定时扫描任务独立启动入口

  用途：独立部署定时扫描任务，作为延迟队列的兜底机制

  启动方式：
  go run cron.go -f etc/order.yaml

  职责：
  1. 连接 MySQL
  2. 启动超时扫描定时任务
  3. 监听信号，优雅退出
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

    // 初始化 Redis（用于库存回滚）
    rdb := redis.NewClusterClient(&redis.ClusterOptions{
        Addrs:    []string{c.RedisConf.Host},
        Password: c.RedisConf.Pass,
    })

    // 初始化 Order Repo
    orderRepo := mysql.NewOrderRepo(db, rdb)

    // 初始化并启动超时扫描任务
    timeoutJob := cron.NewOrderTimeoutJob(orderRepo, c.OrderTimeout)
    timeoutJob.Start()

    log.Printf("🚀 定时扫描任务已启动，PID: %d", os.Getpid())

    // 等待退出信号
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    log.Println("🛑 收到退出信号，正在停止...")
    timeoutJob.Stop()
    log.Println("👋 定时扫描任务已退出")
}
