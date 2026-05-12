// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"ecommerce-demo/app/order/internal/svc"
	"ecommerce-demo/app/order/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelOrderLogic {
	return &CancelOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelOrderLogic) CancelOrder(req *pb.CancelOrderReq) (*pb.CancelOrderResp, error) {
	return l.svcCtx.OrderService.CancelOrder(l.ctx, req)
}
