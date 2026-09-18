package models

import "time"

type Community struct {
	// 显式指定数据库列名（推荐），避免默认命名约定带来的隐患
	CommunityID   int    `gorm:"column:community_id" json:"community_id"`
	CommunityName string `gorm:"column:community_name" json:"community_name"`
}

type CommunityDetail struct {
	// 显式指定数据库列名（推荐），避免默认命名约定带来的隐患
	Community
	Introduction string    `gorm:"column:introduction" json:"introduction"`
	CreateTime   time.Time `gorm:"column:create_time" json:"create_time"`
}

// TableName 指定表名，防止 GORM 自动复数化变成 "users"
// 若没有这个
// 建的是 community（单数）。所以 GetAllCommunity / GetCommunity 实际去查 communities 表 → 报 “table doesn't exist” → 两个接口都返回“服务繁忙”。
func (c *Community) TableName() string {
	return "community" // 映射数据库中的 community 表
}
func (c *CommunityDetail) TableName() string {
	return "community"
}
