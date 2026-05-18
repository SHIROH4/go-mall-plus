package main

import (
	"flag"
	"fmt"

	"ecommerce-demo/app/stock/internal/config"
	"ecommerce-demo/app/stock/internal/server"
	"ecommerce-demo/app/stock/internal/svc"
	"ecommerce-demo/app/stock/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/stock.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterStockServer(grpcServer, server.NewStockServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting Stock RPC server at %s...\n", c.ListenOn)
	s.Start()
}
