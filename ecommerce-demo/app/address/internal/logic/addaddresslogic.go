package logic

import (
	"context"

	"ecommerce-demo/app/address/internal/repo"
	"ecommerce-demo/app/address/internal/svc"
	"ecommerce-demo/app/address/pb"
)

type AddAddressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddAddressLogic {
	return &AddAddressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddAddressLogic) AddAddress(in *pb.AddAddressReq) (*pb.AddAddressResp, error) {
	addr := &repo.Address{
		UserID:        in.UserId,
		ReceiverName:  in.ReceiverName,
		Phone:         in.Phone,
		Province:      in.Province,
		City:          in.City,
		District:      in.District,
		DetailAddress: in.DetailAddress,
		PostalCode:    in.PostalCode,
		IsDefault:     in.IsDefault,
	}

	id, err := l.svcCtx.AddressService.AddAddress(l.ctx, addr)
	if err != nil {
		return &pb.AddAddressResp{
			Success: false,
			Message: "添加失败: " + err.Error(),
		}, nil
	}

	return &pb.AddAddressResp{
		Id:      id,
		Success: true,
		Message: "添加成功",
	}, nil
}
