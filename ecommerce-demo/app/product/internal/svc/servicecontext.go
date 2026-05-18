package svc

import (
	"log"

	"ecommerce-demo/app/product/internal/config"
	"ecommerce-demo/app/product/internal/repo"
	"ecommerce-demo/app/product/internal/repo/mysql"
	"ecommerce-demo/app/product/internal/service"

	"github.com/redis/go-redis/v9"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config         config.Config
	ProductRepo    repo.ProductRepo
	CategoryRepo   repo.CategoryRepo
	ProductService service.ProductService
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 1. 初始化 MySQL
	db, err := gorm.Open(gormMysql.Open(c.DataSource), &gorm.Config{})
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	// 2. 初始化 Redis 客户端
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{c.RedisConf.Host},
		Password: c.RedisConf.Pass,
	})

	// 3. 注入仓储层 (同时传入 db 和 rdb)
	productRepo := mysql.NewProductRepo(db, rdb)
	categoryRepo := mysql.NewCategoryRepo(db)

	// 4. 注入业务层
	productService := service.NewProductService(productRepo, categoryRepo)

	return &ServiceContext{
		Config:         c,
		ProductRepo:    productRepo,
		CategoryRepo:   categoryRepo,
		ProductService: productService,
	}
}
