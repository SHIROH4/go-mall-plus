package main

import (
	"flag"
	"fmt"

	"ecommerce-demo/app/cart/internal/config"
	cartredis "ecommerce-demo/app/cart/internal/repo/redis"
	"ecommerce-demo/app/cart/internal/server"
	cartsvc "ecommerce-demo/app/cart/internal/service"
	"ecommerce-demo/app/cart/internal/svc"
	cartpb "ecommerce-demo/app/cart/pb"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/cart.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 初始化 Redis 客户端
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     c.RedisConf.Host,
		Password: c.RedisConf.Pass,
		DB:       0,
	})

	// 初始化依赖
	cartRepo := cartredis.NewCartRedisRepo(rdb)
	cartService := cartsvc.NewCartService(cartRepo)
	svcCtx := svc.NewServiceContext(c, rdb)
	svcCtx.CartService = cartService

	// 创建 gRPC Server
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		cartpb.RegisterCartServer(grpcServer, server.NewCartServer(svcCtx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting Cart RPC Server on %s...\n", c.ListenOn)
	s.Start()
}
