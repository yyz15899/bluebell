package routes

import (
	"net/http"
	"web_app/controller"
	"web_app/logger"
	"web_app/middlewares"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.New()                                      // 不使用gin.default()来默认中间件
	r.Use(logger.GinLogger(), logger.GinRecovery(true)) //自己注册中间件

	// 注册业务路由
	r.POST("/signup", controller.SignUpHandler)

	// 登录路由
	r.POST("/signin", controller.SigninHandler)
	// 加上jwt中间件
	r.GET("/login", middlewares.JWTAuthMiddleware(), func(c *gin.Context) {
		// 如果是已登录用户 返回pong
		// 判断请求头中是否有 有效的jwt
		c.String(http.StatusOK, "pong")
	})
	return r
}
