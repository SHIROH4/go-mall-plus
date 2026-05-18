package svc

import (
	"ecommerce-demo/app/cart/internal/config"
	"ecommerce-demo/app/cart/internal/repo/redis"
	"ecommerce-demo/app/cart/internal/service"

	goredis "github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config      config.Config
	CartService service.CartService
}

func NewServiceContext(c config.Config, rdb *goredis.ClusterClient) *ServiceContext {
	cartRepo := redis.NewCartRedisRepo(rdb)
	cartService := service.NewCartService(cartRepo)

	return &ServiceContext{
		Config:      c,
		CartService: cartService,
	}
}
