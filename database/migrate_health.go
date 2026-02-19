package database

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// migrateHealthTable 迁移 Health 表从旧结构到新的联合主键结构
//
// 检测旧表是否存在 id 列，如果存在则：
// 1. 备份数据到内存
// 2. 去重（保留每个 ResourceType+ResourceID 组合中 LastCheckAt 最新的记录）
// 3. 删除旧表
// 4. 让 AutoMigrate 创建新表后恢复数据
func migrateHealthTable(db *gorm.DB) error {
	// 检查表是否存在
	if !db.Migrator().HasTable("healths") {
		return nil // 表不存在，跳过
	}

	// 检查是否存在 id 列（旧结构标志）
	if !db.Migrator().HasColumn(&types.Health{}, "id") {
		return nil // 已是新结构，跳过
	}

	slog.Info("检测到旧版 Health 表结构，开始迁移")

	// 定义旧表结构用于读取数据
	type oldHealth struct {
		ID              uint
		ResourceType    types.ResourceType
		ResourceID      uint
		Status          types.HealthStatus
		RetryCount      int
		NextAvailableAt *time.Time
		BackoffDuration int64
		LastError       string
		LastErrorCode   int
		LastCheckAt     time.Time
		LastSuccessAt   *time.Time
		SuccessCount    int
		ErrorCount      int
		CreatedAt       time.Time
		UpdatedAt       time.Time
	}

	// 1. 备份数据
	var oldRecords []oldHealth
	if err := db.Table("healths").Order("resource_type, resource_id, last_check_at DESC").Find(&oldRecords).Error; err != nil {
		return fmt.Errorf("备份 Health 数据失败：%w", err)
	}

	// 2. 去重 - 保留每个组合中最新的记录
	seen := make(map[string]bool)
	var uniqueRecords []types.Health
	for _, record := range oldRecords {
		key := fmt.Sprintf("%d-%d", record.ResourceType, record.ResourceID)
		if !seen[key] {
			seen[key] = true
			uniqueRecords = append(uniqueRecords, types.Health{
				ResourceType:    record.ResourceType,
				ResourceID:      record.ResourceID,
				Status:          record.Status,
				RetryCount:      record.RetryCount,
				NextAvailableAt: record.NextAvailableAt,
				BackoffDuration: record.BackoffDuration,
				LastError:       record.LastError,
				LastErrorCode:   record.LastErrorCode,
				LastCheckAt:     record.LastCheckAt,
				LastSuccessAt:   record.LastSuccessAt,
				SuccessCount:    record.SuccessCount,
				ErrorCount:      record.ErrorCount,
				CreatedAt:       record.CreatedAt,
				UpdatedAt:       record.UpdatedAt,
			})
		}
	}

	slog.Info("Health 数据备份完成",
		"total_records", len(oldRecords),
		"unique_records", len(uniqueRecords),
		"duplicates_removed", len(oldRecords)-len(uniqueRecords))

	// 3. 删除旧表
	if err := db.Migrator().DropTable("healths"); err != nil {
		return fmt.Errorf("删除旧 Health 表失败：%w", err)
	}

	// 4. 创建新表（由 AutoMigrate 完成）并恢复数据
	// 先创建新表
	if err := db.AutoMigrate(&types.Health{}); err != nil {
		return fmt.Errorf("创建新 Health 表失败：%w", err)
	}

	// 恢复数据
	if len(uniqueRecords) > 0 {
		if err := db.Create(&uniqueRecords).Error; err != nil {
			return fmt.Errorf("恢复 Health 数据失败：%w", err)
		}
	}

	slog.Info("Health 表迁移完成", "restored_records", len(uniqueRecords))
	return nil
}

