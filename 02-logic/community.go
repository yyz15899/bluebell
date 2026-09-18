package logic

import (
	"errors"
	"web_app/03-dao/mysql"
	models "web_app/05-models"
	pkg "web_app/07-pkg"

	"go.uber.org/zap"
)

func GetCommunityList() ([]models.Community, error) {
	list, err := mysql.GetAllCommunity()
	if err != nil {
		zap.L().Error("get communitylist failed", zap.Error(err))
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	return list, nil
}

func GetCommunity(id int64) (*models.CommunityDetail, error) {
	info, err := mysql.GetCommunity(id)
	if err != nil {
		if errors.Is(err, mysql.ErrCommunityNotExist) { // 帖子存在属于正常现象,为了防止数据库真报错时被这种日志刷屏,这里不打日志
			return nil, pkg.NewBizError(pkg.CodeCommunityNotExist)
		}
		zap.L().Error("get community failed", zap.Error(err), zap.Int64("community_id", id))
		return nil, pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	return info, nil
}
