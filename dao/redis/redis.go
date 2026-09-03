package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// 暴露rdb到外部,充当句柄
var RDB *redis.Client

func Init() (err error) {
	ctx := context.Background()
	RDB = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", viper.GetString("redis.host"), viper.GetInt("redis.port")), // 从配置文件读取 "127.0.0.1:6379"
		Password:     viper.GetString("redis.password"),                                               // 密码
		DB:           viper.GetInt("redis.db"),                                                        // 数据库 index
		PoolSize:     viper.GetInt("redis.pool_size"),                                                 // 连接池大小（默认一般为 10 * runtime.GOMAXPROCS）
		MinIdleConns: viper.GetInt("redis.min_idle"),                                                  // 最小空闲连接数
	})

	_, err = RDB.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("connect redis failed: %w", err)
	}
	return nil
}

func Close() error {
	return RDB.Close()
}
