package controller

import (
	"web_app/logic"
	"web_app/pkg"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GetCommunity(c *gin.Context) {
	// 查询所有社区(community_id,community_name) 以列表形式返回
	data, err := logic.GetCommunityList()
	if err != nil {
		zap.L().Error("logic.GetCommunityList failed", zap.Error(err)) // 记录日志
		pkg.ResponseError(c, pkg.CodeServerBusy)
		return
	}
	pkg.ResponseSuccess(c, data)
}
