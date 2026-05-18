package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/outbox"
	"ecommerce-demo/app/order/internal/repo"
	"ecommerce-demo/app/order/internal/server"
	"ecommerce-demo/app/order/internal/svc"
	"ecommerce-demo/app/order/pb"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var configFile = flag.String("f", "etc/order.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	// 启动 Outbox Worker（替代旧的直接 MQ 投递，保证消息不丢）
	outboxWorker := outbox.NewWorker(ctx.OutboxRepo, ctx.Producer, c)
	outboxWorker.Start()
	defer outboxWorker.Stop()

	// 启动 MQ 消费者（异步落库消息的消费端）
	mqConsumer := ctx.StartMQConsumer()
	if mqConsumer != nil {
		mqConsumer.Start()
		defer mqConsumer.Stop()
	}

	// 启动超时订单处理 Worker（定时扫描兜底）
	go startTimeoutWorker(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterOrderServer(grpcServer, server.NewOrderServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting Order RPC server (with Outbox Pattern) at %s...\n", c.ListenOn)
	s.Start()
}

// startTimeoutWorker 启动超时订单处理 Worker（兜底机制，处理 Outbox 延迟消息未覆盖的场景）
func startTimeoutWorker(c config.Config) {
	db, err := gorm.Open(mysql.Open(c.DataSource), &gorm.Config{})
	if err != nil {
		log.Printf("[OrderTimeoutWorker] MySQL 初始化失败: %v", err)
		return
	}

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{c.RedisConf.Host},
		Password: c.RedisConf.Pass,
	})

	worker := &orderTimeoutWorker{
		db:    db,
		redis: rdb,
	}

	scanInterval := c.OrderTimeout.ScanIntervalSeconds
	if scanInterval <= 0 {
		scanInterval = 60
	}
	log.Printf("[OrderTimeoutWorker] 超时订单处理worker已启动，每 %d 秒扫描一次", scanInterval)

	ticker := time.NewTicker(time.Duration(scanInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		worker.processTimeoutOrders()
	}
}

type orderTimeoutWorker struct {
	db    *gorm.DB
	redis *redis.ClusterClient
}

func (w *orderTimeoutWorker) processTimeoutOrders() {
	ctx := context.Background()

	var orders []repo.Order
	if err := w.db.WithContext(ctx).
		Where("status = ? AND expire_time IS NOT NULL AND expire_time < ?",
			repo.OrderStatusPending, time.Now()).
		Order("expire_time ASC").
		Limit(100).
		Find(&orders).Error; err != nil {
		log.Printf("[OrderTimeoutWorker] 查询超时订单失败: %v", err)
		return
	}

	if len(orders) == 0 {
		return
	}

	log.Printf("[OrderTimeoutWorker] 发现 %d 个超时订单", len(orders))

	for _, order := range orders {
		lockKey := fmt.Sprintf("order:timeout:lock:%s", order.OrderNo)
		success, err := w.redis.SetNX(ctx, lockKey, "1", 5*time.Minute).Result()
		if err != nil {
			log.Printf("[OrderTimeoutWorker] 获取分布式锁失败: %v", err)
			continue
		}
		if !success {
			continue
		}

		if err := w.cancelTimeoutOrder(ctx, &order); err != nil {
			log.Printf("[OrderTimeoutWorker] 取消超时订单 %s 失败: %v", order.OrderNo, err)
		} else {
			log.Printf("[OrderTimeoutWorker] 成功取消超时订单: %s", order.OrderNo)
		}

		w.redis.Del(ctx, lockKey)
	}
}

func (w *orderTimeoutWorker) cancelTimeoutOrder(ctx context.Context, order *repo.Order) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&repo.Order{}).
			Where("order_no = ? AND status = ?", order.OrderNo, repo.OrderStatusPending).
			Update("status", repo.OrderStatusTimeout).Error; err != nil {
			return err
		}

		result := tx.Exec(
			"UPDATE stock SET stock_num = stock_num + ?, version = version + 1 WHERE product_id = ?",
			order.Count, order.ProductID,
		)
		if result.Error != nil {
			return result.Error
		}

		stockKey := fmt.Sprintf("{stock}:%d", order.ProductID)
		w.redis.IncrBy(ctx, stockKey, int64(order.Count))

		return nil
	})
}
