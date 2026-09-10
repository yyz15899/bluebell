package models

type Community struct {
	// 显式指定数据库列名（推荐），避免默认命名约定带来的隐患
	Community_id   int    `gorm:"column:community_id" json:"community_id"`
	Community_name string `gorm:"column:community_name" json:"community_name"`
}
