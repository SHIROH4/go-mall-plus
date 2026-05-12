package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	MySQLConf struct {
		DataSource string
	}

	RedisConf struct {
		Host string
		Type string
		Pass string
		DB   int
	}

	OrderRpcConf zrpc.RpcClientConf

	PayExpireMinutes int // 支付过期时间(分钟)，默认30分钟
}
