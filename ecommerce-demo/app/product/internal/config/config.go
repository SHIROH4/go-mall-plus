package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf

	//MYSQL数据源
	DataSource string

	//Redis配置(直接复用 go-zero 预设的Redis 配置结构)
	RedisConf struct {
		Host string
		Type string
		Pass string
	}
}
