// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/common/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshLogic {
	return &RefreshLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshLogic) Refresh(req *types.RefreshTokenReq) (resp *types.RefreshTokenResp, err error) {
	// 1. 加载 RSA 公钥用于验证 RefreshToken
	publicKey, err := utils.LoadRSAPublicKey(l.svcCtx.Config.Auth.PublicKeyPath)
	if err != nil {
		return nil, errors.New("安全配置异常")
	}

	// 2. 解析并校验 RefreshToken
	claims, err := utils.ParseRsaToken(req.RefreshToken, publicKey)
	if err != nil {
		return nil, errors.New("refresh_token无效或已过期")
	}

	// 3. 必须是 RefreshToken 类型
	if claims.TokenType != utils.TokenTypeRefresh {
		return nil, errors.New("非法的refresh_token")
	}

	// 4. 检查 RefreshToken 是否在黑名单
	blackKey := "refresh_blacklist:" + claims.ID
	exists, _ := l.svcCtx.RDB.Exists(l.ctx, blackKey).Result()
	if exists > 0 {
		return nil, errors.New("refresh_token已失效，请重新登录")
	}

	// 5. 加载 RSA 私钥用于签发新 Token
	privateKey, err := utils.LoadRSAPrivateKey(l.svcCtx.Config.Auth.PrivateKeyPath)
	if err != nil {
		return nil, errors.New("安全配置异常")
	}

	// 6. 签发新的 AccessToken
	accessToken, _, err := utils.GenerateRsaToken(
		privateKey,
		l.svcCtx.Config.Auth.AccessExpire,
		claims.UserId,
		utils.TokenTypeAccess,
	)
	if err != nil {
		return nil, errors.New("token签发失败")
	}

	// 7. 签发新的 RefreshToken (刷新后旧 refresh token 作废)
	refreshToken, newJti, err := utils.GenerateRsaToken(
		privateKey,
		l.svcCtx.Config.Auth.RefreshExpire,
		claims.UserId,
		utils.TokenTypeRefresh,
	)
	if err != nil {
		return nil, errors.New("token签发失败")
	}

	// 8. 将旧 RefreshToken 加入黑名单 (设置与原 RefreshToken 剩余有效期一致)
	// 注意：这里简化处理，实际上应该从 claims.ExpiresAt 计算剩余时间
	l.svcCtx.RDB.Set(l.ctx, blackKey, 1, time.Duration(l.svcCtx.Config.Auth.RefreshExpire)*time.Second)

	// 9. 将新 RefreshToken 的 JTI 存入 Redis (用于管理 Token 状态)
	newRefreshJtiKey := fmt.Sprintf("refresh_jti:%d", claims.UserId)
	l.svcCtx.RDB.Set(l.ctx, newRefreshJtiKey, newJti, time.Duration(l.svcCtx.Config.Auth.RefreshExpire)*time.Second)

	l.Infof("用户 %d 刷新了 Token，JTI: %s -> %s", claims.UserId, claims.ID, newJti)

	return &types.RefreshTokenResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