// cleanOrphanedHealthRecords 清理健康表中不存在的资源
//
// 该函数检查健康表中的每个条目，验证其对应的资源是否仍然存在。
// 如果资源已被删除，则移除健康表中的孤立条目。
//
// 参数：
//   - db: GORM 数据库连接对象
//
// 返回值：
//   - error: 清理过程中可能发生的错误
func cleanOrphanedHealthRecords(db *gorm.DB) error {
	slog.Info("开始清理健康表中的孤立记录")

	// 检查健康表是否存在
	if !db.Migrator().HasTable("healths") {
		slog.Info("健康表不存在，跳过清理")
		return nil
	}

	// 获取所有健康记录
	var healthRecords []types.Health
	if err := db.Find(&healthRecords).Error; err != nil {
		return fmt.Errorf("查询健康表失败：%w", err)
	}

	if len(healthRecords) == 0 {
		slog.Info("健康表为空，无需清理")
		return nil
	}

	// 按资源类型分组
	platformIDs := make(map[uint]bool)
	apiKeyIDs := make(map[uint]bool)
	modelIDs := make(map[uint]bool)

	for _, record := range healthRecords {
		switch record.ResourceType {
		case types.ResourceTypePlatform:
			platformIDs[record.ResourceID] = true
		case types.ResourceTypeAPIKey:
			apiKeyIDs[record.ResourceID] = true
		case types.ResourceTypeModel:
			modelIDs[record.ResourceID] = true
		}
	}

	// 统计需要删除的记录
	var toDelete []types.Health
	deletedCount := 0

	// 检查平台资源
	if len(platformIDs) > 0 {
		var existingPlatforms []types.Platform
		if err := db.Select("id").Find(&existingPlatforms).Error; err != nil {
			return fmt.Errorf("查询平台表失败：%w", err)
		}

		existingMap := make(map[uint]bool)
		for _, platform := range existingPlatforms {
			existingMap[platform.ID] = true
		}

		for id := range platformIDs {
			if !existingMap[id] {
				toDelete = append(toDelete, types.Health{
					ResourceType: types.ResourceTypePlatform,
					ResourceID:   id,
				})
				deletedCount++
				slog.Debug("发现孤立的平台健康记录", "platform_id", id)
			}
		}
	}

	// 检查密钥资源
	if len(apiKeyIDs) > 0 {
		var existingAPIKeys []types.APIKey
		if err := db.Select("id").Find(&existingAPIKeys).Error; err != nil {
			return fmt.Errorf("查询密钥表失败：%w", err)
		}

		existingMap := make(map[uint]bool)
		for _, apiKey := range existingAPIKeys {
			existingMap[apiKey.ID] = true
		}

		for id := range apiKeyIDs {
			if !existingMap[id] {
				toDelete = append(toDelete, types.Health{
					ResourceType: types.ResourceTypeAPIKey,
					ResourceID:   id,
				})
				deletedCount++
				slog.Debug("发现孤立的密钥健康记录", "api_key_id", id)
			}
		}
	}

	// 检查模型资源
	if len(modelIDs) > 0 {
		var existingModels []types.Model
		if err := db.Select("id").Find(&existingModels).Error; err != nil {
			return fmt.Errorf("查询模型表失败：%w", err)
		}

		existingMap := make(map[uint]bool)
		for _, model := range existingModels {
			existingMap[model.ID] = true
		}

		for id := range modelIDs {
			if !existingMap[id] {
				toDelete = append(toDelete, types.Health{
					ResourceType: types.ResourceTypeModel,
					ResourceID:   id,
				})
				deletedCount++
				slog.Debug("发现孤立的模型健康记录", "model_id", id)
			}
		}
	}

	// 批量删除孤立记录
	if len(toDelete) > 0 {
		for _, record := range toDelete {
			if err := db.Where("resource_type = ? AND resource_id = ?",
				record.ResourceType, record.ResourceID).Delete(&types.Health{}).Error; err != nil {
				return fmt.Errorf("删除孤立健康记录失败 (type=%d, id=%d)：%w",
					record.ResourceType, record.ResourceID, err)
			}
		}
		slog.Info("健康表孤立记录清理完成",
			"total_records", len(healthRecords),
			"deleted_count", deletedCount)
	} else {
		slog.Info("未发现孤立的健康记录", "total_records", len(healthRecords))
	}

	return nil
}
