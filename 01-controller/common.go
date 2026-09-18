package controller

import (
	"errors"
	pkg "web_app/07-pkg"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ResponseWithError(c *gin.Context, err error) {
	var be *pkg.BizError
	if errors.As(err, &be) { // 业务错误
		zap.L().Debug("biz error", zap.String("path", c.FullPath()), zap.Error(err)) // 业务结果:Debug
		pkg.ResponseError(c, be.Code)
		return
	}
	// 系统错误
	zap.L().Error("internal error", zap.String("path", c.FullPath()), zap.Error(err)) // 真故障:Error
	pkg.ResponseError(c, pkg.CodeServerBusy)
}
