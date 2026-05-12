package logic

import (
	"context"

	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCartLogic {
	return &UpdateCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateCartLogic) UpdateCart(req *types.UpdateCartReq) (*types.UpdateCartResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.CartRpc.UpdateCart(l.ctx, &cartpb.UpdateCartReq{
		UserId:    userId,
		ProductId: req.ProductId,
		Count:     req.Count,
	})

	if err != nil {
		return nil, err
	}

	return &types.UpdateCartResp{
		Message: resp.Message,
	}, nil
}
