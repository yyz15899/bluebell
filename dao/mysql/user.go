package mysql

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"web_app/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CheckUserExist 查重username
func CheckUserExist(username string) (bool, error) {
	var count int64
	err := db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil { // 数据库自身问题(连接失败,sql语句错误...)
		// 记录日志
		zap.L().Error("mysql.CheckUserExist failed ...", zap.String("username", username), zap.Error(err))
		return false, err // 数据库查询异常
	}
	// 当数据库正常查询到了结构
	return count > 0, nil // 将err的位置设为nil 告诉logic层没有错误
	// 如果数据库内没有重复的 return false nil
	// 如果数据库内有重复的 return true nil
}

// InsertUser 向数据库中插入一条新的用户数据
func InsertUser(user *models.User) error {
	// 对密码进行加密
	user.Password = encryptPassword(user.Password)
	// 执行sql语句入库
	// result := db.Create(user)
	// err := result.Error
	err := db.Create(user).Error // 传入指针(*models.User)来在数据创建数据
	if err != nil {
		// 记录日志(带上userid username 方便排查)
		zap.L().Error("mysql insert failed",
			zap.Int64("user_id", user.UserID),
			zap.String("username", user.Username),
			zap.Error(err))
		return err
	}
	return nil
}

// encryptPassword 使用md5加密密码
const secret = "hello world"

func encryptPassword(opassword string) string {
	h := md5.New()
	h.Write([]byte(secret))
	return hex.EncodeToString(h.Sum([]byte(opassword)))
}

// GetUser 按照username取出用户信息 包括密码密文

var ErrorUserNotExist = errors.New("user not exist")

func GetUser(username string) (error, *models.User) { // 将数据库中该用户注册时的信息返回
	u := new(models.User)
	if err := db.Where("username = ?", username).First(u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // gorm.ErrRecordNotFound 表示找不到数据报err
			return ErrorUserNotExist, nil
		}
		zap.L().Error("mysql getuser failed", zap.String("username", username), zap.Error(err))
		return err, nil
	}
	return nil, u
}

// 判断密码是否正确(都是加密后的)
func ComparePassword(u *models.User, password string) bool {
	return u.Password == encryptPassword(password)
}
