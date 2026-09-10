package middlewares

import (
	"strings"
	"web_app/dao/redis"
	"web_app/pkg"
	"web_app/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// token认证中间件

const ContextUserIDKey = "userID"

func JWTAuthMiddleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		// token形式: Authorization: Bearer ......
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			pkg.ResponseError(c, pkg.CodeNeedLogin) //没有登陆导致没有请求参数
			c.Abort()
			return
		}
		// 按照空格分隔校验格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			pkg.ResponseError(c, pkg.CodeInvalidToken)
			c.Abort()
			return
		}
		mc, err := jwt.ParseToken(parts[1])
		if err != nil {
			pkg.ResponseError(c, pkg.CodeInvalidToken)
			c.Abort()
			return
		}

		// 挤下线校验
		// 拿到本次请求中redis中有效的token
		curToken, err := redis.GetLoginToken(mc.UserId)
		if err != nil || curToken != parts[1] {
			// nil 未登录/无效token
			// !=  token不同(账号已在别处登录,原先token被挤掉了)
			pkg.ResponseError(c, pkg.CodeInvalidToken)
			c.Abort()
			return
		}
		// 接收到
		c.Set(ContextUserIDKey, mc.UserId)
		c.Next()
	}
}
