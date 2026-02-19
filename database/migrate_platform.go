package database

import (
	"database/sql"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// getDefaultVariant 根据 Provider 获取默认 Variant
func getDefaultVariant(provider string) string {
	switch provider {
	case "OpenAI":
		return "chat_completions"
	case "Anthropic":
		return "messages"
	case "Gemini":
		return "generate"
	default:
		return "default"
	}
}

// migratePlatformFormatToProvider 将旧平台表的 Format 迁移到 Provider/Variant
//
// 仅在旧表仍存在 format 列时执行，且只更新 provider 为空的记录。
//
// 参数：
//   - db: GORM 数据库连接对象
//
// 返回值：
//   - error: 迁移过程中可能发生的错误
func migratePlatformFormatToProvider(db *gorm.DB) error {
	if !db.Migrator().HasTable("platforms") {
		return nil
	}
	if !db.Migrator().HasColumn("platforms", "format") {
		return nil
	}
	if !db.Migrator().HasColumn("platforms", "provider") {
		return nil
	}

	slog.Info("检测到旧平台表 Format 列，开始迁移")

	type oldPlatform struct {
		ID       uint
		Format   sql.NullString
		Provider sql.NullString
	}

	var platforms []oldPlatform
	if err := db.Table("platforms").Select("id, format, provider").Find(&platforms).Error; err != nil {
		return fmt.Errorf("读取平台旧数据失败：%w", err)
	}
	if len(platforms) == 0 {
		slog.Info("未发现平台数据，跳过 Format 迁移")
		return nil
	}

	migratedCount := 0
	for _, platform := range platforms {
		if platform.Provider.Valid && platform.Provider.String != "" {
			continue
		}
		if !platform.Format.Valid || platform.Format.String == "" {
			continue
		}
		variant := getDefaultVariant(platform.Format.String)
		if err := db.Table("platforms").Where("id = ?", platform.ID).Updates(map[string]interface{}{
			"provider": platform.Format.String,
			"variant":  variant,
		}).Error; err != nil {
			return fmt.Errorf("迁移平台 %d 的 Provider 失败：%w", platform.ID, err)
		}
		migratedCount++
	}

	if migratedCount > 0 {
		slog.Info("平台 Format 迁移完成", "migrated_platforms", migratedCount)
	} else {
		slog.Info("无需迁移平台 Format，Provider 均已存在或 Format 为空")
	}

	// 删除旧的 Format 列
	if err := db.Migrator().DropColumn("platforms", "format"); err != nil {
		return fmt.Errorf("删除平台表 Format 列失败：%w", err)
	}
	slog.Info("已删除平台表 Format 列")

	return nil
}
