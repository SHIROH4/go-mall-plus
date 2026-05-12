// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"net/http"

	"ecommerce-demo/app/gateway/internal/logic/user"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/common/response"
)

func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := user.NewLogoutLogic(r.Context(), svcCtx)
		resp, err := l.Logout()
		response.Response(w, resp, err)
	}
}
