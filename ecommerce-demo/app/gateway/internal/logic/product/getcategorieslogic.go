package product

import (
	"context"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCategoriesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCategoriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCategoriesLogic {
	return &GetCategoriesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCategoriesLogic) GetCategories() (resp *types.GetCategoriesResp, err error) {
	rpcResp, err := l.svcCtx.ProductRpc.GetCategories(l.ctx, &pb.GetCategoriesReq{})
	if err != nil {
		return nil, err
	}

	var categories []types.CategoryItem
	for _, c := range rpcResp.Categories {
		categories = append(categories, types.CategoryItem{
			Id:   c.Id,
			Name: c.Name,
			Icon: c.Icon,
			Sort: c.Sort,
		})
	}

	return &types.GetCategoriesResp{
		Categories: categories,
	}, nil
}
