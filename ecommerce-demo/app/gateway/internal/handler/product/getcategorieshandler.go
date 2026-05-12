package product

import (
	"net/http"

	"ecommerce-demo/app/gateway/internal/logic/product"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/common/response"
)

func GetCategoriesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := product.NewGetCategoriesLogic(r.Context(), svcCtx)
		resp, err := l.GetCategories()
		response.Response(w, resp, err)
	}
}
