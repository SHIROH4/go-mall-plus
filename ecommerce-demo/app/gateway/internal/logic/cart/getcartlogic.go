package logic

import (
	"context"

	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCartLogic {
	return &GetCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCartLogic) GetCart(req *types.GetCartReq) (*types.GetCartResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.CartRpc.GetCart(l.ctx, &cartpb.GetCartReq{
		UserId: userId,
	})

	if err != nil {
		return nil, err
	}

	items := make([]types.CartItem, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = types.CartItem{
			ProductId:   item.ProductId,
			ProductName: item.ProductName,
			Price:       item.Price,
			ImageUrl:    item.ImageUrl,
			Count:       item.Count,
			Selected:    item.Selected,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
	}

	return &types.GetCartResp{
		Items:       items,
		TotalCount:  resp.TotalCount,
		TotalAmount: resp.TotalAmount,
	}, nil
}
