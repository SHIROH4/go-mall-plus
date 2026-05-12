package validator

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// CustomValidator 封装 validator.Validate
type CustomValidator struct {
	Validator *validator.Validate
}

// NewCustomValidator 构造函数
func NewCustomValidator() *CustomValidator {
	return &CustomValidator{
		Validator: validator.New(),
	}
}

// Validate 实现 go-zero 内部的 httpx.Validator 接口
// 只要实现了这个接口，go-zero 就会在解析参数时自动调用它！
func (cv *CustomValidator) Validate(r *http.Request, data interface{}) error {
	err := cv.Validator.Struct(data)
	if err != nil {
		// 遇到校验错误，直接抛出。可以根据业务需要翻译成中文，这里演示抛出第一个校验错误
		for _, e := range err.(validator.ValidationErrors) {
			return errors.New("参数校验失败: 字段 " + e.Field() + " 不符合规则 " + e.Tag())
		}
	}
	return nil
}
