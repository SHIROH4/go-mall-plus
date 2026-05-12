package logic

import (
	"context"

	"ecommerce-demo/app/user/internal/svc"
	"ecommerce-demo/app/user/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户登录
func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginResp, error) {
	return l.svcCtx.UserService.Login(l.ctx, in)
}
