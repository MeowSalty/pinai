package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// getPlatformByID 辅助函数：根据 ID 查询平台
// 如果未找到或查询失败会返回相应错误
func (s *service) getPlatformByID(ctx context.Context, platformId uint) (*types.Platform, error) {
	platform, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(platformId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", platformId)
		}
		return nil, fmt.Errorf("查询平台失败：%w", err)
	}
	return platform, nil
}

// validatePlatformExists 辅助函数：验证平台是否存在
func (s *service) validatePlatformExists(ctx context.Context, platformId uint) error {
	_, err := s.getPlatformByID(ctx, platformId)
	return err
}

// batchValidateAPIKeys 辅助函数：批量验证 API 密钥
// 通过一次查询验证所有需要的密钥是否存在且属于该平台，避免 N+1 查询问题
func (s *service) batchValidateAPIKeys(ctx context.Context, platformId uint, models []types.Model, logger *slog.Logger) error {
	// 收集所有模型中不重复的 API 密钥 ID（使用 map 作为 Set 去重）
	apiKeyIDSet := make(map[uint]struct{})
	for _, model := range models {
		if len(model.APIKeys) == 0 {
			return fmt.Errorf("模型 '%s' 必须至少关联一个 API 密钥", model.Name)
		}
		for _, key := range model.APIKeys {
			apiKeyIDSet[key.ID] = struct{}{}
		}
	}

	// 转换为切片用于查询
	apiKeyIDs := make([]uint, 0, len(apiKeyIDSet))
	for id := range apiKeyIDSet {
		apiKeyIDs = append(apiKeyIDs, id)
	}

	// 一次性查询所有相关的、属于该平台的有效密钥
	validKeys, err := query.Q.APIKey.WithContext(ctx).
		Where(query.Q.APIKey.ID.In(apiKeyIDs...), query.Q.APIKey.PlatformID.Eq(platformId)).
		Find()
	if err != nil {
		logger.Error("批量查询 API 密钥失败", slog.Any("error", err))
		return fmt.Errorf("批量验证 API 密钥失败：%w", err)
	}

	// 检查是否所有请求的密钥都有效
	if len(validKeys) != len(apiKeyIDs) {
		// 找出哪个密钥是无效的，提供更清晰的错误信息
		validKeyMap := make(map[uint]struct{}, len(validKeys))
		for _, key := range validKeys {
			validKeyMap[key.ID] = struct{}{}
		}
		for _, id := range apiKeyIDs {
			if _, ok := validKeyMap[id]; !ok {
				return fmt.Errorf("API 密钥 ID %d 不存在或不属于平台 ID %d", id, platformId)
			}
		}
	}

	return nil
}

// getAPIKeyByID 辅助函数：根据 ID 查询 API 密钥
// 如果未找到或查询失败会返回相应错误
func (s *service) getAPIKeyByID(ctx context.Context, keyId uint) (*types.APIKey, error) {
	apiKey, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的 API 密钥", keyId)
		}
		return nil, fmt.Errorf("查询 API 密钥失败：%w", err)
	}
	return apiKey, nil
}

// extractAPIKeyIDs 辅助函数：从 APIKey 切片中提取 ID
func extractAPIKeyIDs(apiKeys []types.APIKey) []uint {
	apiKeyIDs := make([]uint, len(apiKeys))
	for i, key := range apiKeys {
		apiKeyIDs[i] = key.ID
	}
	return apiKeyIDs
}

// validateAndGetAPIKeys 辅助函数：验证 API 密钥并返回有效密钥列表
// 验证密钥是否存在、是否属于指定平台，并且至少有一个密钥
func (s *service) validateAndGetAPIKeys(ctx context.Context, platformId uint, apiKeys []types.APIKey, logger *slog.Logger) ([]*types.APIKey, error) {
	// 验证至少关联一个密钥
	if len(apiKeys) == 0 {
		logger.Warn("未提供 API 密钥")
		return nil, fmt.Errorf("模型必须至少关联一个 API 密钥")
	}

	// 提取 API 密钥 ID
	apiKeyIDs := extractAPIKeyIDs(apiKeys)

	// 验证所有密钥是否存在且属于该平台
	validKeys, err := query.Q.APIKey.WithContext(ctx).
		Where(query.Q.APIKey.ID.In(apiKeyIDs...), query.Q.APIKey.PlatformID.Eq(platformId)).
		Find()
	if err != nil {
		logger.Error("验证 API 密钥失败", slog.Any("error", err))
		return nil, fmt.Errorf("验证 API 密钥失败：%w", err)
	}

	// 检查是否所有密钥都有效
	if len(validKeys) != len(apiKeyIDs) {
		logger.Warn("部分 API 密钥不存在或不属于该平台")
		return nil, fmt.Errorf("部分 API 密钥不存在或不属于平台 ID %d", platformId)
	}

	return validKeys, nil
}

// getModelByID 辅助函数：根据 ID 查询模型
func (s *service) getModelByID(ctx context.Context, modelId uint) (*types.Model, error) {
	model, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
		}
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}
	return model, nil
}

// getModelWithAPIKeys 辅助函数：根据 ID 查询模型（预加载 API 密钥）
func (s *service) getModelWithAPIKeys(ctx context.Context, modelId uint) (*types.Model, error) {
	model, err := query.Q.Model.WithContext(ctx).
		Preload(query.Q.Model.APIKeys).
		Where(query.Q.Model.ID.Eq(modelId)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
		}
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}
	return model, nil
}

// removeOrphanedModels 辅助函数：检测并移除指定平台的孤立模型
// 孤立模型是指没有和任何密钥关联的模型
// 返回被删除的模型数量和可能的错误
func (s *service) removeOrphanedModels(ctx context.Context, platformId uint, logger *slog.Logger) (int64, error) {
	logger.Debug("开始检测孤立模型")

	// 查询该平台下的所有模型（预加载密钥关联）
	models, err := query.Q.Model.WithContext(ctx).
		Preload(query.Q.Model.APIKeys).
		Where(query.Q.Model.PlatformID.Eq(platformId)).
		Find()
	if err != nil {
		logger.Error("查询平台模型失败", slog.Any("error", err))
		return 0, fmt.Errorf("查询平台模型失败：%w", err)
	}

	// 找出孤立模型（没有关联任何密钥的模型）
	var orphanedModelIDs []uint
	for _, model := range models {
		if len(model.APIKeys) == 0 {
			orphanedModelIDs = append(orphanedModelIDs, model.ID)
			logger.Debug("发现孤立模型",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.String("model_name", model.Name))
		}
	}

	// 如果没有孤立模型，直接返回
	if len(orphanedModelIDs) == 0 {
		logger.Debug("没有发现孤立模型")
		return 0, nil
	}

	// 批量删除孤立模型
	result, err := query.Q.Model.WithContext(ctx).
		Where(query.Q.Model.ID.In(orphanedModelIDs...)).
		Delete()
	if err != nil {
		logger.Error("删除孤立模型失败",
			slog.Int("orphaned_count", len(orphanedModelIDs)),
			slog.Any("error", err))
		return 0, fmt.Errorf("删除孤立模型失败：%w", err)
	}

	logger.Info("成功移除孤立模型",
		slog.Int64("deleted_count", result.RowsAffected),
		slog.Any("model_ids", orphanedModelIDs))

	return result.RowsAffected, nil
}
