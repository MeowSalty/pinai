package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var Q = query.Q

func Connect(dbType, host, port, user, password, dbname string, logger *slog.Logger) (*sql.DB, error) {
	var db *gorm.DB
	var err error

	gormConfig := &gorm.Config{
		Logger: slogGorm.New(
			slogGorm.WithHandler(logger.Handler()),
		),
	}

	switch dbType {
	case "mysql":
		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			return nil, errors.New("使用 MySQL 数据库需要提供主机、端口、用户名、密码和数据库名")
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, dbname)
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	case "postgres":
		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			return nil, errors.New("使用 PostgreSQL 数据库需要提供主机、端口、用户名、密码和数据库名")
		}
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", host, user, password, dbname, port)
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
	case "sqlite":
		fallthrough
	default:
		db, err = gorm.Open(sqlite.Open("pinai.db"), gormConfig)
	}

	if err != nil {
		return nil, errors.New("无法打开数据库：" + err.Error())
	}
	err = autoMigrate(db)
	if err != nil {
		return nil, errors.New("无法自动迁移数据库：" + err.Error())
	}

	dbConn, err := db.DB()
	if err != nil {
		return nil, errors.New("无法获取数据库连接：" + err.Error())
	}

	query.SetDefault(db)

	return dbConn, nil
}

func autoMigrate(db *gorm.DB) error {
	// Add your model migrations here
	return db.AutoMigrate(
		types.Types...,
	)
}
