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

type DeleteProductLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteProductLogic {
	return &DeleteProductLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteProductLogic) DeleteProduct(req *types.DeleteProductReq) (resp *types.DeleteProductResp, err error) {
	rpcResp, err := l.svcCtx.ProductRpc.DeleteProduct(l.ctx, &pb.DeleteProductReq{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}

	return &types.DeleteProductResp{
		Success: rpcResp.Success,
		Message: rpcResp.Message,
	}, nil
}
