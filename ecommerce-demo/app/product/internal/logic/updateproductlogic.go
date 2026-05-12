package logic

import (
	"context"

	"ecommerce-demo/app/product/internal/svc"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProductLogic {
	return &UpdateProductLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdateProduct 更新商品
func (l *UpdateProductLogic) UpdateProduct(in *pb.UpdateProductReq) (*pb.UpdateProductResp, error) {
	return l.svcCtx.ProductService.UpdateProduct(l.ctx, in)
}
