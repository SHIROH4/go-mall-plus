package main

import (
	"flag"
	"fmt"

	"ecommerce-demo/app/address/internal/config"
	"ecommerce-demo/app/address/internal/server"
	"ecommerce-demo/app/address/internal/svc"
	"ecommerce-demo/app/address/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/address.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	svcCtx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterAddressServer(grpcServer, server.NewAddressServer(svcCtx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting Address RPC Server on %s...\n", c.ListenOn)
	s.Start()
}
