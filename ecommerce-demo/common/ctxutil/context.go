package ctxutil

import (
	"context"
	"errors"
)

type contextKey string

const (
	UserIdKey contextKey = "userId"
	JtiKey    contextKey = "jti"
	ExpKey    contextKey = "exp"
)

// ErrUserNotFound 用户未登录或登录信息丢失
var ErrUserNotFound = errors.New("用户未登录")

// GetUserId 安全获取用户 ID（统一类型，所有 Logic 调用此方法）
func GetUserId(ctx context.Context) (int64, error) {
	val := ctx.Value(UserIdKey)
	if val == nil {
		return 0, ErrUserNotFound
	}

	switch v := val.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	default:
		return 0, ErrUserNotFound
	}
}

// GetJti 获取 JWT ID
func GetJti(ctx context.Context) (string, error) {
	val := ctx.Value(JtiKey)
	if val == nil {
		return "", errors.New("token标识丢失")
	}

	jti, ok := val.(string)
	if !ok {
		return "", errors.New("token标识格式错误")
	}
	return jti, nil
}

// GetExp 获取过期时间戳
func GetExp(ctx context.Context) (int64, error) {
	val := ctx.Value(ExpKey)
	if val == nil {
		return 0, errors.New("token过期时间丢失")
	}

	switch v := val.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	default:
		return 0, errors.New("token过期时间格式错误")
	}
}

// WithUserId 向 context 注入用户 ID（中间件使用）
func WithUserId(ctx context.Context, userId int64) context.Context {
	return context.WithValue(ctx, UserIdKey, userId)
}

// WithJti 向 context 注入 JTI（中间件使用）
func WithJti(ctx context.Context, jti string) context.Context {
	return context.WithValue(ctx, JtiKey, jti)
}

// WithExp 向 context 注入过期时间（中间件使用）
func WithExp(ctx context.Context, exp int64) context.Context {
	return context.WithValue(ctx, ExpKey, exp)
}
