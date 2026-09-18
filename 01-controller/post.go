package controller

import (
	"strconv"
	logic "web_app/02-logic"
	models "web_app/05-models"
	middlewares "web_app/06-middlewares"
	pkg "web_app/07-pkg"

	"github.com/gin-gonic/gin"
)

func CreatePost(c *gin.Context) {
	// 接收,校验参数
	p := new(models.ParamCreatePost)
	if err := c.ShouldBind(p); err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	//  用户如果已经登录,经过中间件后会将userid存到redis中
	uid := c.GetInt64(middlewares.ContextUserIDKey)
	post := &models.Post{
		CommunityID: p.CommunityID,
		Title:       p.Title,
		Content:     p.Content,
		AuthorID:    uid, // 将uid拼进来
	}
	// 业务
	if err := logic.CreatePost(post); err != nil {
		ResponseWithError(c, err)
		return
	}
	// 返回响应
	pkg.ResponseSuccess(c, nil)
}

func GetPost(c *gin.Context) {
	postid := c.Param("postid")
	pid, err := strconv.ParseInt(postid, 10, 64)
	if err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	data, err := logic.GetPost(pid)
	if err != nil {
		ResponseWithError(c, err)
		return
	}
	pkg.ResponseSuccess(c, data)
}

// post分页展示
func GetPostList(c *gin.Context) {
	p := new(models.ParamPostList)
	if err := c.ShouldBindQuery(p); err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	// 防御：不传或乱传时的兜底
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 || p.Size > 100 { // size 必须封顶，防止 ?size=999999 拖垮DB
		p.Size = 10
	}
	if p.CommunityID < 0 {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	data, err := logic.GetPostList(p)
	if err != nil {
		ResponseWithError(c, err)
		return
	}
	pkg.ResponseSuccess(c, data)
}

// 热榜分页
func GetPostListByHot(c *gin.Context) {
	p := new(models.ParamPostList)
	if err := c.ShouldBindQuery(p); err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	// 防御：不传或乱传时的兜底
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 || p.Size > 100 { // size 必须封顶，防止 ?size=999999 拖垮DB
		p.Size = 10
	}
	data, err := logic.GetPostListByHot(p.Page, p.Size)
	if err != nil {
		ResponseWithError(c, err)
		return
	}
	pkg.ResponseSuccess(c, data)
}

// 帖子投票(点赞)
func PostLikeHandler(c *gin.Context) {
	// 参数校验与获取uid
	p := new(models.PostLikeData)
	if err := c.ShouldBind(p); err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	// 拿到当前用户userid
	uid := c.GetInt64(middlewares.ContextUserIDKey)

	//业务处理
	err := logic.PostLike(p, uid)
	if err != nil {
		ResponseWithError(c, err)
		return
	}

	// 返回响应
	pkg.ResponseSuccess(c, "success")

}

func DeletePost(c *gin.Context) {
	postid := c.Param("postid")
	pid, err := strconv.ParseInt(postid, 10, 64)
	if err != nil {
		pkg.ResponseError(c, pkg.CodeInvalidParam)
		return
	}
	// 删除
	// 拿到当前userid
	uid := c.GetInt64(middlewares.ContextUserIDKey)
	// 删除逻辑
	if err := logic.DeletePost(pid, uid); err != nil {
		ResponseWithError(c, err)
		return
	}
	// 返回响应
	pkg.ResponseSuccess(c, "success")
}
