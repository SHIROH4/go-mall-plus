package cart

import (
	"net/http"

	cartlogic "ecommerce-demo/app/gateway/internal/logic/cart"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func AddCartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AddCartReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cartlogic.NewAddCartLogic(r.Context(), svcCtx)
		resp, err := l.AddCart(&req)
		response.Response(w, resp, err)
	}
}

func GetCartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetCartReq
		l := cartlogic.NewGetCartLogic(r.Context(), svcCtx)
		resp, err := l.GetCart(&req)
		response.Response(w, resp, err)
	}
}

func UpdateCartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateCartReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cartlogic.NewUpdateCartLogic(r.Context(), svcCtx)
		resp, err := l.UpdateCart(&req)
		response.Response(w, resp, err)
	}
}

func RemoveCartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RemoveCartReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cartlogic.NewRemoveCartLogic(r.Context(), svcCtx)
		resp, err := l.RemoveCart(req.ProductId)
		response.Response(w, resp, err)
	}
}

func ClearCartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ClearCartReq
		l := cartlogic.NewClearCartLogic(r.Context(), svcCtx)
		resp, err := l.ClearCart(&req)
		response.Response(w, resp, err)
	}
}

func SelectCartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SelectCartReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := cartlogic.NewSelectCartLogic(r.Context(), svcCtx)
		resp, err := l.SelectCart(&req)
		response.Response(w, resp, err)
	}
}

func GetSelectedCartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetSelectedCartReq
		l := cartlogic.NewGetSelectedCartLogic(r.Context(), svcCtx)
		resp, err := l.GetSelectedCart(&req)
		response.Response(w, resp, err)
	}
}
