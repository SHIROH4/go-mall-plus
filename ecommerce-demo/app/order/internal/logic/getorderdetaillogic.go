// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"ecommerce-demo/app/order/internal/svc"
	"ecommerce-demo/app/order/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrderDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderDetailLogic {
	return &GetOrderDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrderDetailLogic) GetOrderDetail(req *pb.GetOrderDetailReq) (*pb.GetOrderDetailResp, error) {
	return l.svcCtx.OrderService.GetOrderDetail(l.ctx, req)
}
