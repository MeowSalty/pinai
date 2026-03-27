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
	return s.batchAddModelsToPlatformApp(ctx, platformId, models)
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

// GetModel 实现获取指定模型详情
func (s *service) GetModel(ctx context.Context, modelId uint) (*types.Model, error) {
	logger := s.logger.With(slog.Uint64("model_id", uint64(modelId)))
	logger.Debug("开始获取模型详情")

	// 查询模型（预加载关联的 API 密钥）
	model, err := s.getModelWithAPIKeys(ctx, modelId)
	if err != nil {
		logger.Warn("模型不存在或查询失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功获取模型详情")
	return model, nil
}

// UpdateModel 实现更新指定模型信息
func (s *service) UpdateModel(ctx context.Context, modelId uint, model types.Model) (*types.Model, error) {
	return s.updateModelApp(ctx, modelId, model)
}

// DeleteModel 实现删除指定模型
func (s *service) DeleteModel(ctx context.Context, modelId uint) error {
	return s.deleteModelApp(ctx, modelId)
}

// BatchDeleteModels 实现批量删除指定平台的模型（原子性操作）
func (s *service) BatchDeleteModels(ctx context.Context, platformId uint, modelIds []uint) (int, error) {
	return s.batchDeleteModelsApp(ctx, platformId, modelIds)
}

// BatchUpdateModels 实现批量更新指定平台的模型（原子性操作）
func (s *service) BatchUpdateModels(ctx context.Context, platformId uint, updateItems []ModelUpdateItem) ([]*types.Model, error) {
	return s.batchUpdateModelsApp(ctx, platformId, updateItems)
}
