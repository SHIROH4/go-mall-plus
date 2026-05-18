package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	DataSource string
	RedisConf  struct {
		Host string
		Type string
		Pass string
	}
	// 库存对账配置
	Reconciliation struct {
		Enabled            bool
		IntervalSeconds    int // 对账间隔，默认 300 秒
		AlertThresholdPct  int // 差异百分比告警阈值，默认 10%
	}
}
