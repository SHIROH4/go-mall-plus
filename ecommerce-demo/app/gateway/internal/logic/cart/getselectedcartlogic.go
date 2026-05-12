package logic

import (
	"context"

	cartpb "ecommerce-demo/app/cart/pb"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSelectedCartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSelectedCartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSelectedCartLogic {
	return &GetSelectedCartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetSelectedCartLogic) GetSelectedCart(req *types.GetSelectedCartReq) (*types.GetSelectedCartResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.CartRpc.GetSelectedCart(l.ctx, &cartpb.GetSelectedCartReq{
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

	return &types.GetSelectedCartResp{
		Items:         items,
		SelectedCount: resp.SelectedCount,
		TotalAmount:   resp.TotalAmount,
	}, nil
}
