package logic

import (
	"errors"
	"web_app/03-dao/mysql"
	"web_app/03-dao/redis"
	models "web_app/05-models"
	pkg "web_app/07-pkg"
	"web_app/07-pkg/snowflake"

	"go.uber.org/zap"
)

// 存放业务逻辑的代码

func SignUp(p *models.ParamSignUp) error { // 接收一个指针(结构体)变量
	// 判断用户(name)是否存在
	exist, err := mysql.CheckUserExist(p.Username)
	if err != nil {
		return pkg.WrapBizError(pkg.CodeServerBusy, err)
	}
	if exist { // if 返回true
		// 用户名占用:预期内的业务结果,非故障;保留 username 便于后续做枚举探测风控
		zap.L().Warn("signup: username already taken", zap.String("username", p.Username))
		return pkg.NewBizError(pkg.CodeUserExist)
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

func SignIn(p *models.ParamSignIn) (int64, error) {
	// 查看用户是否存在(直接取出用户)
	info, err := mysql.GetUser(p.Username) // 这里的info 是存储get过来的用户信息的结构体指针
	if err != nil {
		if errors.Is(err, mysql.ErrUserNotExist) {
			// 只有区分进日志(内部排查/防爆破监控)因为此时不能让用户知道,但需要被开发人员知道,响应统一话术
			zap.L().Warn("signin: user not exist", zap.String("username", p.Username))
			return 0, pkg.NewBizError(pkg.CodeInvalidPassword)
		} else {
			return 0, pkg.WrapBizError(pkg.CodeServerBusy, err)
		}
		// return 0, errors.New("用户名或密码错误") // 统一文案, 不泄露账号是否存在
	}
	// 密码是否正确
	if !mysql.ComparePassword(info, p.Password) {
		zap.L().Warn("signin: password mismatch", zap.String("username", p.Username))
		return 0, pkg.NewBizError(pkg.CodeInvalidPassword)
	}
	return info.UserID, nil
}

func Signout(uid int64) error {
	// 执行jwt中间件 执行到这里能判断已经登陆 不用判断
	// 删除jwt
	err := redis.DelToken(uid)
	if err != nil {
		zap.L().Error("redis deltoken failed")
		// redis操作错误 不return err
	}
	return nil
}

// 更新信息
func UpdateInfo(uid int64, p *models.ParamUpdate) error {
	if err := mysql.UpdatePassword(uid, p.OldPassword, p.NewPassword); err != nil {
		switch {
		case errors.Is(err, mysql.ErrPasswordMismatch):
			zap.L().Warn("update password: old password mismatch", zap.Int64("user_id", uid))
			return pkg.NewBizError(pkg.CodeInvalidPassword)
		case errors.Is(err, mysql.ErrUserNotExist):
			return pkg.NewBizError(pkg.CodeUserNotExist)
		default:
			zap.L().Error("update password failed", zap.Error(err), zap.Int64("user_id", uid))
			return pkg.WrapBizError(pkg.CodeServerBusy, err)
		}
	}
	return nil
}
