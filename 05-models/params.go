package models

// 注册请求参数
type ParamSignUp struct {
	Username   string `json:"username" binding:"required"` // "" 空字符串也不行
	Password   string `json:"password" binding:"required"`
	RePassword string `json:"re_password" binding:"required,eqfield=Password"`
}

// 登录请求参数
type ParamSignIn struct {
	Username string `json:"username" binding:"required"` // "" 空字符串也不行
	Password string `json:"password" binding:"required"`
}

// 修改信息
type ParamUpdate struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// 修改post
type ParamUpdatePost struct {
	Title       *string `gorm:"column:title" json:"title"`
	Content     *string `gorm:"column:content" json:"content"`
	CommunityID *int64  `gorm:"column:community_id" json:"community_id"`
}
