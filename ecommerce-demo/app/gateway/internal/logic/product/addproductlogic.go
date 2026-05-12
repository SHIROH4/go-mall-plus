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

type AddProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddProductLogic {
	return &AddProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddProductLogic) AddProduct(req *types.AddProductReq) (resp *types.AddProductResp, err error) {
	rpcResp, err := l.svcCtx.ProductRpc.AddProduct(l.ctx, &pb.AddProductReq{
		Name:       req.Name,
		Desc:       req.Desc,
		Price:      req.Price,
		Stock:      req.Stock,
		ImageUrl:   req.ImageUrl,
		CategoryId: req.CategoryId,
	})
	if err != nil {
		return nil, err
	}

	return &types.AddProductResp{
		Id:      rpcResp.Id,
		Message: rpcResp.Message,
	}, nil
}
