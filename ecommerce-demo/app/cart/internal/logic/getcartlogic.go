package logic

import (
	"context"

	"ecommerce-demo/app/cart/internal/svc"
	"ecommerce-demo/app/cart/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCartLogic {
	return &GetCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCartLogic) GetCart(in *pb.GetCartReq) (*pb.GetCartResp, error) {
	return l.svcCtx.CartService.GetCart(l.ctx, in)
}
