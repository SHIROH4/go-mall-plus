package response

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

var (
	// 系统级错误（不暴露详情）
	ErrSystemBusy    = &serverError{Code: 500, Msg: "系统繁忙，请稍后重试"}
	ErrUnauthorized  = &serverError{Code: 401, Msg: "未授权访问"}
	ErrForbidden     = &serverError{Code: 403, Msg: "权限不足"}
	ErrResourceNotFound = &serverError{Code: 404, Msg: "资源不存在"}
	ErrInternalError = &serverError{Code: 500, Msg: "服务器内部错误"}
)

// serverError 内部错误结构，用于区分系统错误和业务错误
type serverError struct {
	Code int
	Msg  string
}

func (e *serverError) Error() string {
	return e.Msg
}

// IsServerError 判断是否为系统错误
func IsServerError(err error) bool {
	_, ok := err.(*serverError)
	return ok
}

// Response 统一 HTTP 响应封装函数
func Response(w http.ResponseWriter, resp interface{}, err error) {
	if err != nil {
		// 系统错误：返回通用提示，不暴露后端细节
		if se, ok := err.(*serverError); ok {
			httpx.OkJson(w, Body{
				Code: se.Code,
				Msg:  se.Msg,
			})
			return
		}

		// 业务错误：直接返回错误信息
		// 但需要脱敏：隐藏敏感路径、SQL 错误、堆栈信息等
		errMsg := sanitizeError(err.Error())

		httpx.OkJson(w, Body{
			Code: 400, // 业务级错误默认 400
			Msg:  errMsg,
		})
		return
	}

	httpx.OkJson(w, Body{
		Code: 0,
		Msg:  "success",
		Data: resp,
	})
}

// sanitizeError 脱敏错误信息，移除敏感内容
func sanitizeError(errMsg string) string {
	// 常见敏感关键词 - 只有包含这些才隐藏
	sensitivePatterns := []string{
		"password", "token", "secret", "key",
		"sql", "mysql", "gorm",
		"panic", "runtime",
		"/Users/", "/home/", "C:\\",
	}

	lowerMsg := strings.ToLower(errMsg)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerMsg, pattern) {
			return "请求处理失败，请稍后重试"
		}
	}

	// 限制错误信息长度
	if len(errMsg) > 100 {
		return errMsg[:100] + "..."
	}

	return errMsg
}

// Success 快速成功响应
func Success(w http.ResponseWriter, data interface{}) {
	httpx.OkJson(w, Body{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

// Fail 快速失败响应
func Fail(w http.ResponseWriter, msg string) {
	httpx.OkJson(w, Body{
		Code: 400,
		Msg:  msg,
	})
}
