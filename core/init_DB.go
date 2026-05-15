// core/init_db.go
package core

import (
	"AutoOps/global"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitDB 初始化数据库连接
// 参数: 无
// 返回: *gorm.DB - 数据库连接实例
// 说明: 连接主库,配置连接池,支持读写分离
func InitDB() *gorm.DB {
	// 获取数据库配置
	dc := global.Config.DB //写库

	// 连接主数据库
	db, err := gorm.Open(dc.DSN(), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 不生成外键约束
	})
	if err != nil {
		logrus.Fatalf("数据库连接失败 %s", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期
	logrus.Println("数据库连接成功")

	if global.Config.RunMode == "develop" {
		db = db.Debug()
		logrus.Println("数据库调试模式已开启")
		return db
	}
	return db
}

func InitAiDB() *gorm.DB {
	// 获取AI数据库配置
	dc := global.Config.AiDB // AI数据库配置

	// 连接AI数据库
	db, err := gorm.Open(dc.DSN(), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 不生成外键约束
	})
	if err != nil {
		logrus.Fatalf("AI数据库连接失败 %s", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	sqlDB.SetMaxIdleConns(10)           // 最大空闲连接数
	sqlDB.SetMaxOpenConns(100)          // 最大打开连接数
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最大生命周期
	logrus.Println("AI数据库连接成功")

	if global.Config.RunMode == "develop" {
		db = db.Debug()
		logrus.Println("AI数据库调试模式已开启")
		return db
	}
	return db
}
