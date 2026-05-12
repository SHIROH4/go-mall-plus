package utils

import (
	"crypto/rsa"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// 定义 Token 类型常量
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// CustomClaims 自定义 JWT Payload
type CustomClaims struct {
	UserId    int64  `json:"userId"`
	TokenType string `json:"tokenType"` // 区分长短 token
	jwt.RegisteredClaims
}

// LoadRSAPrivateKey 从文件加载私钥 (用于签发)
func LoadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPrivateKeyFromPEM(keyData)
}

// LoadRSAPublicKey 从文件加载公钥 (用于验证)
func LoadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jwt.ParseRSAPublicKeyFromPEM(keyData)
}

// GenerateRsaToken 签发基于 RSA 非对称加密的 Token
func GenerateRsaToken(
	privateKey *rsa.PrivateKey,
	expireSeconds int64,
	userId int64,
	tokenType string,
) (string, string, error) {
	// ========== 安全校验 ==========
	if userId <= 0 {
		return "", "", errors.New("userId 必须大于0")
	}
	if tokenType != TokenTypeAccess && tokenType != TokenTypeRefresh {
		return "", "", errors.New("非法的 token 类型")
	}
	if expireSeconds <= 0 {
		return "", "", errors.New("过期时间必须大于0")
	}

	// 1. 生成 JTI (JWT ID，用于黑名单)
	jti := uuid.NewString()

	now := time.Now()

	// 2. 构建标准 Claims
	claims := CustomClaims{
		UserId:    userId,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireSeconds) * time.Second)),
		},
	}

	// 3. 使用 RS256 算法签名
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// 4. 用私钥签名
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", "", err
	}

	return tokenString, jti, nil
}

// ParseRsaToken 解析并校验 RSA Token
func ParseRsaToken(tokenString string, publicKey *rsa.PublicKey) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 校验算法必须是 RSA
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("非法签名算法")
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	// 校验 token 是否有效
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的 Token")
}
