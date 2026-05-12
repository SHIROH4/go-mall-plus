package logic

import (
	"context"

	"ecommerce-demo/app/cart/internal/svc"
	"ecommerce-demo/app/cart/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSelectedCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSelectedCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSelectedCartLogic {
	return &GetSelectedCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetSelectedCartLogic) GetSelectedCart(in *pb.GetSelectedCartReq) (*pb.GetSelectedCartResp, error) {
	return l.svcCtx.CartService.GetSelectedCart(l.ctx, in)
}
