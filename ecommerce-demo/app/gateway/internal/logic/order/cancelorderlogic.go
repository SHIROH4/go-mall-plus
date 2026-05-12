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

func (l *CancelOrderLogic) CancelOrder(req *types.CancelOrderReq) (resp *types.CancelOrderResp, err error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.OrderRpc.CancelOrder(l.ctx, &pb.CancelOrderReq{
		UserId:  userId,
		OrderNo: req.OrderNo,
	})
	if err != nil {
		return nil, err
	}

	return &types.CancelOrderResp{
		Success: rpcResp.Success,
		Message: rpcResp.Message,
	}, nil
}
