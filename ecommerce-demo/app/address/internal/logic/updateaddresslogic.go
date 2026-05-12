package logic

import (
	"context"

	"ecommerce-demo/app/address/internal/repo"
	"ecommerce-demo/app/address/internal/svc"
	"ecommerce-demo/app/address/pb"
)

type UpdateAddressLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAddressLogic {
	return &UpdateAddressLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAddressLogic) UpdateAddress(in *pb.UpdateAddressReq) (*pb.UpdateAddressResp, error) {
	addr := &repo.Address{
		ID:            in.Id,
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

	err := l.svcCtx.AddressService.UpdateAddress(l.ctx, addr)
	if err != nil {
		return &pb.UpdateAddressResp{
			Success: false,
			Message: "更新失败: " + err.Error(),
		}, nil
	}

	return &pb.UpdateAddressResp{
		Success: true,
		Message: "更新成功",
	}, nil
}
