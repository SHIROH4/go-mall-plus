package middleware

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

/*
  CORS 中间件 + 安全响应头

  功能：
  1. 跨域请求处理（OPTIONS 预检 + 响应头）
  2. Origin 白名单校验
  3. 安全响应头注入（XSS防护、MIME嗅探防护、点击劫持防护）
*/

type CORSConfig struct {
	Enabled        bool
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
		MaxAge:         3600,
	}
}

type CORSMiddleware struct {
	config CORSConfig
}

func NewCORSMiddleware(cfg CORSConfig) *CORSMiddleware {
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 3600
	}
	return &CORSMiddleware{config: cfg}
}

func (m *CORSMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 安全响应头（每次请求都注入）
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if !m.config.Enabled {
			next(w, r)
			return
		}

		origin := r.Header.Get("Origin")

		// Origin 白名单校验
		if origin != "" && len(m.config.AllowedOrigins) > 0 {
			if !slices.Contains(m.config.AllowedOrigins, origin) && !slices.Contains(m.config.AllowedOrigins, "*") {
				next(w, r)
				return
			}
		}

		if origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if len(m.config.AllowedOrigins) == 0 || slices.Contains(m.config.AllowedOrigins, "*") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.config.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.config.AllowedHeaders, ", "))
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(m.config.MaxAge))

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}
