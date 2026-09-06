package logic

import (
	"errors"
	"web_app/dao/mysql"
	"web_app/models"
	"web_app/pkg/snowflake"

	"go.uber.org/zap"
)

// 存放业务逻辑的代码

func SignUp(p *models.ParamSignUp) error { // 接收一个指针(结构体)变量
	// 判断用户(name)是否存在
	exist, err := mysql.CheckUserExist(p.Username)
	if err != nil {
		// DB 查询报错，记录逻辑层日志信息并直接返回
		zap.L().Error("SignUp logic failed on CheckUserExist", zap.String("username", p.Username), zap.Error(err)) //数据库本体报错
		return errors.New("服务繁忙，请稍后再试")
	}
	if exist { // if 返回true
		zap.L().Warn("SignUp logic blocked: user already exists", zap.String("username", p.Username))
		return errors.New("用户已存在")
	}
	// 生成UID
	userID := snowflake.GenID()
	// 构造user实例
	user := models.User{
		UserID:   userID,
		Username: p.Username,
		Password: p.Password,
	}
	// 保存进mysql
	return mysql.InsertUser(&user)

}
