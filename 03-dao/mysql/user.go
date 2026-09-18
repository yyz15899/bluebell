package mysql

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	models "web_app/05-models"

	"gorm.io/gorm"
)

var ErrUserExist = errors.New("user already exists")
var ErrUserNotExist = errors.New("user not exist")

// CheckUserExist 查重username
func CheckUserExist(username string) (bool, error) {
	var count int64
	err := db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil { // 数据库自身问题(连接失败,sql语句错误...)
		return false, fmt.Errorf("sql link warn%w", err) // 数据库查询异常
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

func GetUser(username string) (*models.User, error) { // 将数据库中该用户注册时的信息返回
	u := new(models.User)
	if err := db.Where("username = ?", username).First(u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // gorm.ErrRecordNotFound 表示找不到数据报err
			return nil, ErrUserNotExist
		}
		return nil, fmt.Errorf("sql link warn%w", err)
	}
	return u, nil
}

// 判断密码是否正确(都是加密后的)
func ComparePassword(u *models.User, password string) bool {
	return u.Password == encryptPassword(password)
}

// 更新账号信息
var ErrPasswordMismatch = errors.New("密码错误")

// 校验旧密码后更新密码,userid来自jwt
func UpdatePassword(userID int64, oldPassword, newPassword string) error {
	user := new(models.User)
	if err := db.Where("user_id = ?", userID).First(user).Error; err != nil { // 将userid写入models.User
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotExist
		}
		return fmt.Errorf("mysql link failed,%w", err)
	}
	if !ComparePassword(user, oldPassword) {
		return ErrPasswordMismatch
	}
	if err := db.Model(&models.User{}).Where("user_id = ?", userID).Update("password", encryptPassword(newPassword)).Error; err != nil {
		return fmt.Errorf("update password for user %d: %w", userID, err)
	}
	return nil

}
