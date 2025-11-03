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

	// 设置模型的平台 ID
	model.PlatformID = providerId

	// 创建模型
	model.ID = 0
	if err := query.Q.Model.WithContext(ctx).Create(&model); err != nil {
		logger.Error("创建模型失败", slog.Any("error", err))
		return nil, fmt.Errorf("创建模型失败：%w", err)
	}

	logger.Info("成功为平台添加模型", slog.String("model_name", model.Name), slog.Uint64("model_id", uint64(model.ID)))
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

	// 获取模型列表
	models, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.PlatformID.Eq(providerId)).Find()
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

	// 只更新非零值字段
	result, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).Updates(model)
	if err != nil {
		logger.Error("更新模型失败", slog.Any("error", err))
		return nil, fmt.Errorf("更新 ID 为 %d 的模型失败：%w", modelId, err)
	}
	if result.RowsAffected == 0 {
		logger.Warn("模型不存在")
		return nil, fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
	}

	// 返回更新后的完整对象
	updatedModel, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).First()
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
