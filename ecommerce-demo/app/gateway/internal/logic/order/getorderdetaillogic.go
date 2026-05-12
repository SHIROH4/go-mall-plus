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

func (l *GetOrderDetailLogic) GetOrderDetail(req *types.GetOrderDetailReq) (resp *types.GetOrderDetailResp, err error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.OrderRpc.GetOrderDetail(l.ctx, &pb.GetOrderDetailReq{
		UserId:  userId,
		OrderNo: req.OrderNo,
	})
	if err != nil {
		return nil, err
	}

	return &types.GetOrderDetailResp{
		Id:           rpcResp.Id,
		OrderNo:      rpcResp.OrderNo,
		ProductId:    rpcResp.ProductId,
		ProductName:  rpcResp.ProductName,
		ProductDesc:  rpcResp.ProductDesc,
		ProductImage: rpcResp.ProductImage,
		Count:        rpcResp.Count,
		TotalAmount:  rpcResp.TotalAmount,
		Status:       rpcResp.Status,
		StatusText:   rpcResp.StatusText,
		CreateTime:   rpcResp.CreateTime,
		PayTime:      rpcResp.PayTime,
		ExpireTime:   rpcResp.ExpireTime,
	}, nil
}
