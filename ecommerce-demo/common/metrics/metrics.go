package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ========== 订单指标 ==========
	OrderCreateTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "order_create_total",
			Help: "下单请求总数",
		},
		[]string{"status"}, // success, fail
	)

	// 微服务延迟分桶：1ms ~ 10s，覆盖正常到超时范围
	microBuckets = []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

	OrderCreateDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_create_duration_seconds",
			Help:    "下单请求耗时分布",
			Buckets: microBuckets,
		},
		[]string{},
	)

	OrderStatusDistribution = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "order_status_distribution",
			Help: "各状态订单数量快照",
		},
		[]string{"status"}, // pending, paid, cancelled, timeout
	)

	OrderTimeoutTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "order_timeout_total",
			Help: "超时自动取消的订单总数",
		},
	)

	// ========== 支付指标 ==========
	PaymentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_total",
			Help: "支付请求总数",
		},
		[]string{"status", "channel"}, // success/fail, alipay/wechat
	)

	PaymentAmountTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_amount_total",
			Help: "支付总金额（单位：分）",
		},
		[]string{"channel"},
	)

	// ========== 库存指标 ==========
	StockRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "stock_remaining",
			Help: "当前剩余库存（Redis实时值）",
		},
		[]string{"product_id"},
	)

	StockDeductTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stock_deduct_total",
			Help: "库存扣减总次数",
		},
		[]string{"status"}, // success, fail
	)

	// ========== MQ 指标 ==========
	MQPublishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mq_publish_total",
			Help: "MQ消息投递总数",
		},
		[]string{"type", "status"}, // order.created/order.delay.check, success/fail
	)

	MQConsumeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mq_consume_total",
			Help: "MQ消息消费总数",
		},
		[]string{"type", "status"}, // order.created/order.delay.check, success/fail/dql
	)

	OutboxPendingGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "outbox_pending_count",
			Help: "Outbox表中待投递消息数量",
		},
	)

	// ========== HTTP 指标 ==========
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_http_requests_total",
			Help: "网关HTTP请求总数",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_http_request_duration_seconds",
			Help:    "网关HTTP请求耗时分布",
			Buckets: microBuckets,
		},
		[]string{"method", "path"},
	)

	// ========== RPC 指标 ==========
	RPCCallTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_call_total",
			Help: "RPC调用总数",
		},
		[]string{"service", "method", "status"},
	)

	RPCCallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rpc_call_duration_seconds",
			Help:    "RPC调用耗时分布",
			Buckets: microBuckets,
		},
		[]string{"service", "method"},
	)
)
