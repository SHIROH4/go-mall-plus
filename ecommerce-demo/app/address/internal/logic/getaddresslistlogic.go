package logic

import (
	"context"

	"ecommerce-demo/app/address/internal/svc"
	"ecommerce-demo/app/address/pb"
)

type GetAddressListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAddressListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAddressListLogic {
	return &GetAddressListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAddressListLogic) GetAddressList(in *pb.GetAddressListReq) (*pb.GetAddressListResp, error) {
	list, err := l.svcCtx.AddressService.GetAddressList(l.ctx, in.UserId)
	if err != nil {
		return nil, err
	}

	var items []*pb.AddressItem
	for _, addr := range list {
		items = append(items, &pb.AddressItem{
			Id:            addr.ID,
			UserId:        addr.UserID,
			ReceiverName:  addr.ReceiverName,
			Phone:         addr.Phone,
			Province:      addr.Province,
			City:          addr.City,
			District:      addr.District,
			DetailAddress: addr.DetailAddress,
			PostalCode:    addr.PostalCode,
			IsDefault:     addr.IsDefault,
		})
	}

	return &pb.GetAddressListResp{Addresses: items}, nil
}
