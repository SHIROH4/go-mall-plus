package config

import (
    "github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
    zrpc.RpcServerConf

    RedisConf struct {
        Host string
        Type string
        Pass string
    }
}
