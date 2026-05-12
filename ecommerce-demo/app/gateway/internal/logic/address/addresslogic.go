package address

import (
	"context"

	"ecommerce-demo/app/address/pb"
	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddressLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddressLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddressLogic {
	return &AddressLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddressLogic) GetAddressList() (*types.GetAddressListResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.AddressRpc.GetAddressList(l.ctx, &pb.GetAddressListReq{
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	var items []types.AddressItem
	for _, addr := range rpcResp.Addresses {
		items = append(items, types.AddressItem{
			Id:            addr.Id,
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

	return &types.GetAddressListResp{Addresses: items}, nil
}

func (l *AddressLogic) GetAddress(req *types.GetAddressReq) (*types.GetAddressResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.AddressRpc.GetAddressList(l.ctx, &pb.GetAddressListReq{
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	for _, addr := range rpcResp.Addresses {
		if addr.Id == req.Id {
			return &types.GetAddressResp{
				Address: types.AddressItem{
					Id:            addr.Id,
					ReceiverName:  addr.ReceiverName,
					Phone:         addr.Phone,
					Province:      addr.Province,
					City:          addr.City,
					District:      addr.District,
					DetailAddress: addr.DetailAddress,
					PostalCode:    addr.PostalCode,
					IsDefault:     addr.IsDefault,
				},
			}, nil
		}
	}

	return nil, ErrAddressNotFound
}

func (l *AddressLogic) AddAddress(req *types.AddAddressReq) (*types.AddAddressResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.AddressRpc.AddAddress(l.ctx, &pb.AddAddressReq{
		UserId:        userId,
		ReceiverName:  req.ReceiverName,
		Phone:         req.Phone,
		Province:      req.Province,
		City:          req.City,
		District:      req.District,
		DetailAddress: req.DetailAddress,
		PostalCode:    req.PostalCode,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		return nil, err
	}

	return &types.AddAddressResp{
		Id:      rpcResp.Id,
		Success: rpcResp.Success,
		Message: rpcResp.Message,
	}, nil
}

func (l *AddressLogic) UpdateAddress(req *types.UpdateAddressReq) (*types.UpdateAddressResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.AddressRpc.UpdateAddress(l.ctx, &pb.UpdateAddressReq{
		Id:            req.Id,
		UserId:        userId,
		ReceiverName:  req.ReceiverName,
		Phone:         req.Phone,
		Province:      req.Province,
		City:          req.City,
		District:      req.District,
		DetailAddress: req.DetailAddress,
		PostalCode:    req.PostalCode,
		IsDefault:     req.IsDefault,
	})
	if err != nil {
		return nil, err
	}

	return &types.UpdateAddressResp{
		Success: rpcResp.Success,
		Message: rpcResp.Message,
	}, nil
}

func (l *AddressLogic) DeleteAddress(req *types.DeleteAddressReq) (*types.DeleteAddressResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.AddressRpc.DeleteAddress(l.ctx, &pb.DeleteAddressReq{
		Id:     req.Id,
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	return &types.DeleteAddressResp{
		Success: rpcResp.Success,
		Message: rpcResp.Message,
	}, nil
}

func (l *AddressLogic) SetDefaultAddress(req *types.SetDefaultAddressReq) (*types.SetDefaultAddressResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	rpcResp, err := l.svcCtx.AddressRpc.SetDefaultAddress(l.ctx, &pb.SetDefaultAddressReq{
		Id:     req.Id,
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}

	return &types.SetDefaultAddressResp{
		Success: rpcResp.Success,
		Message: rpcResp.Message,
	}, nil
}

var ErrAddressNotFound = &addressError{msg: "地址不存在"}

type addressError struct {
	msg string
}

func (e *addressError) Error() string {
	return e.msg
}
