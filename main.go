package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"web_app/dao/mysql"
	"web_app/dao/redis"
	"web_app/logger"
	snowflask "web_app/pkg/snowflake"
	"web_app/routes"
	"web_app/settings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	// 1:加载配置
	if err := settings.Init(); err != nil {
		fmt.Printf("init settings failed err:%v\n", err)
		return
	}

	// 2:初始化日志
	if err := logger.Init(); err != nil {
		fmt.Printf("init logger failed err:%v\n", err)
		return
	}
	defer zap.L().Sync() // 追加缓冲区日志(将内存中暂存的日志立刻强制写入到磁盘文件或标准输出（终端）中。)
	zap.L().Debug("logger init success...")

	// 3:初始化sql连接
	if err := mysql.Init(); err != nil {
		fmt.Printf("init sql failed err:%v\n", err)
		return
	}
	// 关闭连接池
	defer func() {
		if err := mysql.Close(); err != nil {
			zap.L().Error("mysql close error", zap.Error(err))
		}
	}()

	// 4:初始化redis连接
	if err := redis.Init(); err != nil {
		fmt.Printf("init redis failed err:%v\n", err)
		return
	}
	// 注册 Redis 延迟关闭
	defer func() {
		if err := redis.Close(); err != nil { // 需确保 redis 包内暴露 Close()
			zap.L().Error("redis close error", zap.Error(err))
		}
	}()

	// 雪花生成user_id
	if err := snowflask.Init(viper.GetString("snowflake.starttime"), viper.GetInt64("snowflake.machineid")); err != nil {
		fmt.Printf("init snowflake failed,err:%v\n", err)
	}

	// 5:注册路由
	r := routes.Setup()
	// 6:启动服务(优雅关机)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", viper.GetString("app.port")),
		Handler: r,
	}
	go func() {
		// 开启一个goroutine启动服务
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			zap.L().Fatal("listen:", zap.Error(err))
		}
	}()

	// 等待中断信号来优雅的关闭服务器,设置一个5s的超时
	quit := make(chan os.Signal, 1) // 创建一个接收信号的通道
	//kill 默认发送 syscall.SIGTERM信号
	// kill  -2 发送 syscall.SIGINT 信号,常用的crtl + c 就是触发SIGINT信号
	// kill  -9 发送 syscall.SIGKILL 信号,但是不能被捕获
	// signil.Notify会把收到的信号转发给quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 此处不会堵塞
	<-quit                                               // 在此堵塞,当接收到上述两种信号才会继续向下执行
	zap.L().Info("shoudown server ....")

	//创建一个5s超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 5S内优雅关闭服务,超过5s就超时退出(最先关闭网络请求)
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("shoudown server:", zap.Error(err))
	}
	zap.L().Info("server exiting")
}
