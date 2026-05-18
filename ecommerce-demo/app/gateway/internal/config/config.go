package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	Auth struct {
		PrivateKeyPath string
		PublicKeyPath  string
		AccessExpire   int64
		RefreshExpire  int64
	}

	RedisConf struct {
		Host string
		Type string
		Pass string
		DB   int
	}

	MysqlConf struct {
		Host     string
		User     string
		Password string
		Database string
	}

	// 限流配置
	RateLimit struct {
		Enabled bool
		Global  struct {
			Rate  int
			Burst int
		}
		PerIP struct {
			Rate  int
			Burst int
		}
		Routes map[string]struct {
			Rate  int
			Burst int
		}
	}

	// CORS 配置
	CORS struct {
		Enabled        bool
		AllowedOrigins []string
		AllowedMethods []string
		AllowedHeaders []string
		MaxAge         int
	}

	// 断路器配置（应用于 RPC 客户端）
	CircuitBreaker struct {
		Enabled   bool
		Threshold int // 触发熔断的失败率阈值（百分比，默认 50）
	}

	// RPC 客户端发现配置
	UserRpcConf    zrpc.RpcClientConf
	ProductRpcConf zrpc.RpcClientConf
	OrderRpcConf   zrpc.RpcClientConf
	StockRpcConf   zrpc.RpcClientConf // 库存 RPC
	CartRpcConf    zrpc.RpcClientConf // 购物车 RPC
	PaymentRpcConf zrpc.RpcClientConf // 支付 RPC
	AddressRpcConf zrpc.RpcClientConf // 地址 RPC
}
