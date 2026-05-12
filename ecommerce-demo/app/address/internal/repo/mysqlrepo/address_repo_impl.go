package mysqlrepo

import (
	"context"
	"errors"

	"ecommerce-demo/app/address/internal/repo"

	"gorm.io/gorm"
)

type addressRepoImpl struct {
	db *gorm.DB
}

func NewAddressRepo(db *gorm.DB) repo.AddressRepo {
	return &addressRepoImpl{db: db}
}

func (r *addressRepoImpl) GetAddressByID(ctx context.Context, id int64) (*repo.Address, error) {
	var addr repo.Address
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&addr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addr, nil
}

func (r *addressRepoImpl) ListAddressesByUserID(ctx context.Context, userID int64) ([]*repo.Address, error) {
	var addresses []*repo.Address
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, update_time DESC").
		Find(&addresses).Error
	if err != nil {
		return nil, err
	}
	return addresses, nil
}

func (r *addressRepoImpl) GetDefaultAddress(ctx context.Context, userID int64) (*repo.Address, error) {
	var addr repo.Address
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&addr).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &addr, nil
}

func (r *addressRepoImpl) CreateAddress(ctx context.Context, addr *repo.Address) (int64, error) {
	// 如果设置为默认地址，先清除其他默认地址
	if addr.IsDefault {
		r.db.WithContext(ctx).Model(&repo.Address{}).
			Where("user_id = ?", addr.UserID).
			Update("is_default", false)
	}

	err := r.db.WithContext(ctx).Create(addr).Error
	if err != nil {
		return 0, err
	}
	return addr.ID, nil
}

func (r *addressRepoImpl) UpdateAddress(ctx context.Context, addr *repo.Address) error {
	// 如果设置为默认地址，先清除其他默认地址
	if addr.IsDefault {
		r.db.WithContext(ctx).Model(&repo.Address{}).
			Where("user_id = ?", addr.UserID).
			Update("is_default", false)
	}

	err := r.db.WithContext(ctx).Save(addr).Error
	return err
}

func (r *addressRepoImpl) DeleteAddress(ctx context.Context, id int64, userID int64) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&repo.Address{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("地址不存在或无权限删除")
	}
	return nil
}

func (r *addressRepoImpl) ClearDefaultAddress(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&repo.Address{}).
		Where("user_id = ?", userID).
		Update("is_default", false).Error
}

func (r *addressRepoImpl) SetDefaultAddress(ctx context.Context, id int64, userID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清除所有默认地址
		if err := tx.Model(&repo.Address{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// 设置新默认地址
		result := tx.Model(&repo.Address{}).
			Where("id = ? AND user_id = ?", id, userID).
			Update("is_default", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("地址不存在或无权限")
		}
		return nil
	})
}
