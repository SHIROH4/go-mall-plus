// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package product

import (
	"net/http"

	"ecommerce-demo/app/gateway/internal/logic/product"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/common/response"
)

func ListProductHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := product.NewListProductLogic(r.Context(), svcCtx)
		resp, err := l.ListProduct()
		response.Response(w, resp, err)
	}
}
