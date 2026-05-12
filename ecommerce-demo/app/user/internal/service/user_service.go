package service

import (
	"context"
	"errors"

	"ecommerce-demo/app/user/internal/repo"
	"ecommerce-demo/app/user/pb"
	"ecommerce-demo/common/utils"
)

// 定义业务层常见的错误
var (
	ErrUserExists   = errors.New("用户名已存在")
	ErrUserNotFound = errors.New("用户不存在")
	ErrWrongPwd     = errors.New("密码错误")
)

// UserService 定义业务层接口
type UserService interface {
	Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error)
	Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error)
	GetUserInfo(ctx context.Context, req *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error)
}

// userServiceImpl 业务层实现类
type userServiceImpl struct {
	userRepo repo.UserRepo // 依赖仓储层接口，不依赖具体实现
}

// NewUserService 构造函数注入
func NewUserService(userRepo repo.UserRepo) UserService {
	return &userServiceImpl{
		userRepo: userRepo,
	}
}

// Register 核心注册业务
func (s *userServiceImpl) Register(ctx context.Context, req *pb.RegisterReq) (*pb.RegisterResp, error) {
	// 1. 业务校验：检查用户名是否已存在
	existUser, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existUser != nil {
		return nil, ErrUserExists // 抛出纯粹的业务错误
	}

	// 2. 数据处理：密码哈希加密
	hashPwd, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 3. 组装实体
	newUser := &repo.User{
		Username: req.Username,
		Password: hashPwd,
	}

	// 4. 落库
	if err := s.userRepo.CreateUser(ctx, newUser); err != nil {
		return nil, err
	}

	return &pb.RegisterResp{
		Id: newUser.ID, // CreateUser 后 GORM 会自动回填 ID
	}, nil
}

// Login 核心登录业务
func (s *userServiceImpl) Login(ctx context.Context, req *pb.LoginReq) (*pb.LoginResp, error) {
	// 1. 根据用户名查用户
	user, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 2. 校验密码哈希
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, ErrWrongPwd
	}

	return &pb.LoginResp{
		Id: user.ID,
	}, nil
}

// GetUserInfo 核心查询业务
func (s *userServiceImpl) GetUserInfo(ctx context.Context, req *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	user, err := s.userRepo.GetUserByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return &pb.GetUserInfoResp{
		Id:       user.ID,
		Username: user.Username,
	}, nil
}
