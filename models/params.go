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
