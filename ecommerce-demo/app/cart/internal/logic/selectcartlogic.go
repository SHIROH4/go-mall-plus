package logic

import (
	"context"

	"ecommerce-demo/app/cart/internal/svc"
	"ecommerce-demo/app/cart/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type SelectCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSelectCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SelectCartLogic {
	return &SelectCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SelectCartLogic) SelectCart(in *pb.SelectCartReq) (*pb.SelectCartResp, error) {
	return l.svcCtx.CartService.SelectCart(l.ctx, in)
}
