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

type GetProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProductLogic {
	return &GetProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetProductLogic) GetProduct(req *types.GetProductReq) (resp *types.GetProductResp, err error) {
	rpcResp, err := l.svcCtx.ProductRpc.GetProduct(l.ctx, &pb.GetProductReq{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}

	return &types.GetProductResp{
		Product: types.ProductItem{
			Id:    rpcResp.Product.Id,
			Name:  rpcResp.Product.Name,
			Desc:  rpcResp.Product.Desc,
			Price: rpcResp.Product.Price,
		},
	}, nil
}
