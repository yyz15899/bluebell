package mysql

import (
	"errors"
	"fmt"
	models "web_app/05-models"

	"gorm.io/gorm"
)

var ErrCommunityNotExist = errors.New("community not exist")

func GetAllCommunity() ([]models.Community, error) {
	var community []models.Community

	err := db.Find(&community).Error
	if err != nil { // 这里如果查询不到数据会返回[],所以执行到这个分支时一定是数据库连接断开、SQL 语法错误等系统级故障
		return nil, fmt.Errorf("sql link warn%w", err)
	}
	return community, err
}

// 根据community_id 查询单个
func GetCommunity(id int64) (*models.CommunityDetail, error) {
	var community models.CommunityDetail
	err := db.Where("community_id = ?", id).First(&community).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommunityNotExist
		} else {
			return nil, fmt.Errorf("sql link warn%w", err)
		}
	}
	return &community, nil
}

// 根据community_id 查询多个
func GetCommunityByid(id []int64) ([]models.Community, error) {
	var community []models.Community
	if err := db.Where("community_id IN ?", id).Find(&community).Error; err != nil {
		return nil, fmt.Errorf("sql link warn%w", err)
	}
	return community, nil
}

// 检查community是否存在
func CheckCommunityID(cid int64) (bool, error) {
	var count int64
	err := db.Model(&models.Community{}).Where("community_id = ?", cid).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("count community %d: %w", cid, err)
	}
	return count > 0, nil // 合理的cid 则返回true
}
