package database

import (
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// migrateModelAPIKeysToAssociations 对旧数据进行结构迁移，兼容模型与密钥的新多对多关联
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
func migrateModelAPIKeysToAssociations(db *gorm.DB) error {
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
