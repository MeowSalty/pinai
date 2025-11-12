package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// AddModelToPlatform 实现为指定平台添加新模型
func (s *service) AddModelToPlatform(ctx context.Context, platformId uint, model types.Model) (*types.Model, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(platformId)))
	logger.Debug("开始为平台添加模型")

	// 验证平台是否存在
	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台不存在", slog.Any("error", err))
		return nil, err
	}

	// 验证并获取有效的 API 密钥
	validKeys, err := s.validateAndGetAPIKeys(ctx, platformId, model.APIKeys, logger)
	if err != nil {
		return nil, err
	}

	// 保存密钥引用以便后续关联
	model.APIKeys = make([]types.APIKey, len(validKeys))
	for i, key := range validKeys {
		model.APIKeys[i] = *key
	}

	// 设置模型的平台 ID
	model.PlatformID = platformId

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

// BatchAddModelsToPlatform 实现批量为指定平台添加模型（原子性操作）
func (s *service) BatchAddModelsToPlatform(ctx context.Context, platformId uint, models []types.Model) ([]*types.Model, error) {
	logger := s.logger.With(
		slog.Uint64("platform_id", uint64(platformId)),
		slog.Int("model_count", len(models)),
	)
	logger.Debug("开始批量为平台添加模型")

	// 基本参数验证
	if len(models) == 0 {
		logger.Warn("未提供任何模型")
		return nil, fmt.Errorf("必须至少提供一个模型")
	}

	// 验证平台是否存在
	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台验证失败", slog.Any("error", err))
		return nil, err
	}

	// 批量验证所有 API 密钥是否存在且属于该平台
	if err := s.batchValidateAPIKeys(ctx, platformId, models, logger); err != nil {
		logger.Error("API 密钥验证失败", slog.Any("error", err))
		return nil, err
	}

	// 在事务中批量创建模型
	var createdModels []*types.Model
	err := query.Q.Transaction(func(tx *query.Query) error {
		for i := range models {
			model := &models[i]
			model.PlatformID = platformId
			model.ID = 0

			// 创建模型（GORM 会自动处理多对多关系）
			if err := tx.Model.WithContext(ctx).Create(model); err != nil {
				logger.Error("创建模型失败",
					slog.String("model_name", model.Name),
					slog.Any("error", err))
				return fmt.Errorf("创建模型 '%s' 失败：%w", model.Name, err)
			}

			createdModels = append(createdModels, model)
		}
		return nil
	})

	if err != nil {
		logger.Error("批量创建模型事务失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功批量为平台添加模型",
		slog.Int("created_count", len(createdModels)))
	return createdModels, nil
}

// GetModelsByPlatform 实现获取指定平台的所有模型列表
func (s *service) GetModelsByPlatform(ctx context.Context, providerId uint) ([]*types.Model, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(providerId)))
	logger.Debug("开始获取平台的模型列表")

	// 验证平台是否存在
	if err := s.validatePlatformExists(ctx, providerId); err != nil {
		logger.Warn("平台不存在", slog.Any("error", err))
		return nil, err
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
	existingModel, err := s.getModelByID(ctx, modelId)
	if err != nil {
		logger.Warn("查询模型失败", slog.Any("error", err))
		return nil, err
	}

	// 如果提供了 API 密钥列表，则更新关联关系
	if len(model.APIKeys) > 0 {
		validKeys, err := s.validateAndGetAPIKeys(ctx, existingModel.PlatformID, model.APIKeys, logger)
		if err != nil {
			return nil, err
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
	updatedModel, err := s.getModelWithAPIKeys(ctx, modelId)
	if err != nil {
		logger.Error("获取更新后的模型失败", slog.Any("error", err))
		return nil, err
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
