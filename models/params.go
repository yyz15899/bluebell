package models

type ParamSignUp struct {
	Username   string `json:"username" binding:"required"` // "" 空字符串也不行
	Password   string `json:"password" binding:"required"`
	RePassword string `json:"re_password" binding:"required,eqfield=Password"`
}
