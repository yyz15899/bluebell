package controller

import (
	"net/http"
	"web_app/logic"
	"web_app/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SignUpHandler(c *gin.Context) {
	// 1:获取参数和参数校验
	// var p models.ParamSignUp
	p := new(models.ParamSignUp) // 创建指针类型
	// shouldbindjson() 会将客户通过post请求传递的参数信息写入结构体中(顺带进行参数校验)
	if err := c.ShouldBindJSON(p); err != nil {
		//请求参数有误
		zap.L().Error("SignUp with invaild param", zap.Error(err)) // 记录日志
		c.JSON(http.StatusOK, gin.H{
			"msg": "请求参数有误",
		})
		return
	}
	// fmt.Println(p)
	// 方案一
	// 手动校验空值与密码不一样
	// if len(p.Password) == 0 || len(p.RePassword) == 0 || len(p.Username) == 0 || p.RePassword != p.Password {
	// 	//请求参数有误
	// 	zap.L().Error("SignUp with invaild param") // 记录日志
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"msg": "请求参数有误",
	// 	})
	// 	return
	// }

	// 方案2
	// 直接使用validator库对params中的结构体进行校验(binding tag)

	// 2:业务处理(logic层(文件夹))
	if err := logic.SignUp(p); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg": "注册失败",
		})
		return
	}
	// 3: 返回响应
	c.JSON(http.StatusOK, gin.H{
		"msg": "SignUp success!!!",
	})
}
