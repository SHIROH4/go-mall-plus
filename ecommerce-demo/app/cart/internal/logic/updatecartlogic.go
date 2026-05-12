package logic

import (
	"context"

	"ecommerce-demo/app/cart/internal/svc"
	"ecommerce-demo/app/cart/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCartLogic {
	return &UpdateCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateCartLogic) UpdateCart(in *pb.UpdateCartReq) (*pb.UpdateCartResp, error) {
	return l.svcCtx.CartService.UpdateCart(l.ctx, in)
}
