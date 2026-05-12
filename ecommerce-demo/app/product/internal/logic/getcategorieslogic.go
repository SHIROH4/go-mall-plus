package logic

import (
	"context"

	"ecommerce-demo/app/product/internal/svc"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCategoriesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCategoriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCategoriesLogic {
	return &GetCategoriesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetCategories 获取商品分类
func (l *GetCategoriesLogic) GetCategories(in *pb.GetCategoriesReq) (*pb.GetCategoriesResp, error) {
	return l.svcCtx.ProductService.GetCategories(l.ctx, in)
}
