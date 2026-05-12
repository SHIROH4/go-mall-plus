package logic

import (
	"context"

	"ecommerce-demo/app/product/internal/svc"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListProductByPageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListProductByPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListProductByPageLogic {
	return &ListProductByPageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListProductByPage 分页获取商品列表
func (l *ListProductByPageLogic) ListProductByPage(in *pb.ListProductByPageReq) (*pb.ListProductByPageResp, error) {
	return l.svcCtx.ProductService.ListProductByPage(l.ctx, in)
}
