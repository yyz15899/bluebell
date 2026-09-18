package redis

import (
	"context"
	"fmt"
	"time"
)

const loginKeyPrefix = "bluebell:login:"

// 记录登陆时的token
func SaveLoginToken(userid int64, Token string, expire time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", loginKeyPrefix, userid)
	return RDB.Set(ctx, key, Token, expire).Err()
}

// 退出登录时,删除token记录(在登出时使用)
func DelToken(userid int64) error {
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", loginKeyPrefix, userid)
	return RDB.Del(ctx, key).Err()
}

// 返回当前有效的token
func GetLoginToken(userid int64) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", loginKeyPrefix, userid)
	return RDB.Get(ctx, key).Result()
}
