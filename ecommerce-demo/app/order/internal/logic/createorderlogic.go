package logic

import (
	"context"

	"ecommerce-demo/app/order/internal/svc"
	"ecommerce-demo/app/order/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderLogic {
	return &CreateOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 核心下单接口 (内部将实现高并发扣库存)
func (l *CreateOrderLogic) CreateOrder(in *pb.CreateOrderReq) (*pb.CreateOrderResp, error) {
	return l.svcCtx.OrderService.CreateOrder(l.ctx, in)
}
