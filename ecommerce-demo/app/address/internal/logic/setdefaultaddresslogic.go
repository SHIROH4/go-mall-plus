package logic

import (
	"context"

	"ecommerce-demo/app/address/internal/svc"
	"ecommerce-demo/app/address/pb"
)

type SetDefaultAddressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetDefaultAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetDefaultAddressLogic {
	return &SetDefaultAddressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetDefaultAddressLogic) SetDefaultAddress(in *pb.SetDefaultAddressReq) (*pb.SetDefaultAddressResp, error) {
	err := l.svcCtx.AddressService.SetDefaultAddress(l.ctx, in.Id, in.UserId)
	if err != nil {
		return &pb.SetDefaultAddressResp{
			Success: false,
			Message: "设置默认地址失败: " + err.Error(),
		}, nil
	}

	return &pb.SetDefaultAddressResp{
		Success: true,
		Message: "设置默认地址成功",
	}, nil
}
