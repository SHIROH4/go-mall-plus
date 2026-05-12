package logic

import (
	"context"

	"ecommerce-demo/app/cart/internal/svc"
	"ecommerce-demo/app/cart/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewClearCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearCartLogic {
	return &ClearCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ClearCartLogic) ClearCart(in *pb.ClearCartReq) (*pb.ClearCartResp, error) {
	return l.svcCtx.CartService.ClearCart(l.ctx, in)
}
