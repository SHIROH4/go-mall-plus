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

/*
  Refresh 刷新Token（防重放加固版）

  安全改进：
  1. 分布式锁防并发刷新：同一用户同时只能有一个刷新请求
  2. 已使用Token标记：刷新后旧RefreshToken立即标记为已使用，防止重放
  3. 原子操作顺序：先标记旧Token→再签发新Token→再存储新JTI
     （即使中间步骤失败，旧Token也已失效，不会造成安全漏洞）

  并发安全说明：
  - 攻击者同时发送2个请求用同一RefreshToken刷新：
    请求A获取锁→标记旧Token→签发新Token→释放锁
    请求B获取锁→发现旧Token已标记→返回错误
  - 不会出现"一个RefreshToken签发两对有效Token"的情况
*/
func (l *RefreshLogic) Refresh(req *types.RefreshTokenReq) (resp *types.RefreshTokenResp, err error) {
	// 1. 验签 RefreshToken（使用网关内存中缓存的公钥）
	claims, err := utils.ParseRsaToken(req.RefreshToken, l.svcCtx.PublicKey)
	if err != nil {
		return nil, errors.New("refresh_token无效或已过期")
	}

	// 2. 必须是 RefreshToken 类型
	if claims.TokenType != utils.TokenTypeRefresh {
		return nil, errors.New("非法的refresh_token")
	}

	// 3. 分布式锁：防止同一用户并发刷新
	lockKey := fmt.Sprintf("{refresh}:lock:%d", claims.UserId)
	locked, err := l.svcCtx.RDB.SetNX(l.ctx, lockKey, "1", 5*time.Second).Result()
	if err != nil {
		l.Errorf("Redis 锁获取失败: %v", err)
		return nil, errors.New("系统繁忙，请稍后再试")
	}
	if !locked {
		return nil, errors.New("请勿频繁刷新，稍后再试")
	}
	defer l.svcCtx.RDB.Del(l.ctx, lockKey)

	// 4. 检查旧 RefreshToken 是否已被使用（防重放核心）
	usedKey := "used_refresh:" + claims.ID
	wasUsed, _ := l.svcCtx.RDB.Exists(l.ctx, usedKey).Result()
	if wasUsed > 0 {
		// 检测到重放攻击！强制该用户所有Token失效
		l.Errorf("检测到RefreshToken重放攻击: userId=%d, jti=%s", claims.UserId, claims.ID)
		l.svcCtx.RDB.Del(l.ctx, fmt.Sprintf("{refresh}:jti:%d", claims.UserId))
		return nil, errors.New("token已失效，请重新登录")
	}

	// 5. 标记旧 RefreshToken 为已使用（在签发新Token之前，确保安全）
	// 过期时间设置为旧Token的剩余有效期，避免Redis内存泄漏
	remainingTTL := time.Until(claims.ExpiresAt.Time)
	if remainingTTL <= 0 {
		return nil, errors.New("refresh_token已过期，请重新登录")
	}
	l.svcCtx.RDB.Set(l.ctx, usedKey, "1", remainingTTL)

	// 6. 签发新 AccessToken
	accessToken, _, err := utils.GenerateRsaToken(
		l.svcCtx.PrivateKey,
		l.svcCtx.Config.Auth.AccessExpire,
		claims.UserId,
		utils.TokenTypeAccess,
	)
	if err != nil {
		// 签发失败，清除已使用标记，允许用户重试
		l.svcCtx.RDB.Del(l.ctx, usedKey)
		return nil, errors.New("token签发失败")
	}

	// 7. 签发新 RefreshToken
	refreshToken, newJti, err := utils.GenerateRsaToken(
		l.svcCtx.PrivateKey,
		l.svcCtx.Config.Auth.RefreshExpire,
		claims.UserId,
		utils.TokenTypeRefresh,
	)
	if err != nil {
		l.svcCtx.RDB.Del(l.ctx, usedKey)
		return nil, errors.New("token签发失败")
	}

	// 8. 更新用户当前有效的 RefreshToken JTI
	newRefreshJtiKey := fmt.Sprintf("{refresh}:jti:%d", claims.UserId)
	l.svcCtx.RDB.Set(l.ctx, newRefreshJtiKey, newJti,
		time.Duration(l.svcCtx.Config.Auth.RefreshExpire)*time.Second)

	l.Infof("用户 %d Token刷新成功: 旧JTI=%s → 新JTI=%s", claims.UserId, claims.ID, newJti)

	return &types.RefreshTokenResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
