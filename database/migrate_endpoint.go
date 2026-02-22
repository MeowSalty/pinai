package database

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/MeowSalty/pinai/database/types"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// migrateEndpoints 将平台的端点配置从单表结构迁移到独立的端点表。
//
// 该函数执行以下操作：
// 1. 创建 endpoints 表
// 2. 检查 platforms 表是否存在 provider、variant、custom_headers 列
// 3. 如果存在，则将数据迁移到 endpoints 表并删除旧列
//
// 参数：
//   - db: GORM 数据库连接对象
//
// 返回值：
//   - error: 迁移过程中可能发生的错误
func migrateEndpoints(db *gorm.DB) error {
	logger := slog.With("migration", "endpoints")

	// 步骤：检查 platforms 表是否存在需要迁移的列
	migrator := db.Migrator()

	// 临时结构体，用于处理旧表结构中的字段
	type Platform struct {
		ID            uint              `gorm:"primaryKey" json:"id"`                  // 平台 ID
		Provider      string            `json:"provider"`                              // 平台类型
		Variant       string            `json:"variant"`                               // 平台变体
		CustomHeaders map[string]string `gorm:"serializer:json" json:"custom_headers"` // 自定义请求头
	}

	// 检查 provider 列是否存在
	hasProvider := migrator.HasColumn(&Platform{}, "provider")
	logger.Debug("检查 provider 列", "exists", hasProvider)

	// 检查 variant 列是否存在
	hasVariant := migrator.HasColumn(&Platform{}, "variant")
	logger.Debug("检查 variant 列", "exists", hasVariant)

	// 检查 custom_headers 列是否存在
	hasCustomHeaders := migrator.HasColumn(&Platform{}, "custom_headers")
	logger.Debug("检查 custom_headers 列", "exists", hasCustomHeaders)

	// 只有三个列都存在时才执行数据迁移
	if !hasProvider || !hasVariant || !hasCustomHeaders {
		logger.Info("platforms 表不存在需要迁移的列，跳过数据迁移",
			"has_provider", hasProvider,
			"has_variant", hasVariant,
			"has_custom_headers", hasCustomHeaders,
		)
		return nil
	}

	logger.Info("platforms 表存在需要迁移的列，开始数据迁移")

	// 步骤：查询所有平台数据，使用临时结构体
	var platforms []Platform
	if err := db.Find(&platforms).Error; err != nil {
		logger.Error("查询平台数据失败", "error", err)
		return errors.New("查询平台数据失败：" + err.Error())
	}

	logger.Info("查询到平台数据", "count", len(platforms))

	// 步骤：为每个平台回填默认端点到 endpoints 表
	for _, platform := range platforms {
		// 转换平台类型：转为小写，并将 Gemini 映射为 google
		endpointType := strings.ToLower(platform.Provider)
		if endpointType == "gemini" {
			endpointType = "google"
		}

		// 创建默认端点
		endpoint := types.Endpoint{
			PlatformID:      platform.ID,
			EndpointType:    endpointType,
			EndpointVariant: platform.Variant,
			Path:            "",
			CustomHeaders:   platform.CustomHeaders,
			IsDefault:       true,
		}

		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&endpoint).Error; err != nil {
			logger.Error("创建端点失败",
				"platform_id", platform.ID,
				"provider", platform.Provider,
				"variant", platform.Variant,
				"error", err,
			)
			return errors.New("创建端点失败：" + err.Error())
		}

		logger.Debug("成功创建默认端点",
			"platform_id", platform.ID,
			"endpoint_id", endpoint.ID,
			"endpoint_type", endpoint.EndpointType,
			"endpoint_variant", endpoint.EndpointVariant,
		)
	}

	logger.Info("所有平台端点数据迁移完成")

	// 步骤：删除 platforms 表的旧列
	logger.Info("开始删除 platforms 表的旧列")

	// 删除 provider 列
	if err := migrator.DropColumn(&Platform{}, "provider"); err != nil {
		logger.Error("删除 provider 列失败", "error", err)
		return errors.New("删除 provider 列失败：" + err.Error())
	}
	logger.Info("provider 列删除成功")

	// 删除 variant 列
	if err := migrator.DropColumn(&Platform{}, "variant"); err != nil {
		logger.Error("删除 variant 列失败", "error", err)
		return errors.New("删除 variant 列失败：" + err.Error())
	}
	logger.Info("variant 列删除成功")

	// 删除 custom_headers 列
	if err := migrator.DropColumn(&Platform{}, "custom_headers"); err != nil {
		logger.Error("删除 custom_headers 列失败", "error", err)
		return errors.New("删除 custom_headers 列失败：" + err.Error())
	}
	logger.Info("custom_headers 列删除成功")

	logger.Info("端点迁移完成")
	return nil
}
