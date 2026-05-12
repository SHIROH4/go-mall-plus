package mysql

import (
	"context"
	"errors"
	"time"

	"ecommerce-demo/app/payment/internal/model"
	"ecommerce-demo/app/payment/internal/repo"

	"gorm.io/gorm"
)

type paymentRepoImpl struct {
	db *gorm.DB
}

func NewPaymentRepo(db *gorm.DB) *paymentRepoImpl {
	return &paymentRepoImpl{db: db}
}

func (r *paymentRepoImpl) Create(ctx context.Context, payment *model.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *paymentRepoImpl) GetByPaymentNo(ctx context.Context, paymentNo string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.WithContext(ctx).Where("payment_no = ?", paymentNo).First(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repo.ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *paymentRepoImpl) GetByOrderNo(ctx context.Context, orderNo string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repo.ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *paymentRepoImpl) UpdateStatus(ctx context.Context, paymentNo string, status model.PaymentStatus, callbackData string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if callbackData != "" {
		updates["callback_data"] = callbackData
	}
	if status == model.PaymentStatus_Paid {
		now := time.Now()
		updates["pay_time"] = &now
	}

	result := r.db.WithContext(ctx).
		Model(&model.Payment{}).
		Where("payment_no = ?", paymentNo).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrPaymentNotFound
	}
	return nil
}

func (r *paymentRepoImpl) UpdateStatusWithTx(ctx context.Context, paymentNo string, status model.PaymentStatus, callbackData string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var payment model.Payment
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("payment_no = ?", paymentNo).
			First(&payment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repo.ErrPaymentNotFound
			}
			return err
		}

		updates := map[string]interface{}{
			"status": status,
		}
		if callbackData != "" {
			updates["callback_data"] = callbackData
		}
		if status == model.PaymentStatus_Paid {
			now := time.Now()
			updates["pay_time"] = &now
		}

		return tx.Model(&model.Payment{}).
			Where("payment_no = ?", paymentNo).
			Updates(updates).Error
	})
}

func (r *paymentRepoImpl) ListByUserID(ctx context.Context, userID int64, page, pageSize int32) ([]*model.Payment, int32, error) {
	var payments []*model.Payment
	var total int64

	offset := (page - 1) * pageSize

	err := r.db.WithContext(ctx).Model(&model.Payment{}).
		Where("user_id = ?", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Find(&payments).Error

	if err != nil {
		return nil, 0, err
	}

	return payments, int32(total), nil
}

func (r *paymentRepoImpl) GetExpiredPayments(ctx context.Context, limit int) ([]*model.Payment, error) {
	var payments []*model.Payment
	err := r.db.WithContext(ctx).
		Where("status = ? AND expire_time < ?", model.PaymentStatus_Pending, time.Now()).
		Limit(limit).
		Find(&payments).Error
	if err != nil {
		return nil, err
	}
	return payments, nil
}

func (r *paymentRepoImpl) CloseExpiredPayments(ctx context.Context, paymentNos []string) (int64, error) {
	if len(paymentNos) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).
		Model(&model.Payment{}).
		Where("payment_no IN ? AND status = ?", paymentNos, model.PaymentStatus_Pending).
		Where("expire_time < ?", time.Now()).
		Update("status", model.PaymentStatus_Expired)

	return result.RowsAffected, result.Error
}
