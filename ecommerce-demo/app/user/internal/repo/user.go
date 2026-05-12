package repo

import (
	"context"
	"time"
)

// User 数据库实体模型，对应 MySQL 的 user 表
type User struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Username   string    `gorm:"column:username"`
	Password   string    `gorm:"column:password"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

// TableName 显式指定表名，防止 gorm 自动加 's'
func (User) TableName() string {
	return "user"
}

// UserRepo 仓储层接口定义 (业务层只认这个接口，不认具体的 GORM 实现)
type UserRepo interface {
	// CreateUser 创建用户 (注册)
	CreateUser(ctx context.Context, user *User) error
	// GetUserByUsername 根据用户名查询 (登录、注册防重)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	// GetUserByID 根据ID查询 (获取信息)
	GetUserByID(ctx context.Context, id int64) (*User, error)
}
