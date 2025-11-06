package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// AddModelToPlatform 实现为指定平台添加新模型
func (s *service) AddModelToPlatform(ctx context.Context, providerId uint, model types.Model) (*types.Model, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(providerId)))
	logger.Debug("开始为平台添加模型")

	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("平台不存在")
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		logger.Error("查询平台失败", slog.Any("error", err))
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 验证 API 密钥
	if len(model.APIKeys) > 0 {
		apiKeyIDs := make([]uint, len(model.APIKeys))
		for i, key := range model.APIKeys {
			apiKeyIDs[i] = key.ID
		}

		// 验证所有密钥是否存在且属于该平台
		validKeys, err := query.Q.APIKey.WithContext(ctx).
			Where(query.Q.APIKey.ID.In(apiKeyIDs...), query.Q.APIKey.PlatformID.Eq(providerId)).
			Find()
		if err != nil {
			logger.Error("验证 API 密钥失败", slog.Any("error", err))
			return nil, fmt.Errorf("验证 API 密钥失败：%w", err)
		}
		if len(validKeys) != len(apiKeyIDs) {
			logger.Warn("部分 API 密钥不存在或不属于该平台")
			return nil, fmt.Errorf("部分 API 密钥不存在或不属于平台 ID %d", providerId)
		}

		// 保存密钥引用以便后续关联
		model.APIKeys = make([]types.APIKey, len(validKeys))
		for i, key := range validKeys {
			model.APIKeys[i] = *key
		}
	}

	// 设置模型的平台 ID
	model.PlatformID = providerId

	// 创建模型（GORM 会自动处理多对多关系）
	model.ID = 0
	if err := query.Q.Model.WithContext(ctx).Create(&model); err != nil {
		logger.Error("创建模型失败", slog.Any("error", err))
		return nil, fmt.Errorf("创建模型失败：%w", err)
	}

	logger.Info("成功为平台添加模型",
		slog.String("model_name", model.Name),
		slog.Uint64("model_id", uint64(model.ID)),
		slog.Int("api_key_count", len(model.APIKeys)))
	return &model, nil
}

// GetModelsByProvider 实现获取指定平台的所有模型列表
func (s *service) GetModelsByPlatform(ctx context.Context, providerId uint) ([]*types.Model, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(providerId)))
	logger.Debug("开始获取平台的模型列表")

	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("平台不存在")
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		logger.Error("查询平台失败", slog.Any("error", err))
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 获取模型列表（预加载关联的 API 密钥）
	models, err := query.Q.Model.WithContext(ctx).
		Preload(query.Q.Model.APIKeys).
		Where(query.Q.Model.PlatformID.Eq(providerId)).
		Find()
	if err != nil {
		logger.Error("获取模型列表失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取平台 ID 为 %d 的模型失败：%w", providerId, err)
	}

	logger.Info("成功获取平台的模型列表", slog.Int("count", len(models)))
	return models, nil
}

// UpdateModel 实现更新指定模型信息
func (s *service) UpdateModel(ctx context.Context, modelId uint, model types.Model) (*types.Model, error) {
	logger := s.logger.With(slog.Uint64("model_id", uint64(modelId)))
	logger.Debug("开始更新模型")

	// 查询现有模型
	existingModel, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("模型不存在")
			return nil, fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
		}
		logger.Error("查询模型失败", slog.Any("error", err))
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}

	// 如果提供了 API 密钥列表，则更新关联关系
	if len(model.APIKeys) > 0 {
		apiKeyIDs := make([]uint, len(model.APIKeys))
		for i, key := range model.APIKeys {
			apiKeyIDs[i] = key.ID
		}

		// 验证所有密钥是否存在且属于同一平台
		validKeys, err := query.Q.APIKey.WithContext(ctx).
			Where(query.Q.APIKey.ID.In(apiKeyIDs...), query.Q.APIKey.PlatformID.Eq(existingModel.PlatformID)).
			Find()
		if err != nil {
			logger.Error("验证 API 密钥失败", slog.Any("error", err))
			return nil, fmt.Errorf("验证 API 密钥失败：%w", err)
		}
		if len(validKeys) != len(apiKeyIDs) {
			logger.Warn("部分 API 密钥不存在或不属于该平台")
			return nil, fmt.Errorf("部分 API 密钥不存在或不属于平台 ID %d", existingModel.PlatformID)
		}

		// 使用 Association 的 Replace 方法更新多对多关系
		apiKeyPtrs := make([]*types.APIKey, len(validKeys))
		copy(apiKeyPtrs, validKeys)
		if err := query.Q.Model.APIKeys.Model(existingModel).Replace(apiKeyPtrs...); err != nil {
			logger.Error("更新模型密钥关联失败", slog.Any("error", err))
			return nil, fmt.Errorf("更新模型密钥关联失败：%w", err)
		}

		logger.Info("成功更新模型密钥关联", slog.Int("api_key_count", len(validKeys)))
	}

	// 更新模型的其他字段（排除 APIKeys 字段，因为已单独处理）
	if model.Name != "" || model.Alias != "" {
		result, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).Updates(model)
		if err != nil {
			logger.Error("更新模型字段失败", slog.Any("error", err))
			return nil, fmt.Errorf("更新 ID 为 %d 的模型失败：%w", modelId, err)
		}
		if result.RowsAffected == 0 {
			logger.Warn("模型更新无影响行")
		}
	}

	// 返回更新后的完整对象（包含关联的 API 密钥）
	updatedModel, err := query.Q.Model.WithContext(ctx).
		Preload(query.Q.Model.APIKeys).
		Where(query.Q.Model.ID.Eq(modelId)).
		First()
	if err != nil {
		logger.Error("获取更新后的模型失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取更新后的 ID 为 %d 的模型失败：%w", modelId, err)
	}

	logger.Info("成功更新模型", slog.String("model_name", updatedModel.Name))
	return updatedModel, nil
}

// DeleteModel 实现删除指定模型
func (s *service) DeleteModel(ctx context.Context, modelId uint) error {
	logger := s.logger.With(slog.Uint64("model_id", uint64(modelId)))
	logger.Debug("开始删除模型")

	// 删除模型
	result, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).Delete()
	if err != nil {
		logger.Error("删除模型失败", slog.Any("error", err))
		return fmt.Errorf("删除 ID 为 %d 的模型失败：%w", modelId, err)
	}
	if result.RowsAffected == 0 {
		logger.Warn("模型不存在")
		return fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
	}

	logger.Info("成功删除模型")
	return nil
}
