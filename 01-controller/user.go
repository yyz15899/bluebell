package controller

import (
	logic "web_app/02-logic"
	"web_app/03-dao/redis"
	models "web_app/05-models"
	middlewares "web_app/06-middlewares"
	pkg "web_app/07-pkg"
	"web_app/07-pkg/jwt"

	"github.com/gin-gonic/gin"
)

// jwt.go

func SignUpHandler(c *gin.Context) {
	// 1:获取参数和参数校验
	// var p models.ParamSignUp
	p := new(models.ParamSignUp) // 创建指针类型
	// shouldbindjson() 会将客户通过post请求传递的参数信息写入结构体中(顺带进行参数校验)
	if err := c.ShouldBindJSON(p); err != nil {
		//请求参数有误
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	// 校验参数
	// 方案2
	// 直接使用validator库对params中的结构体进行校验(binding tag)

	// 2:业务处理(logic层(文件夹))
	err := logic.SignUp(p)
	if err != nil {
		ResponseWithError(c, err)
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
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	// 业务处理
	userid, err := logic.SignIn(p)
	if err != nil {
		ResponseWithError(c, err)
		return
	}
	token, err := jwt.GenToken(userid, p.Username) // 发放token
	if err != nil {
		pkg.ResponseError(c, pkg.CodeServerBusy)
		return
	}

	// 存储该用户该次登陆发的token和该用户的userid 设置存储时间		将token等存到了redis中
	// 用本次登录的新 token 覆盖 redis 中的旧记录
	// (若 A 先登录存 tokenA, B 再登录这里覆盖成 tokenB → A 的 tokenA 下次请求就会被中间件拦下)
	if err := redis.SaveLoginToken(userid, token, jwt.TokenExpireDuration); err != nil {
		pkg.ResponseError(c, pkg.CodeServerBusy) // redis保存失败
		return
	}
	// 返回响应
	pkg.ResponseSuccess(c, token)
}

// SignoutHandler 登出
func SignoutHandler(c *gin.Context) {
	// 业务处理
	// 拿到userid
	uid := middlewares.ContextUserIDKey
	userid := c.GetInt64(uid)
	if err := logic.Signout(userid); err != nil {
		pkg.ResponseError(c, pkg.CodeServerBusy)
		return
	}

	// 返回响应
	pkg.ResponseSuccess(c, "signout success")
}

// UpdateHandler  修改信息
func UpdateHandler(c *gin.Context) {
	p := new(models.ParamUpdate)
	if err := c.ShouldBindJSON(p); err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	uid := c.GetInt64(middlewares.ContextUserIDKey)
	if err := logic.UpdateInfo(uid, p); err != nil {
		ResponseWithError(c, err)
		return
	}
	pkg.ResponseSuccess(c, "success")
}
