package svc

import (
	"crypto/rsa"

	"ecommerce-demo/app/address/pb"
	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/config"
	"ecommerce-demo/app/gateway/internal/middleware"
	orderclient "ecommerce-demo/app/order/order"
	paymentclient "ecommerce-demo/app/payment/pb"
	productclient "ecommerce-demo/app/product/product"
	stockclient "ecommerce-demo/app/stock/stock"
	userclient "ecommerce-demo/app/user/user"
	"ecommerce-demo/common/metrics/rpcmetrics"
	"ecommerce-demo/common/utils"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	UserRpc    userclient.User
	ProductRpc productclient.Product
	OrderRpc   orderclient.Order
	StockRpc   stockclient.Stock
	CartRpc    cartpb.CartClient
	PaymentRpc paymentclient.PaymentClient
	AddressRpc pb.AddressClient
	RDB        *redis.ClusterClient

	// RSA 密钥对（启动时加载到内存，供鉴权和Token签发使用）
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey

	// 中间件
	AuthMiddleware      rest.Middleware
	RateLimitMiddleware rest.Middleware
	CORSMiddleware      rest.Middleware
	MetricsMiddleware   rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    []string{c.RedisConf.Host},
		Password: c.RedisConf.Pass,
	})

	// 启动时加载 RSA 密钥对（一次加载，全局复用）
	pubKey, err := utils.LoadRSAPublicKey(c.Auth.PublicKeyPath)
	if err != nil {
		panic("加载RSA公钥失败: " + err.Error())
	}
	priKey, err := utils.LoadRSAPrivateKey(c.Auth.PrivateKeyPath)
	if err != nil {
		panic("加载RSA私钥失败: " + err.Error())
	}

	userRpc := zrpc.MustNewClient(c.UserRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor()))
	productRpc := zrpc.MustNewClient(c.ProductRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor()))
	orderRpc := zrpc.MustNewClient(c.OrderRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor()))
	stockRpc := zrpc.MustNewClient(c.StockRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor()))
	cartRpc := zrpc.MustNewClient(c.CartRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor()))
	paymentRpc := zrpc.MustNewClient(c.PaymentRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor()))
	addressRpc := zrpc.MustNewClient(c.AddressRpcConf, zrpc.WithUnaryClientInterceptor(rpcmetrics.UnaryClientInterceptor()))

	// 构建限流配置
	rateLimitCfg := middleware.RateLimitConfig{
		Enabled: c.RateLimit.Enabled,
	}
	rateLimitCfg.Global.Rate = c.RateLimit.Global.Rate
	rateLimitCfg.Global.Burst = c.RateLimit.Global.Burst
	rateLimitCfg.PerIP.Rate = c.RateLimit.PerIP.Rate
	rateLimitCfg.PerIP.Burst = c.RateLimit.PerIP.Burst
	rateLimitCfg.Routes = c.RateLimit.Routes

	// 构建 CORS 配置
	corsCfg := middleware.CORSConfig{
		Enabled:        c.CORS.Enabled,
		AllowedOrigins: c.CORS.AllowedOrigins,
		AllowedMethods: c.CORS.AllowedMethods,
		AllowedHeaders: c.CORS.AllowedHeaders,
		MaxAge:         c.CORS.MaxAge,
	}
	if !corsCfg.Enabled && len(c.CORS.AllowedOrigins) > 0 {
		corsCfg.Enabled = true
	}
	if corsCfg.MaxAge <= 0 {
		corsCfg.MaxAge = 3600
	}

	authMw := middleware.NewAuthMiddleware(c, rdb)

	return &ServiceContext{
		Config:              c,
		UserRpc:             userclient.NewUser(userRpc),
		ProductRpc:          productclient.NewProduct(productRpc),
		OrderRpc:            orderclient.NewOrder(orderRpc),
		StockRpc:            stockclient.NewStock(stockRpc),
		CartRpc:             cartpb.NewCartClient(cartRpc.Conn()),
		PaymentRpc:          paymentclient.NewPaymentClient(paymentRpc.Conn()),
		AddressRpc:          pb.NewAddressClient(addressRpc.Conn()),
		RDB:                 rdb,
		PublicKey:           pubKey,
		PrivateKey:          priKey,
		AuthMiddleware:      authMw.Handle,
		RateLimitMiddleware: middleware.NewRateLimitMiddleware(rateLimitCfg).Handle,
		CORSMiddleware:      middleware.NewCORSMiddleware(corsCfg).Handle,
		MetricsMiddleware:   middleware.MetricsMiddleware,
	}
}
