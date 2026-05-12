// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

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

	// RPC 客户端发现配置
	UserRpcConf    zrpc.RpcClientConf
	ProductRpcConf zrpc.RpcClientConf
	OrderRpcConf   zrpc.RpcClientConf
	CartRpcConf    zrpc.RpcClientConf // 购物车 RPC
	PaymentRpcConf zrpc.RpcClientConf // 支付 RPC
	AddressRpcConf zrpc.RpcClientConf // 地址 RPC
}
