package logic

import (
	"context"

	"ecommerce-demo/app/product/internal/svc"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddCategoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddCategoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddCategoryLogic {
	return &AddCategoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// AddCategory 添加商品分类
func (l *AddCategoryLogic) AddCategory(in *pb.AddCategoryReq) (*pb.AddCategoryResp, error) {
	return l.svcCtx.ProductService.AddCategory(l.ctx, in)
}
