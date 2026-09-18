package models

import "time"

// 查询post
type Post struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PostID      int64     `gorm:"column:post_id" json:"post_id"`
	AuthorID    int64     `gorm:"column:author_id" json:"author_id"`
	CommunityID int64     `gorm:"column:community_id" json:"community_id"`
	VoteNum     int64     `gorm:"column:vote_num;default:0" json:"vote_num"`
	Status      int32     `gorm:"column:status;default:1" json:"status"` // 内存对齐
	Title       string    `gorm:"column:title" json:"title"`
	Content     string    `gorm:"column:content" json:"content"`
	CreateTime  time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
}

// 创建post
type ParamCreatePost struct {
	CommunityID int64  `gorm:"column:community_id" json:"community_id" binding:"required"`
	Title       string `gorm:"column:title" json:"title" binding:"required"`
	Content     string `gorm:"column:content" json:"content" binding:"required"`
}

// 分页查询
type ParamPostList struct {
	CommunityID int64 `form:"community_id"` // 0 表示查全部，>0 表示查某个社区
	Page        int64 `form:"page"`
	Size        int64 `form:"size"`
}

type ApiPostDetail struct {
	AuthorName string `json:"author_name"`
	*Post             // 嵌入帖子结构体
	// Community        // 社区信息
	// 运行发现community_id消失,因为两个嵌入结构体的同名 tag 被 Go 判定冲突后整体剔除，编译和运行都不报错。建议保持扁平结构（
	CommunityName string `json:"community_name"`
}

type PostLikeData struct {
	PostID    int64 `json:"post_id" binding:"required"`
	Direction int64 `json:"direction" binding:"required,oneof=1 -1"` // onenf 限制值只能是1 -1
}

const (
	PostLike   = 1
	PostUnLike = -1
)

func (p *Post) TableName() string {
	return "post"
}
