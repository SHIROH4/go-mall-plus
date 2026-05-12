package main

import (
	"flag"
	"fmt"
	"net/http"

	"ecommerce-demo/app/gateway/internal/config"
	"ecommerce-demo/app/gateway/internal/handler"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/common/response"
	customValidator "ecommerce-demo/common/validator"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf,
		rest.WithCors("*"),
	)
	defer server.Stop()

	// 注册 /metrics 端点
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/metrics",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			promhttp.Handler().ServeHTTP(w, r)
		},
	})

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	httpx.SetValidator(customValidator.NewCustomValidator())

	httpx.SetErrorHandler(func(err error) (int, interface{}) {
		return http.StatusOK, response.Body{
			Code: 400,
			Msg:  err.Error(),
		}
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
