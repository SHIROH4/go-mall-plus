package main

import (
	"flag"
	"fmt"

	"ecommerce-demo/app/payment/internal/config"
	"ecommerce-demo/app/payment/internal/server"
	"ecommerce-demo/app/payment/internal/svc"
	"ecommerce-demo/app/payment/pb"
	orderclient "ecommerce-demo/app/order/order"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/payment.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 初始化 Order RPC 客户端（用于支付成功后通知订单）
	var orderRpc orderclient.Order
	if c.OrderRpcConf.Endpoints != nil {
		orderRpc = orderclient.NewOrder(zrpc.MustNewClient(c.OrderRpcConf))
	}

	// 初始化服务上下文
	svcCtx := svc.NewServiceContext(c, orderRpc)

	// 创建 gRPC Server
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterPaymentServer(grpcServer, server.NewPaymentServer(svcCtx))

		// 开发/测试模式下启用 gRPC 反射
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting Payment RPC Server on %s...\n", c.ListenOn)
	s.Start()
}
