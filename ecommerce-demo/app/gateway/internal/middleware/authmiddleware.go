package middleware

import (
	"net/http"
	"strings"

	"ecommerce-demo/app/gateway/internal/config"
	"ecommerce-demo/common/ctxutil"
	"ecommerce-demo/common/response"
	"ecommerce-demo/common/utils"

	"github.com/redis/go-redis/v9"
)

type AuthMiddleware struct {
	Config config.Config
	RDB    *redis.Client
}

func NewAuthMiddleware(c config.Config, rdb *redis.Client) *AuthMiddleware {
	return &AuthMiddleware{
		Config: c,
		RDB:    rdb,
	}
}

// Handle go-zero 标准中间件签名
func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 放行 OPTIONS 预检请求（跨域需要）
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 2. 获取 Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Fail(w, "缺少身份认证凭证")
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 2. 验签
		publicKey, err := utils.LoadRSAPublicKey(m.Config.Auth.PublicKeyPath)
		if err != nil {
			response.Fail(w, "服务配置异常，请联系管理员")
			return
		}

		claims, err := utils.ParseRsaToken(tokenString, publicKey)
		if err != nil {
			response.Fail(w, "token无效或已过期")
			return
		}

		// 3. 必须是 AccessToken
		if claims.TokenType != utils.TokenTypeAccess {
			response.Fail(w, "请使用有效的访问令牌")
			return
		}

		// 4. 黑名单校验
		blackKey := "blacklist:" + claims.ID
		exists, _ := m.RDB.Exists(r.Context(), blackKey).Result()
		if exists > 0 {
			response.Fail(w, "令牌已失效，请重新登录")
			return
		}

		// 5. 统一方式写入上下文（使用 ctxutil）
		ctx := ctxutil.WithUserId(r.Context(), claims.UserId)
		ctx = ctxutil.WithJti(ctx, claims.ID)
		ctx = ctxutil.WithExp(ctx, claims.ExpiresAt.Unix())

		// 6. 放行
		next(w, r.WithContext(ctx))
	}
}
