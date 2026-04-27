package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/MeowSalty/pinai/database/query"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const sqliteBusyTimeoutMS = 5000

const sqliteJournalModeWAL = "WAL"

// Connect 连接到数据库
//
// 该函数根据提供的数据库类型和连接信息连接到数据库，并配置 slog-gorm 日志记录器。
//
// 参数：
//   - dbType: 数据库类型 ("sqlite", "mysql", "postgres")
//   - host: 数据库主机地址
//   - port: 数据库端口
//   - user: 数据库用户名
//   - password: 数据库密码
//   - dbname: 数据库名称
//   - sslMode: PostgreSQL SSL 模式 (disable, require, verify-ca, verify-full)
//   - tlsConfig: MySQL TLS 配置 (true, false, skip-verify, preferred)
//   - logger: 用于数据库操作的日志记录器
//
// 返回值：
//   - *sql.DB: 数据库连接对象
//   - error: 连接过程中可能发生的错误
func Connect(dbType, host, port, user, password, dbname, sslMode, tlsConfig string, logger *slog.Logger) (*sql.DB, error) {
	var db *gorm.DB
	var err error
	isSQLite := false

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
		// 添加 TLS 配置
		if tlsConfig != "" {
			dsn += fmt.Sprintf("&tls=%s", tlsConfig)
		}
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	case "postgres":
		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			return nil, errors.New("使用 PostgreSQL 数据库需要提供主机、端口、用户名、密码和数据库名")
		}
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s", host, user, password, dbname, port)
		// 添加 SSL 模式配置
		if sslMode != "" {
			dsn += fmt.Sprintf(" sslmode=%s", sslMode)
		}
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
	case "sqlite":
		fallthrough
	default:
		isSQLite = true
		sqliteDSN := fmt.Sprintf("data/pinai.db?_busy_timeout=%d", sqliteBusyTimeoutMS)
		db, err = gorm.Open(sqlite.Open(sqliteDSN), gormConfig)
	}

	if err != nil {
		return nil, errors.New("无法打开数据库：" + err.Error())
	}

	if isSQLite {
		if err := enableSQLiteWAL(db, logger); err != nil {
			return nil, fmt.Errorf("无法启用 SQLite WAL 模式：%w", err)
		}
	}

	err = autoMigrate(db)
	if err != nil {
		return nil, errors.New("无法自动迁移数据库：" + err.Error())
	}

	// 清理健康表中的孤立记录
	if err := cleanOrphanedHealthRecords(db); err != nil {
		return nil, errors.New("无法清理健康表孤立记录：" + err.Error())
	}

	dbConn, err := db.DB()
	if err != nil {
		return nil, errors.New("无法获取数据库连接：" + err.Error())
	}

	query.SetDefault(db)

	return dbConn, nil
}

func enableSQLiteWAL(db *gorm.DB, logger *slog.Logger) error {
	if db == nil {
		return errors.New("数据库连接为空")
	}

	var journalMode string
	if err := db.Raw("PRAGMA journal_mode=WAL;").Scan(&journalMode).Error; err != nil {
		return fmt.Errorf("执行 PRAGMA journal_mode=WAL 失败：%w", err)
	}

	if !strings.EqualFold(journalMode, sqliteJournalModeWAL) {
		return fmt.Errorf("SQLite journal_mode=%s，未成功切换为 WAL", journalMode)
	}

	if logger != nil {
		logger.Debug("SQLite WAL 模式已启用")
	}

	return nil
}
