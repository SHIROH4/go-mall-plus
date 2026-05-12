package payment

import (
	"net/http"
	"strconv"

	paymentlogic "ecommerce-demo/app/gateway/internal/logic/payment"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func CreatePayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreatePayReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := paymentlogic.NewCreatePayLogic(r.Context(), svcCtx)
		resp, err := l.CreatePay(&req)
		response.Response(w, resp, err)
	}
}

func GetPayStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetPayStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := paymentlogic.NewGetPayStatusLogic(r.Context(), svcCtx)
		resp, err := l.GetPayStatus(req.PaymentNo)
		response.Response(w, resp, err)
	}
}

func CancelPayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CancelPayReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := paymentlogic.NewCancelPayLogic(r.Context(), svcCtx)
		resp, err := l.CancelPay(req.PaymentNo)
		response.Response(w, resp, err)
	}
}

func ListPayHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
		pageSize, _ := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 32)

		l := paymentlogic.NewListPayLogic(r.Context(), svcCtx)
		resp, err := l.ListPay(int32(page), int32(pageSize))
		response.Response(w, resp, err)
	}
}
