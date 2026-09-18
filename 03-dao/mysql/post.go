package mysql

import (
	"errors"
	"fmt"
	models "web_app/05-models"

	"gorm.io/gorm"
)

func CreatePost(post *models.Post) error {
	if err := db.Create(post).Error; err != nil {
		return err
	}

	return nil
}

var ErrPostNotExist = errors.New("post not existing")

func GetPost(pid int64) (*models.Post, error) {
	post := new(models.Post)
	if err := db.Where("post_id = ? AND status = ?", pid, 1).First(post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // gorm.ErrRecordNotFound 表示找不到数据报err
			return nil, ErrPostNotExist
		}
		return nil, fmt.Errorf("query post %d: %w", pid, err)
	}
	return post, nil
}

// GetUserByID 根据user_id(雪花ID)查询用户信息
func GetUserByID(userID int64) (*models.User, error) {
	u := new(models.User)
	if err := db.Where("user_id = ?", userID).First(u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // 没查到该用户
			return nil, ErrUserNotExist
		}
		return nil, fmt.Errorf("query user %d: %w", u.UserID, err)
	}
	return u, nil
}

func GetUserByIDs(ids []int64) ([]models.User, error) {
	var users []models.User
	if err := db.Where("user_id IN ?", ids).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("sql link warn%w", err)
	}
	return users, nil
}

// 按照社区id分页
func GetPostList(p *models.ParamPostList) ([]*models.Post, error) {
	var post []*models.Post
	tx := db.Model(&models.Post{}).
		Select([]string{"id", "post_id", "title", "author_id", "community_id", "vote_num", "status", "create_time"}).
		Where("status = ?", 1)
	if p.CommunityID > 0 {
		tx = tx.Where("community_id = ?", p.CommunityID)
	}
	err := tx.Order("id DESC").Limit(int(p.Size)).Offset(int(p.Page-1) * int(p.Size)).Find(&post).Error
	if err != nil {
		return nil, fmt.Errorf("query post list failed: %w", err)
	}
	return post, nil
}

// 根据有序(热榜)地pid批量查详情
func GetPostListByIDs(ids []int64) ([]*models.Post, error) {
	var post []*models.Post
	if err := db.Where("post_id IN ? AND status = ?", ids, 1).Find(&post).Error; err != nil {
		return nil, fmt.Errorf("sql link warn%w", err)
	}
	return post, nil
}

func UpdatePostVoteNums(votes map[int64]int64) error {
	// 内部用一个事务覆盖业务,防止因外部原因导致数据没写入
	return db.Transaction(func(tx *gorm.DB) error {
		for pid, num := range votes {
			if err := tx.Model(&models.Post{}). // model接收的数据只用来指示查询哪张表,所以要传实例而不是类型(&models.Post)
								Where("post_id = ? AND status = ?", pid, 1).
								Update("vote_num", num).Error; err != nil {
				return err
			}
		}
		return nil // 提交事务
	})
}

func DeletePost(postid int64) error {
	res := db.Model(&models.Post{}).Where("post_id = ?", postid).Update("status", 0) // 以status=0 作为post已删除的标记
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPostNotExist
	}
	return nil
}
