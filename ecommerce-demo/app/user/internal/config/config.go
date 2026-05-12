package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	// 新增映射user.yaml里的DataSource
	DataSource string
}
