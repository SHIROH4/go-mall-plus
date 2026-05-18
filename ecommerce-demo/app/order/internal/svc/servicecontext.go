package svc

import (
	"log"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/mq"
	"ecommerce-demo/app/order/internal/repo"
	"ecommerce-demo/app/order/internal/repo/mysql"
	"ecommerce-demo/app/order/internal/service"
	productclient "ecommerce-demo/app/product/product"
	stockclient "ecommerce-demo/app/stock/stock"
	"ecommerce-demo/common/metrics/rpcmetrics"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config       config.Config
	OrderRepo    repo.OrderRepo
	OutboxRepo   repo.OutboxRepo
	OrderService service.OrderService
	Producer     mq.Producer
	RDB          *redis.ClusterClient
	DB           *gorm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(gormMysql.Open(c.DataSource), &gorm.Config{})
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{c.RedisConf.Host},
		Password: c.RedisConf.Pass,
	})

	productRpc := productclient.NewProduct(zrpc.MustNewClient(c.ProductRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor())))
	stockRpc := stockclient.NewStock(zrpc.MustNewClient(c.StockRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor())))

	orderRepo := mysql.NewOrderRepo(db, rdb)
	outboxRepo := mysql.NewOutboxRepo(db)

	producer := mq.NewRabbitProducer(c)

	orderService := service.NewOrderService(
		orderRepo, outboxRepo, productRpc, stockRpc, producer, c.OrderTimeout,
	)

	return &ServiceContext{
		Config:       c,
		OrderRepo:    orderRepo,
		OutboxRepo:   outboxRepo,
		OrderService: orderService,
		Producer:     producer,
		RDB:          rdb,
		DB:           db,
	}
}

// StartMQConsumer 启动 MQ 消费者
func (s *ServiceContext) StartMQConsumer() mq.Consumer {
	consumer, err := mq.NewReliableConsumer(s.Config, s.OrderRepo, s.RDB)
	if err != nil {
		log.Printf("MQ消费者初始化失败: %v", err)
		return nil
	}
	return consumer
}
