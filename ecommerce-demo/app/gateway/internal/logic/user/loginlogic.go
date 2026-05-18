package user

import (
	"context"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/app/user/pb"
	"ecommerce-demo/common/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	l.Infof("Login request: username=%s", req.Username)

	// 1. 调用 User RPC 进行密码校验
	rpcResp, err := l.svcCtx.UserRpc.Login(l.ctx, &pb.LoginReq{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		l.Errorf("RPC Login failed: %v", err)
		return nil, err
	}
	l.Infof("RPC Login success, userId=%d", rpcResp.Id)

	// 2. 签发 AccessToken（使用内存中预加载的私钥，不再读文件）
	accessToken, _, err := utils.GenerateRsaToken(
		l.svcCtx.PrivateKey,
		l.svcCtx.Config.Auth.AccessExpire,
		rpcResp.Id,
		utils.TokenTypeAccess,
	)
	if err != nil {
		l.Errorf("Generate AccessToken failed: %v", err)
		return nil, err
	}

	// 3. 签发 RefreshToken
	refreshToken, _, err := utils.GenerateRsaToken(
		l.svcCtx.PrivateKey,
		l.svcCtx.Config.Auth.RefreshExpire,
		rpcResp.Id,
		utils.TokenTypeRefresh,
	)
	if err != nil {
		l.Errorf("Generate RefreshToken failed: %v", err)
		return nil, err
	}

	l.Infof("Login success: userId=%d", rpcResp.Id)

	return &types.LoginResp{
		Id:           rpcResp.Id,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
