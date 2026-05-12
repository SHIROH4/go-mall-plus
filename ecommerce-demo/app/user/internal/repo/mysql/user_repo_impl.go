package mysql

import (
	"context"
	"errors"
	"gorm.io/gorm"

	"ecommerce-demo/app/user/internal/repo"
)

// userRepoImpl 是 UserRepo 接口的 MySQL 实现
type userRepoImpl struct {
	db *gorm.DB
}

// NewUserRepo 构造函数，返回接口类型，强制面向接口编程
func NewUserRepo(db *gorm.DB) repo.UserRepo {
	return &userRepoImpl{
		db: db,
	}
}

func (r *userRepoImpl) CreateUser(ctx context.Context, user *repo.User) error {
	// WithContext 传递链路追踪上下文
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepoImpl) GetUserByUsername(ctx context.Context, username string) (*repo.User, error) {
	var user repo.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没找到不当作严重错误，返回 nil 让业务层判断
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepoImpl) GetUserByID(ctx context.Context, id int64) (*repo.User, error) {
	var user repo.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
