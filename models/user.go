package models

import "time"

type User struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"column:user_id;uniqueIndex:idx_user_id;not null" json:"user_id"`
	Username   string    `gorm:"column:username;uniqueIndex:idx_username;not null" json:"username"`
	Password   string    `gorm:"column:password;not null" json:"-"` // 用 "-" 避免密码在 JSON 序列化时泄露
	Email      string    `gorm:"column:email" json:"email"`
	Gender     int8      `gorm:"column:gender;default:0;not null" json:"gender"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

//  TableName 指定表名，防止 GORM 自动复数化变成 "users"
func (u *User) TableName() string {
	return "user"
}
