package product

import (
	"context"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListProductByPageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListProductByPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProductByPageLogic {
	return &ListProductByPageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListProductByPageLogic) ListProductByPage(req *types.ListProductByPageReq) (resp *types.ListProductByPageResp, err error) {
	rpcResp, err := l.svcCtx.ProductRpc.ListProductByPage(l.ctx, &pb.ListProductByPageReq{
		CategoryId: req.CategoryId,
		Keyword:    req.Keyword,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	var products []types.ProductItem
	for _, p := range rpcResp.Products {
		products = append(products, types.ProductItem{
			Id:           p.Id,
			Name:         p.Name,
			Desc:         p.Desc,
			Price:        p.Price,
			ImageUrl:     p.ImageUrl,
			CategoryId:   p.CategoryId,
			CategoryName: p.CategoryName,
			Stock:        p.Stock,
		})
	}

	return &types.ListProductByPageResp{
		Products: products,
		Total:    rpcResp.Total,
		Page:     rpcResp.Page,
		PageSize: rpcResp.PageSize,
	}, nil
}
