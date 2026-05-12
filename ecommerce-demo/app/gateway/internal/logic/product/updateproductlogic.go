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

type UpdateProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProductLogic {
	return &UpdateProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateProductLogic) UpdateProduct(req *types.UpdateProductReq) (resp *types.UpdateProductResp, err error) {
	rpcResp, err := l.svcCtx.ProductRpc.UpdateProduct(l.ctx, &pb.UpdateProductReq{
		Id:         req.Id,
		Name:       req.Name,
		Desc:       req.Desc,
		Price:      req.Price,
		ImageUrl:   req.ImageUrl,
		CategoryId: req.CategoryId,
	})
	if err != nil {
		return nil, err
	}

	return &types.UpdateProductResp{
		Success: rpcResp.Success,
		Message: rpcResp.Message,
	}, nil
}
