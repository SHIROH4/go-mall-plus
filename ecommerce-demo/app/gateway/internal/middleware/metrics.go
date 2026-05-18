package middleware

import (
	"net/http"
	"strconv"
	"time"

	"ecommerce-demo/common/metrics"
)

// statusRecorder 包装 ResponseWriter 以捕获状态码
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware 记录每个 HTTP 请求的 QPS 和延迟
func MetricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next(sr, r)

		metrics.HTTPRequestTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(sr.statusCode)).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())
	}
}
