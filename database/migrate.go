package database

import (
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

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
	// 迁移 Health 表
	if err := migrateHealthTable(db); err != nil {
		return err
	}
	// 自动迁移表结构
	if err := db.AutoMigrate(
		types.Types...,
	); err != nil {
		return err
	}
	// 迁移旧平台表的 Format 字段到 Provider/Variant
	if err := migratePlatformFormatToProvider(db); err != nil {
		return err
	}
	// 调用数据迁移函数，迁移老数据到新结构
	if err := migrateModelAPIKeysToAssociations(db); err != nil {
		return err
	}
	// 迁移端点配置
	if err := migrateEndpoints(db); err != nil {
		return err
	}
	return nil
}
