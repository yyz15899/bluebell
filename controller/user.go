package controller

import (
	"web_app/dao/redis"
	"web_app/logic"
	"web_app/models"
	"web_app/pkg"
	"web_app/pkg/jwt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// jwt.go

func SignUpHandler(c *gin.Context) {
	// 1:获取参数和参数校验
	// var p models.ParamSignUp
	p := new(models.ParamSignUp) // 创建指针类型
	// shouldbindjson() 会将客户通过post请求传递的参数信息写入结构体中(顺带进行参数校验)
	if err := c.ShouldBindJSON(p); err != nil {
		//请求参数有误
		zap.L().Error("SignUp with invaild param", zap.Error(err)) // 记录日志
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	// 校验参数
	// 方案2
	// 直接使用validator库对params中的结构体进行校验(binding tag)

	// 2:业务处理(logic层(文件夹))
	if err := logic.SignUp(p); err != nil {
		zap.L().Error("logic.SignUp failed", zap.Error(err))

		// 关键点：将 CodeServerBusy 作为基础 Code，直接用 err.Error() 的提示文字覆盖默认 Msg
		pkg.ResponseErrorWithMsg(c, pkg.CodeServerBusy, err.Error())
		return
	}
	// 3: 返回响应
	// c.JSON(http.StatusOK, gin.H{
	// 	"msg": "SignUp success!!!",
	// })
	pkg.ResponseSuccess(c, nil)
}

func SigninHandler(c *gin.Context) {
	// 参数校验
	p := new(models.ParamSignIn)
	if err := c.ShouldBindJSON(p); err != nil {
		zap.L().Error("SignIn with invaild param", zap.Error(err))
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	// 业务处理
	userid, err := logic.SignIn(p)
	if err != nil {
		zap.L().Error("logic signin failed", zap.Error(err))
		pkg.ResponseError(c, pkg.CodeInvalidPassword) // 密码名或密码错误
		return
	}
	token, err := jwt.GenToken(userid, p.Username) // 发放token
	if err != nil {
		zap.L().Error("token issue failed", zap.Error(err))
		pkg.ResponseError(c, pkg.CodeServerBusy)
		return
	}

	// 存储该用户该次登陆发的token和该用户的userid 设置存储时间		将token等存到了redis中
	// 用本次登录的新 token 覆盖 redis 中的旧记录
	// (若 A 先登录存 tokenA, B 再登录这里覆盖成 tokenB → A 的 tokenA 下次请求就会被中间件拦下)
	if err := redis.SaveLoginToken(userid, token, jwt.TokenExpireDuration); err != nil {
		zap.L().Error("SaveLoginToken failed", zap.Int64("user_id", userid), zap.Error(err))
		pkg.ResponseError(c, pkg.CodeServerBusy) // redis保存失败
		return
	}
	// 返回响应
	pkg.ResponseSuccess(c, token)
}
