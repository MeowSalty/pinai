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
		db, err = gorm.Open(sqlite.Open("data/pinai.db"), gormConfig)
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

// autoMigrate 自动迁移数据库表结构
//
// 该函数负责自动创建或更新数据库表结构以匹配当前的数据模型。
//
// 参数：
//   - db: GORM 数据库连接对象
//
// 返回值：
//   - error: 迁移过程中可能发生的错误
func autoMigrate(db *gorm.DB) error {
	// 自动迁移表结构
	if err := db.AutoMigrate(
		types.Types...,
	); err != nil {
		return err
	}
	// 调用数据迁移函数，迁移老数据到新结构
	if err := migrateOldData(db); err != nil {
		return err
	}
	return nil
}

// migrateOldData 对旧数据进行结构迁移，兼容模型与密钥的新多对多关联
//
// 该函数负责将旧数据结构迁移到新的多对多关联结构。
// 如果旧数据中模型和密钥之间存在隐式关联（例如通过 platform_id），
// 此函数会自动建立多对多关联关系。
//
// 参数：
//   - db: GORM 数据库连接对象
//
// 返回值：
//   - error: 迁移过程中可能发生的错误
func migrateOldData(db *gorm.DB) error {
	slog.Info("开始检查并迁移旧数据至新模型 - 密钥多对多关联")

	// 检查是否需要迁移：查询所有模型
	var models []types.Model
	if err := db.Find(&models).Error; err != nil {
		return fmt.Errorf("查询模型数据失败：%w", err)
	}

	// 如果没有模型数据，跳过迁移
	if len(models) == 0 {
		slog.Info("未发现模型数据，跳过数据迁移")
		return nil
	}

	// 为每个模型建立与同平台密钥的多对多关联
	// 这是一个示例逻辑：假设同一平台下的模型应该关联该平台的所有密钥
	migratedCount := 0
	for _, model := range models {
		// 检查该模型是否已有关联的密钥
		var existingKeys []types.APIKey
		if err := db.Model(&model).Association("APIKeys").Find(&existingKeys); err != nil {
			return fmt.Errorf("查询模型 %d 的关联密钥失败：%w", model.ID, err)
		}

		// 如果已有关联，跳过该模型
		if len(existingKeys) > 0 {
			continue
		}

		// 查询同平台的所有密钥
		var platformKeys []types.APIKey
		if err := db.Where("platform_id = ?", model.PlatformID).Find(&platformKeys).Error; err != nil {
			return fmt.Errorf("查询平台 %d 的密钥失败：%w", model.PlatformID, err)
		}

		// 如果该平台有密钥，建立关联
		if len(platformKeys) > 0 {
			if err := db.Model(&model).Association("APIKeys").Append(platformKeys); err != nil {
				return fmt.Errorf("为模型 %d 建立密钥关联失败：%w", model.ID, err)
			}
			migratedCount++
			slog.Debug("已为模型建立密钥关联",
				"model_id", model.ID,
				"model_name", model.Name,
				"platform_id", model.PlatformID,
				"key_count", len(platformKeys),
			)
		}
	}

	if migratedCount > 0 {
		slog.Info("旧数据迁移完成", "migrated_models", migratedCount)
	} else {
		slog.Info("无需迁移旧数据，所有模型已有密钥关联或平台无密钥")
	}
	return nil
}
