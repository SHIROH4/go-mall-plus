package logic

import (
	"context"

	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewClearCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearCartLogic {
	return &ClearCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ClearCartLogic) ClearCart(req *types.ClearCartReq) (*types.ClearCartResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.CartRpc.ClearCart(l.ctx, &cartpb.ClearCartReq{
		UserId: userId,
	})

	if err != nil {
		return nil, err
	}

	return &types.ClearCartResp{
		Message:      resp.Message,
		RemovedCount: resp.RemovedCount,
	}, nil
}
