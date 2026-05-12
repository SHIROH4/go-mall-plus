package logic

import (
	"context"

	"ecommerce-demo/app/product/internal/svc"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProductLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetProductLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProductLogic {
	return &GetProductLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取商品详情 (这个接口后面我们会做 Redis 高并发缓存)
func (l *GetProductLogic) GetProduct(in *pb.GetProductReq) (*pb.GetProductResp, error) {
	return l.svcCtx.ProductService.GetProduct(l.ctx, in)
}
