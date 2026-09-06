package settings

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func Init() (err error) { // 读取配置
	viper.SetConfigFile("config.yaml")
	viper.AddConfigPath(".") // 使用相对路径

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("viper.ReadInConfig failed , err:%v\n", err)
		return
	}
	// 监控文件变化
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("配置文件修改了")
	})
	return
}
