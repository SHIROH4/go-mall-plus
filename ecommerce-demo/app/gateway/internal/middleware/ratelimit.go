package middleware

import (
	"net/http"
	"sync"
	"time"

	"ecommerce-demo/common/response"
)

/*
  限流中间件（分层限流策略 — 内存令牌桶实现）

  三层防护：
  L1 - 全局限流: 所有请求共享令牌桶，防止整体过载
  L2 - 路由限流: 针对特定接口的独立限流（如下单、登录）
  L3 - IP 限流:  每个客户端IP独立限流，防止单IP刷接口

  配置示例:
    RateLimit:
      Enabled: true
      Global:
        Rate: 1000    # 每秒1000请求
        Burst: 2000
      PerIP:
        Rate: 50      # 每IP每秒50请求
        Burst: 100
      Routes:
        /api/order/create: {Rate: 5, Burst: 10}
        /api/user/login:   {Rate: 3, Burst: 5}
*/

type RateLimitConfig struct {
	Enabled bool
	Global  struct {
		Rate  int
		Burst int
	}
	PerIP struct {
		Rate  int
		Burst int
	}
	Routes map[string]struct {
		Rate  int
		Burst int
	}
}

// tokenBucket 内存令牌桶
type tokenBucket struct {
	rate      float64 // 每秒产生令牌数
	burst     float64 // 桶容量
	tokens    float64
	lastTime  time.Time
	mu        sync.Mutex
}

func newTokenBucket(rate, burst int) *tokenBucket {
	return &tokenBucket{
		rate:     float64(rate),
		burst:    float64(burst),
		tokens:   float64(burst), // 初始满桶
		lastTime: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.burst {
		tb.tokens = tb.burst
	}
	tb.lastTime = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

type RateLimitMiddleware struct {
	config      RateLimitConfig
	globalLim   *tokenBucket
	ipLimits    sync.Map // map[string]*tokenBucket
	routeLimits map[string]*tokenBucket
}

func NewRateLimitMiddleware(cfg RateLimitConfig) *RateLimitMiddleware {
	m := &RateLimitMiddleware{
		config:      cfg,
		routeLimits: make(map[string]*tokenBucket),
	}

	if cfg.Global.Rate > 0 {
		burst := cfg.Global.Burst
		if burst <= 0 {
			burst = cfg.Global.Rate * 2
		}
		m.globalLim = newTokenBucket(cfg.Global.Rate, burst)
	}

	for path, rc := range cfg.Routes {
		if rc.Rate > 0 {
			burst := rc.Burst
			if burst <= 0 {
				burst = rc.Rate * 2
			}
			m.routeLimits[path] = newTokenBucket(rc.Rate, burst)
		}
	}

	return m
}

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.config.Enabled {
			next(w, r)
			return
		}

		// L1: 全局限流
		if m.globalLim != nil && !m.globalLim.allow() {
			response.FailWithStatus(w, "系统繁忙，请稍后再试", http.StatusTooManyRequests)
			return
		}

		// L2: 路由级特殊限流
		if routeLim, ok := m.routeLimits[r.URL.Path]; ok {
			if !routeLim.allow() {
				response.FailWithStatus(w, "操作太频繁，请稍后再试", http.StatusTooManyRequests)
				return
			}
		}

		// L3: IP 限流
		if m.config.PerIP.Rate > 0 {
			clientIP := getClientIP(r)
			ipLim := m.getOrCreateIPLimiter(clientIP)
			if !ipLim.allow() {
				response.FailWithStatus(w, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
				return
			}
		}

		next(w, r)
	}
}

func (m *RateLimitMiddleware) getOrCreateIPLimiter(ip string) *tokenBucket {
	if lim, ok := m.ipLimits.Load(ip); ok {
		return lim.(*tokenBucket)
	}
	burst := m.config.PerIP.Burst
	if burst <= 0 {
		burst = m.config.PerIP.Rate * 2
	}
	lim := newTokenBucket(m.config.PerIP.Rate, burst)
	actual, _ := m.ipLimits.LoadOrStore(ip, lim)
	return actual.(*tokenBucket)
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host := r.RemoteAddr
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}
