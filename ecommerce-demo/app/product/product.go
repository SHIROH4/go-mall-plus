package main

import (
	"context"
	"flag"
	"fmt"

	"ecommerce-demo/app/product/internal/config"
	"ecommerce-demo/app/product/internal/server"
	"ecommerce-demo/app/product/internal/svc"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/product.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	// 初始化库存缓存（将数据库中的库存同步到 Redis，供订单服务使用）
	if err := ctx.ProductRepo.InitStockCache(context.Background()); err != nil {
		fmt.Printf("⚠️ 库存缓存初始化失败: %v\n", err)
		// 不退出服务，容错处理
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterProductServer(grpcServer, server.NewProductServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
