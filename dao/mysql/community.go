package mysql

import (
	"errors"
	"web_app/models"

	"go.uber.org/zap"
)

var ErrorCommunityNotExist = errors.New("community not exist")

func GetAllCommunity() ([]models.Community, error) {
	var community []models.Community

	err := db.Where("community_id > ?", 0).Find(&community).Error
	if err != nil { // 这里如果查询不到数据会返回[],所以执行到这个分支时一定是数据库连接断开、SQL 语法错误等系统级故障
		zap.L().Error("mysql.GetAllCommunity failed", zap.Error(err))
		return nil, err
	}
	return community, err
}
