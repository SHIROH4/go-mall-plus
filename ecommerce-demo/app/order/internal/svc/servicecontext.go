package svc

import (
	"log"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/mq"
	"ecommerce-demo/app/order/internal/repo"
	"ecommerce-demo/app/order/internal/repo/mysql"
	"ecommerce-demo/app/order/internal/service"
	productclient "ecommerce-demo/app/product/product"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config       config.Config
	OrderRepo    repo.OrderRepo
	OrderService service.OrderService
	Producer     mq.Producer
	rdb          *redis.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(gormMysql.Open(c.DataSource), &gorm.Config{})
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     c.RedisConf.Host,
		Password: c.RedisConf.Pass,
	})

	productRpc := productclient.NewProduct(zrpc.MustNewClient(c.ProductRpcConf))

	orderRepo := mysql.NewOrderRepo(db, rdb)

	producer := mq.NewRabbitProducer(c)

	orderService := service.NewOrderService(orderRepo, productRpc, producer, c.OrderTimeout)

	return &ServiceContext{
		Config:       c,
		OrderRepo:    orderRepo,
		OrderService: orderService,
		Producer:     producer,
		rdb:          rdb,
	}
}

// StartMQConsumer 启动 MQ 消费者
func (s *ServiceContext) StartMQConsumer() mq.Consumer {
	consumer, err := mq.NewReliableConsumer(s.Config, s.OrderRepo, s.rdb)
	if err != nil {
		log.Printf("⚠️ MQ消费者初始化失败: %v", err)
		return nil
	}
	return consumer
}
