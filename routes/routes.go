package routes

import (
	"web_app/controller"
	"web_app/logger"
	"web_app/middlewares"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.New()                                      // 不使用gin.default()来默认中间件
	r.Use(logger.GinLogger(), logger.GinRecovery(true)) //自己注册中间件

	// v1版本
	v1 := r.Group("/api/v1")
	// 注册业务路由
	v1.POST("/signup", controller.SignUpHandler)
	// 登录路由
	v1.POST("/signin", controller.SigninHandler)
	// 认证 验证token中间件
	v1.Use(middlewares.JWTAuthMiddleware())

	// 返回帖子
	{
		v1.GET("/community", controller.GetCommunity) //查询社区
	}

	return r
}
