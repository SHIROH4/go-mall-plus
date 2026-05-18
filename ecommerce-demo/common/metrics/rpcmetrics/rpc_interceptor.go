package rpcmetrics

import (
	"context"
	"strings"
	"time"

	"ecommerce-demo/common/metrics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor 返回一个 gRPC 客户端拦截器，自动记录 RPC 调用指标
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		service, m := splitMethod(method)
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		st := status.Code(err).String()
		metrics.RPCCallTotal.WithLabelValues(service, m, st).Inc()
		metrics.RPCCallDuration.WithLabelValues(service, m).Observe(time.Since(start).Seconds())

		return err
	}
}

// UnaryServerInterceptor 返回一个 gRPC 服务端拦截器
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		service, m := splitMethod(info.FullMethod)
		start := time.Now()

		resp, err := handler(ctx, req)

		st := status.Code(err).String()
		metrics.RPCCallTotal.WithLabelValues(service, m, st).Inc()
		metrics.RPCCallDuration.WithLabelValues(service, m).Observe(time.Since(start).Seconds())

		return resp, err
	}
}

func splitMethod(fullMethod string) (service, method string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.SplitN(fullMethod, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "unknown", fullMethod
}
