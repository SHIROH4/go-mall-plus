package logic

import (
	"context"

	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddCartLogic {
	return &AddCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AddCartLogic) AddCart(req *types.AddCartReq) (*types.AddCartResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.CartRpc.AddCart(l.ctx, &cartpb.AddCartReq{
		UserId:      userId,
		ProductId:   req.ProductId,
		ProductName: req.ProductName,
		Price:       req.Price,
		ImageUrl:    req.ImageUrl,
		Count:       req.Count,
	})

	if err != nil {
		return nil, err
	}

	return &types.AddCartResp{
		Message:    resp.Message,
		TotalCount: resp.TotalCount,
	}, nil
}
