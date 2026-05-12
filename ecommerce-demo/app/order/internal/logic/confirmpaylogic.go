package logic

import (
	"context"

	"ecommerce-demo/app/order/internal/svc"
	"ecommerce-demo/app/order/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmPayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmPayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmPayLogic {
	return &ConfirmPayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ConfirmPayLogic) ConfirmPay(in *pb.ConfirmPayReq) (*pb.ConfirmPayResp, error) {
	return l.svcCtx.OrderService.ConfirmPay(l.ctx, in)
}
