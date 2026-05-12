// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package order

import (
	"context"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/app/order/pb"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderLogic {
	return &CreateOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateOrderLogic) CreateOrder(req *types.CreateOrderReq) (resp *types.CreateOrderResp, err error) {
	// 使用统一的 ctxutil 获取用户 ID
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	// 调用 Order RPC 下单
	rpcResp, err := l.svcCtx.OrderRpc.CreateOrder(l.ctx, &pb.CreateOrderReq{
		UserId:    userId,
		ProductId: req.ProductId,
		Count:     req.Count,
	})
	if err != nil {
		return nil, err
	}

	return &types.CreateOrderResp{
		OrderNo: rpcResp.OrderNo,
	}, nil
}
