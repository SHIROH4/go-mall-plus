package svc

import (
	"context"
	"log"

	"ecommerce-demo/app/stock/internal/config"
	"ecommerce-demo/app/stock/internal/repo"
	"ecommerce-demo/app/stock/internal/repo/mysql"
	"ecommerce-demo/app/stock/internal/service"

	"github.com/redis/go-redis/v9"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config       config.Config
	StockRepo    repo.StockRepo
	StockService service.StockService
	RDB          *redis.ClusterClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(gormMysql.Open(c.DataSource), &gorm.Config{})
	if err != nil {
		log.Fatalf("Stock 服务 MySQL 初始化失败: %v", err)
	}

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{c.RedisConf.Host},
		Password: c.RedisConf.Pass,
	})

	// 预热 Redis 连接
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("⚠️ Stock 服务 Redis 连接失败(非致命): %v", err)
	} else {
		log.Println("Stock 服务 Redis 连接就绪")
	}

	stockRepo := mysql.NewStockRepo(db, rdb)

	// 启动时同步库存到 Redis
	if err := stockRepo.InitStockCache(context.Background()); err != nil {
		log.Printf("⚠️ Stock 服务启动时库存缓存初始化失败(非致命): %v", err)
	}

	stockService := service.NewStockService(stockRepo)

	return &ServiceContext{
		Config:       c,
		StockRepo:    stockRepo,
		StockService: stockService,
		RDB:          rdb,
	}
}
