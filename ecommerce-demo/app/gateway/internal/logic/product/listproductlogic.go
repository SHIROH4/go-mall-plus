// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package product

import (
	"context"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProductLogic {
	return &ListProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListProductLogic) ListProduct() (resp *types.ListProductResp, err error) {
	// 1. 请求 Product RPC
	rpcResp, err := l.svcCtx.ProductRpc.ListProduct(l.ctx, &pb.ListProductReq{})
	if err != nil {
		return nil, err
	}

	// 2. 转换结构
	var list []types.ProductItem
	for _, p := range rpcResp.Products {
		list = append(list, types.ProductItem{
			Id:    p.Id,
			Name:  p.Name,
			Desc:  p.Desc,
			Price: p.Price,
		})
	}

	return &types.ListProductResp{
		Products: list,
	}, nil
}
