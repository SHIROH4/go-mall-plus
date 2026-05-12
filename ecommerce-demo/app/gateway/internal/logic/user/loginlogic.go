// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

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

	// 2. 加载 RSA 私钥
	privateKey, err := utils.LoadRSAPrivateKey(l.svcCtx.Config.Auth.PrivateKeyPath)
	if err != nil {
		l.Errorf("Load RSA private key failed: %v, path=%s", err, l.svcCtx.Config.Auth.PrivateKeyPath)
		return nil, err
	}
	l.Infof("RSA private key loaded successfully")

	// 3. 签发 AccessToken (短效)
	accessToken, _, err := utils.GenerateRsaToken(
		privateKey,
		l.svcCtx.Config.Auth.AccessExpire,
		rpcResp.Id,
		utils.TokenTypeAccess,
	)
	if err != nil {
		l.Errorf("Generate AccessToken failed: %v", err)
		return nil, err
	}

	// 4. 签发 RefreshToken (长效)
	refreshToken, _, err := utils.GenerateRsaToken(
		privateKey,
		l.svcCtx.Config.Auth.RefreshExpire,
		rpcResp.Id,
		utils.TokenTypeRefresh,
	)
	if err != nil {
		l.Errorf("Generate RefreshToken failed: %v", err)
		return nil, err
	}

	l.Infof("Login success: userId=%d, accessTokenLen=%d", rpcResp.Id, len(accessToken))

	return &types.LoginResp{
		Id:           rpcResp.Id,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
