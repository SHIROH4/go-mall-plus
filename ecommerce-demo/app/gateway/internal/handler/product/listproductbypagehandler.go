package product

import (
	"net/http"

	"ecommerce-demo/app/gateway/internal/logic/product"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/response"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListProductByPageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListProductByPageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := product.NewListProductByPageLogic(r.Context(), svcCtx)
		resp, err := l.ListProductByPage(&req)
		response.Response(w, resp, err)
	}
}
