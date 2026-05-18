package middleware

import (
	"crypto/rsa"
	"net/http"
	"strings"
	"time"

	"ecommerce-demo/app/gateway/internal/config"
	"ecommerce-demo/common/ctxutil"
	"ecommerce-demo/common/response"
	"ecommerce-demo/common/utils"

	"github.com/redis/go-redis/v9"
)

/*
  AuthMiddleware（加固版）

  改进点：
  1. RSA 公钥启动时加载到内存，不再每次请求读文件（消除磁盘IO瓶颈）
  2. 私钥预加载，支持 AccessToken 无感续期
  3. 当 Token 剩余有效期 < 5分钟时，响应头自动返回新 Token
  4. 黑名单检查保留，确保登出 Token 立即失效
*/

type AuthMiddleware struct {
	Config        config.Config
	RDB           *redis.ClusterClient
	publicKey     *rsa.PublicKey
	privateKey    *rsa.PrivateKey
	accessExpire  int64
	renewThreshold int64 // 续期阈值（秒），默认300秒
}

func NewAuthMiddleware(c config.Config, rdb *redis.ClusterClient) *AuthMiddleware {
	pubKey, err := utils.LoadRSAPublicKey(c.Auth.PublicKeyPath)
	if err != nil {
		panic("加载RSA公钥失败: " + err.Error())
	}

	priKey, err := utils.LoadRSAPrivateKey(c.Auth.PrivateKeyPath)
	if err != nil {
		panic("加载RSA私钥失败: " + err.Error())
	}

	return &AuthMiddleware{
		Config:         c,
		RDB:            rdb,
		publicKey:      pubKey,
		privateKey:     priKey,
		accessExpire:   c.Auth.AccessExpire,
		renewThreshold: 300, // 剩余5分钟时自动续期
	}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 放行 OPTIONS 预检请求（CORS 已在之前处理，此处兜底）
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 2. 获取 Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Fail(w, "缺少身份认证凭证")
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 3. 验签（使用缓存的公钥，不再读文件）
		claims, err := utils.ParseRsaToken(tokenString, m.publicKey)
		if err != nil {
			response.Fail(w, "token无效或已过期")
			return
		}

		// 4. 必须是 AccessToken
		if claims.TokenType != utils.TokenTypeAccess {
			response.Fail(w, "请使用有效的访问令牌")
			return
		}

		// 5. 黑名单校验
		blackKey := "blacklist:" + claims.ID
		exists, _ := m.RDB.Exists(r.Context(), blackKey).Result()
		if exists > 0 {
			response.Fail(w, "令牌已失效，请重新登录")
			return
		}

		// 6. 写入上下文
		ctx := ctxutil.WithUserId(r.Context(), claims.UserId)
		ctx = ctxutil.WithJti(ctx, claims.ID)
		ctx = ctxutil.WithExp(ctx, claims.ExpiresAt.Unix())

		// 7. 无感续期：剩余有效期 < 阈值时，自动签发新 AccessToken
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining > 0 && remaining < time.Duration(m.renewThreshold)*time.Second {
			newToken, _, err := utils.GenerateRsaToken(
				m.privateKey,
				m.accessExpire,
				claims.UserId,
				utils.TokenTypeAccess,
			)
			if err == nil {
				w.Header().Set("X-New-Access-Token", newToken)
			}
		}

		// 8. 放行
		next(w, r.WithContext(ctx))
	}
}
