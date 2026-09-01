package main

import (
	"flag"
	"fmt"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/outbox"
	"ecommerce-demo/app/order/internal/server"
	"ecommerce-demo/app/order/internal/svc"
	"ecommerce-demo/app/order/pb"
	"ecommerce-demo/common/metrics"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/order.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)
	stopMetrics := metrics.StartServer(":9091")
	defer stopMetrics()

	// 启动 Outbox Worker（替代旧的直接 MQ 投递，保证消息不丢）
	outboxWorker := outbox.NewWorker(ctx.OutboxRepo, ctx.Producer, c)
	outboxWorker.Start()
	defer outboxWorker.Stop()

	// 启动订单创建事件消费者（只读校验，不重复创建订单或扣减库存）
	mqConsumer := ctx.StartMQConsumer()
	if mqConsumer != nil {
		mqConsumer.Start()
		defer mqConsumer.Stop()
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterOrderServer(grpcServer, server.NewOrderServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting Order RPC server (with Outbox Pattern) at %s...\n", c.ListenOn)
	s.Start()
}
