package logic

import (
	"context"

	"ecommerce-demo/app/product/internal/svc"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProductLogic {
	return &ListProductLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取商品列表
func (l *ListProductLogic) ListProduct(in *pb.ListProductReq) (*pb.ListProductResp, error) {
	return l.svcCtx.ProductService.ListProduct(l.ctx, in)
}
