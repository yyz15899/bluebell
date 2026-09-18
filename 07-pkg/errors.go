// pkg/errors.go
package pkg

import "fmt"

// BizError 携带业务码的错误,贯穿 logic -> controller
type BizError struct {
	Code ResCode
	Msg  string
	Err  error // 底层原因:只进日志,不给客户端
}

func (e *BizError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Err)
	}
	return e.Msg
}

func (e *BizError) Unwrap() error { return e.Err }

// NewBizError 已知业务错误(用户不存在、参数非法...)
func NewBizError(code ResCode) *BizError {
	return &BizError{Code: code, Msg: code.Msg()}
}

// WrapBizError 未知错误(数据库抖动...)包装后上抛,保留错误链
func WrapBizError(code ResCode, err error) *BizError {
	return &BizError{Code: code, Msg: code.Msg(), Err: err}
}
