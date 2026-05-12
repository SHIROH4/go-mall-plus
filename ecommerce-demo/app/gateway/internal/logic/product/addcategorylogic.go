package product

import (
	"context"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/app/product/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddCategoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddCategoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddCategoryLogic {
	return &AddCategoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddCategoryLogic) AddCategory(req *types.AddCategoryReq) (resp *types.AddCategoryResp, err error) {
	rpcResp, err := l.svcCtx.ProductRpc.AddCategory(l.ctx, &pb.AddCategoryReq{
		Name: req.Name,
		Icon: req.Icon,
		Sort: req.Sort,
	})
	if err != nil {
		return nil, err
	}

	return &types.AddCategoryResp{
		Id:      rpcResp.Id,
		Message: rpcResp.Message,
	}, nil
}
