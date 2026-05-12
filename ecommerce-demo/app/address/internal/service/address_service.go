package service

import (
	"context"
	"errors"

	"ecommerce-demo/app/address/internal/repo"
)

var (
	ErrAddressNotFound = errors.New("地址不存在")
	ErrNoPermission    = errors.New("无权限操作该地址")
)

// AddressService 地址服务接口
type AddressService interface {
	GetAddressList(ctx context.Context, userID int64) ([]*repo.Address, error)
	GetAddress(ctx context.Context, id, userID int64) (*repo.Address, error)
	AddAddress(ctx context.Context, addr *repo.Address) (int64, error)
	UpdateAddress(ctx context.Context, addr *repo.Address) error
	DeleteAddress(ctx context.Context, id, userID int64) error
	SetDefaultAddress(ctx context.Context, id, userID int64) error
}

type addressServiceImpl struct {
	addressRepo repo.AddressRepo
}

func NewAddressService(addressRepo repo.AddressRepo) AddressService {
	return &addressServiceImpl{
		addressRepo: addressRepo,
	}
}

func (s *addressServiceImpl) GetAddressList(ctx context.Context, userID int64) ([]*repo.Address, error) {
	return s.addressRepo.ListAddressesByUserID(ctx, userID)
}

func (s *addressServiceImpl) GetAddress(ctx context.Context, id, userID int64) (*repo.Address, error) {
	addr, err := s.addressRepo.GetAddressByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if addr == nil || addr.UserID != userID {
		return nil, ErrAddressNotFound
	}
	return addr, nil
}

func (s *addressServiceImpl) AddAddress(ctx context.Context, addr *repo.Address) (int64, error) {
	return s.addressRepo.CreateAddress(ctx, addr)
}

func (s *addressServiceImpl) UpdateAddress(ctx context.Context, addr *repo.Address) error {
	// 检查是否存在
	existing, err := s.addressRepo.GetAddressByID(ctx, addr.ID)
	if err != nil {
		return err
	}
	if existing == nil || existing.UserID != addr.UserID {
		return ErrAddressNotFound
	}
	return s.addressRepo.UpdateAddress(ctx, addr)
}

func (s *addressServiceImpl) DeleteAddress(ctx context.Context, id, userID int64) error {
	return s.addressRepo.DeleteAddress(ctx, id, userID)
}

func (s *addressServiceImpl) SetDefaultAddress(ctx context.Context, id, userID int64) error {
	return s.addressRepo.SetDefaultAddress(ctx, id, userID)
}
