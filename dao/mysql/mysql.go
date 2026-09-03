package mysql

// gorm
import (
	"fmt"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 全局 DB 变量（通常脚手架需要暴露 db 句柄供其他包调用）
var db *gorm.DB

func Init() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		viper.GetString("mysql.name"),
		viper.GetString("mysql.password"),
		viper.GetString("mysql.host"),
		viper.GetInt("mysql.port"),
		viper.GetString("mysql.dbname"))

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})

	// ctx := context.Background()
	if err != nil {
		fmt.Printf("connect DB failed,err:%v\n", err)
		return nil
	}

	// 配置连接池
	sqlDB, err := db.DB() //sqlDB是原生数据库连接池对象,负责底层的数据库连接管理（TCP 握手、连接复用、连接超时、最大连接数控制等）。
	if err != nil {
		return fmt.Errorf("get sqlDB failed: %w", err)
	}

	// 读取配置，并加上兜底默认值（如果配置文件未填写，使用合理默认值）
	maxIdle := viper.GetInt("mysql.maxIdleConns")
	if maxIdle == 0 {
		maxIdle = 10 // 默认值 10
	}

	maxOpen := viper.GetInt("mysql.maxOpenConns")
	if maxOpen == 0 {
		maxOpen = 100 // 默认值 100
	}
	// 配置数据
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)

	// sqlDB.SetConnMaxLifetime(time.Hour)	// 建议开启：防止底层连接死锁或网关断开连接（如 MySQL 默认 8小时超时）
	return nil
}

// [ 你的业务代码 ]
//        │
//        ▼ (使用 DB 进行增删改查)
// ┌──────────────┐
// │  GORM (*gorm.DB)   │ ── 负责：对象映射 (ORM)、拼装 SQL
// └──────┬───────┘
//        │ 内部持有 / 通过 DB.DB() 提取
//        ▼
// ┌──────────────┐
// │  database/sql│ ── 负责：连接池管理 (*sql.DB)、TCP 连接生命周期
// └──────┬───────┘
//        │
//        ▼
//   [ MySQL 数据库 ]

// 因为注册的是小写db,不能对外暴露,考虑封装一个close函数
// func GetDB() *gorm.DB {
// 	return db // GetDB 对外提供获取 DB 实例的方法
// }

func Close() error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sqlDB failed during close: %w", err)
	}
	return sqlDB.Close()
}
