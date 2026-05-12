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

func (l *ListOrderLogic) ListOrder(req *types.ListOrderReq) (resp *types.ListOrderResp, err error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.OrderRpc.ListOrder(l.ctx, &pb.ListOrderReq{
		UserId:   userId,
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
	})
	if err != nil {
		return nil, err
	}

	var orders []types.OrderItem
	for _, item := range rpcResp.Orders {
		orders = append(orders, types.OrderItem{
			Id:          item.Id,
			OrderNo:     item.OrderNo,
			ProductId:   item.ProductId,
			ProductName: item.ProductName,
			Count:       item.Count,
			TotalAmount: item.TotalAmount,
			Status:      item.Status,
			StatusText:  item.StatusText,
			CreateTime:  item.CreateTime,
			PayTime:     item.PayTime,
		})
	}

	return &types.ListOrderResp{
		Orders:   orders,
		Total:    rpcResp.Total,
		Page:     rpcResp.Page,
		PageSize: rpcResp.PageSize,
	}, nil
}
