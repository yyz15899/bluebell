package routes

import (
	"net/http"
	"web_app/logger"

	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.New()                                      // 不使用gin.default()来默认中间件
	r.Use(logger.GinLogger(), logger.GinRecovery(true)) //自己注册中间件
	r.GET("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, "ok")
	})
	return r
}
