package routes

import (
	controller "web_app/01-controller"
	logger "web_app/04-logger"
	middlewares "web_app/06-middlewares"

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
	// 登出路由
	v1.GET("/signout", controller.SignoutHandler)
	// 修改密码
	v1.PATCH("/update", controller.UpdateHandler)

	// 返回帖子
	{
		v1.GET("/community", controller.GetCommunities) //查询社区
		v1.GET("/community/:id", controller.GetCommunity)

		v1.POST("/post", controller.CreatePost) // 发布帖子
		v1.GET("/post/:postid", controller.GetPost)
		v1.DELETE("/post/:postid", controller.DeletePost)
		v1.GET("/posts", controller.GetPostList)          // 分页post
		v1.GET("/posts/hot", controller.GetPostListByHot) // 分页热榜post

		//post点赞
		v1.POST("/like", controller.PostLikeHandler)
	}

	return r
}
