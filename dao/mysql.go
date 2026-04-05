package dao

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"time"
)

var (
	DB *gorm.DB
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// InitMySQL 连接数据库，支持通过环境变量覆盖配置
func InitMySQL() {
	username := getEnv("DB_USER", "root")
	pwd := getEnv("DB_PASSWORD", "root")
	db := getEnv("DB_NAME", "person_practice")
	ip := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=True&loc=Local",
		username, pwd, ip, port, db)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("连接数据库失败: %v", err))
	}

	// 配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		panic(fmt.Sprintf("获取数据库实例失败: %v", err))
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
}
