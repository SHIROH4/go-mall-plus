// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"ecommerce-demo/app/order/internal/svc"
	"ecommerce-demo/app/order/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrderLogic {
	return &ListOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOrderLogic) ListOrder(req *pb.ListOrderReq) (*pb.ListOrderResp, error) {
	return l.svcCtx.OrderService.ListOrder(l.ctx, req)
}
