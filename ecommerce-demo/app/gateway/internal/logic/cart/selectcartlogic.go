package logic

import (
	"context"

	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type SelectCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSelectCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SelectCartLogic {
	return &SelectCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SelectCartLogic) SelectCart(req *types.SelectCartReq) (*types.SelectCartResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.CartRpc.SelectCart(l.ctx, &cartpb.SelectCartReq{
		UserId:    userId,
		ProductId: req.ProductId,
		Selected:  req.Selected,
	})

	if err != nil {
		return nil, err
	}

	return &types.SelectCartResp{
		Message: resp.Message,
	}, nil
}
