package pkg

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResponseData struct {
	Code ResCode     `json:"code"`
	Msg  string      `json:"msg"` // 类型修正为 string
	Data interface{} `json:"data"`
}

// ResponseSuccess 成功响应（修复了 data 未赋值的问题）
func ResponseSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, &ResponseData{
		Code: CodeSuccess,
		Msg:  CodeSuccess.Msg(),
		Data: data, // 修复：正确传入 data
	})
}

// ResponseError 通用错误响应（替代了原先重复的多个 Error 函数）
func ResponseError(c *gin.Context, code ResCode) {
	c.JSON(http.StatusOK, &ResponseData{
		Code: code,
		Msg:  code.Msg(),
		Data: nil,
	})
}

// ResponseErrorWithMsg 支持自定义提示信息的错误响应（扩展选配）
func ResponseErrorWithMsg(c *gin.Context, code ResCode, msg string) {
	c.JSON(http.StatusOK, &ResponseData{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}
