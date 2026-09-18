package controller

import (
	"strconv"
	logic "web_app/02-logic"
	pkg "web_app/07-pkg"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// 获取所有社区(列表)
func GetCommunities(c *gin.Context) {
	// 查询所有社区(community_id,community_name) 以列表形式返回
	data, err := logic.GetCommunityList()
	if err != nil {
		zap.L().Error("logic.GetCommunityList failed", zap.Error(err)) // 记录日志
		pkg.ResponseError(c, pkg.CodeServerBusy)
		return
	}
	pkg.ResponseSuccess(c, data)
}

// 社区详情
func GetCommunity(c *gin.Context) {
	// 获取社区id
	communityID := c.Param("id")
	id, err := strconv.ParseInt(communityID, 10, 64) //转译为int64类型
	if err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	detail, err := logic.GetCommunity(id)
	if err != nil {
		ResponseWithError(c, err)
		return
	}
	pkg.ResponseSuccess(c, detail)
}
