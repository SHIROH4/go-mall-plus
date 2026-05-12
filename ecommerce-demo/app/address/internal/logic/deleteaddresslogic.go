package logic

import (
	"context"

	"ecommerce-demo/app/address/internal/svc"
	"ecommerce-demo/app/address/pb"
)

type DeleteAddressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAddressLogic {
	return &DeleteAddressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteAddressLogic) DeleteAddress(in *pb.DeleteAddressReq) (*pb.DeleteAddressResp, error) {
	err := l.svcCtx.AddressService.DeleteAddress(l.ctx, in.Id, in.UserId)
	if err != nil {
		return &pb.DeleteAddressResp{
			Success: false,
			Message: "删除失败: " + err.Error(),
		}, nil
	}

	return &pb.DeleteAddressResp{
		Success: true,
		Message: "删除成功",
	}, nil
}
