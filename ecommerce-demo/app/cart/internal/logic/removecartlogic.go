package logic

import (
	"context"

	"ecommerce-demo/app/cart/internal/svc"
	"ecommerce-demo/app/cart/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveCartLogic {
	return &RemoveCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RemoveCartLogic) RemoveCart(in *pb.RemoveCartReq) (*pb.RemoveCartResp, error) {
	return l.svcCtx.CartService.RemoveCart(l.ctx, in)
}
