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
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout() (resp *types.LogoutResp, err error) {
	// 使用统一的 ctxutil 获取 token 信息
	jti, err := ctxutil.GetJti(l.ctx)
	if err != nil {
		return nil, errors.New("系统异常：无法获取 Token 标识")
	}

	exp, err := ctxutil.GetExp(l.ctx)
	if err != nil {
		return nil, errors.New("系统异常：无法获取 Token 过期时间")
	}

	// 计算还有多久过期
	now := time.Now().Unix()
	remainSeconds := exp - now

	// 1. 将 AccessToken 加入黑名单
	if remainSeconds > 0 {
		err = l.svcCtx.RDB.Set(l.ctx, "blacklist:"+jti, 1, time.Duration(remainSeconds)*time.Second).Err()
		if err != nil {
			l.Errorf("Redis 写入黑名单失败: %v", err)
			return nil, err
		}
	}

	// 2. 使该用户的 RefreshToken 也失效
	userId, err := ctxutil.GetUserId(l.ctx)
	if err == nil && userId > 0 {
		l.svcCtx.RDB.Del(l.ctx, fmt.Sprintf("refresh_jti:%d", userId))
	}

	l.Infof("用户登出，Access JTI: %s", jti)

	return &types.LogoutResp{}, nil
}
