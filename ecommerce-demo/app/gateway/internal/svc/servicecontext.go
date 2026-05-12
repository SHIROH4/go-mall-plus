package svc

import (
	"ecommerce-demo/app/address/pb"
	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/config"
	"ecommerce-demo/app/gateway/internal/middleware"
	orderclient "ecommerce-demo/app/order/order"
	paymentclient "ecommerce-demo/app/payment/pb"
	productclient "ecommerce-demo/app/product/product"
	userclient "ecommerce-demo/app/user/user"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config

	UserRpc    userclient.User
	ProductRpc productclient.Product
	OrderRpc   orderclient.Order
	CartRpc    cartpb.CartClient
	PaymentRpc paymentclient.PaymentClient
	AddressRpc pb.AddressClient
	RDB        *redis.Client

	AuthMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.RedisConf.Host,
		Password: c.RedisConf.Pass,
		DB:       c.RedisConf.DB,
	})

	userRpc := zrpc.MustNewClient(c.UserRpcConf)
	productRpc := zrpc.MustNewClient(c.ProductRpcConf)
	orderRpc := zrpc.MustNewClient(c.OrderRpcConf)
	cartRpc := zrpc.MustNewClient(c.CartRpcConf)
	paymentRpc := zrpc.MustNewClient(c.PaymentRpcConf)
	addressRpc := zrpc.MustNewClient(c.AddressRpcConf)

	return &ServiceContext{
		Config: c,

		UserRpc:        userclient.NewUser(userRpc),
		ProductRpc:     productclient.NewProduct(productRpc),
		OrderRpc:       orderclient.NewOrder(orderRpc),
		CartRpc:        cartpb.NewCartClient(cartRpc.Conn()),
		PaymentRpc:     paymentclient.NewPaymentClient(paymentRpc.Conn()),
		AddressRpc:     pb.NewAddressClient(addressRpc.Conn()),
		RDB:            rdb,
		AuthMiddleware: middleware.NewAuthMiddleware(c, rdb).Handle,
	}
}
