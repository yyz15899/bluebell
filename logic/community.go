package logic

import (
	"web_app/dao/mysql"
	"web_app/models"

	"go.uber.org/zap"
)

func GetCommunityList() ([]models.Community, error) {
	list, err := mysql.GetAllCommunity()
	if err != nil {
		zap.L().Error("GetCommunityList failed in GetAllCommunity", zap.Error(err))
		return nil, err
	}
	return list, nil
}
