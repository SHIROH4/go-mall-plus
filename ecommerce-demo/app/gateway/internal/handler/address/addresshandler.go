package address

import (
	"net/http"

	"ecommerce-demo/app/gateway/internal/logic/address"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/response"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetAddressListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := address.NewAddressLogic(r.Context(), svcCtx)
		resp, err := l.GetAddressList()
		response.Response(w, resp, err)
	}
}

func GetAddressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetAddressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := address.NewAddressLogic(r.Context(), svcCtx)
		resp, err := l.GetAddress(&req)
		response.Response(w, resp, err)
	}
}

func AddAddressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AddAddressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := address.NewAddressLogic(r.Context(), svcCtx)
		resp, err := l.AddAddress(&req)
		response.Response(w, resp, err)
	}
}

func UpdateAddressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateAddressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := address.NewAddressLogic(r.Context(), svcCtx)
		resp, err := l.UpdateAddress(&req)
		response.Response(w, resp, err)
	}
}

func DeleteAddressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeleteAddressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := address.NewAddressLogic(r.Context(), svcCtx)
		resp, err := l.DeleteAddress(&req)
		response.Response(w, resp, err)
	}
}

func SetDefaultAddressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SetDefaultAddressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := address.NewAddressLogic(r.Context(), svcCtx)
		resp, err := l.SetDefaultAddress(&req)
		response.Response(w, resp, err)
	}
}
