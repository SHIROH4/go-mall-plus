// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package order

import (
	"net/http"

	"ecommerce-demo/app/gateway/internal/logic/order"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func CancelOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CancelOrderReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := order.NewCancelOrderLogic(r.Context(), svcCtx)
		resp, err := l.CancelOrder(&req)
		response.Response(w, resp, err)
	}
}
