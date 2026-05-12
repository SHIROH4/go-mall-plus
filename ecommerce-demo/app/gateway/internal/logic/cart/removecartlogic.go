package logic

import (
	"context"

	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveCartLogic {
	return &RemoveCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RemoveCartLogic) RemoveCart(productId int64) (*types.RemoveCartResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.CartRpc.RemoveCart(l.ctx, &cartpb.RemoveCartReq{
		UserId:    userId,
		ProductId: productId,
	})

	if err != nil {
		return nil, err
	}

	return &types.RemoveCartResp{
		Message: resp.Message,
	}, nil
}
