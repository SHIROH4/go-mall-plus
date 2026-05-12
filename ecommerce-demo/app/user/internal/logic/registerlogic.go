package logic

import (
	"context"

	"ecommerce-demo/app/user/internal/svc"
	"ecommerce-demo/app/user/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户注册
func (l *RegisterLogic) Register(in *pb.RegisterReq) (*pb.RegisterResp, error) {
	return l.svcCtx.UserService.Register(l.ctx, in)
}
