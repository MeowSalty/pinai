package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// batchUpdateModelsApp 以应用服务方式实现批量更新模型。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) batchUpdateModelsApp(ctx context.Context, platformID uint, updateItems []ModelUpdateItem) ([]*types.Model, error) {
	logger := s.logger.With(
		slog.String("operation", "batch_update_models"),
		slog.Uint64("platform_id", uint64(platformID)),
		slog.Int("model_count", len(updateItems)),
	)
	logger.Debug("开始批量更新模型")

	if s.modelControlRepo == nil {
		return nil, fmt.Errorf("批量更新模型失败：模型控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("批量更新模型失败：事务执行器未初始化")
	}

	if len(updateItems) == 0 {
		logger.Warn("未提供任何更新项")
		return nil, fmt.Errorf("必须至少提供一个模型更新项")
	}

	exists, err := s.modelControlRepo.ExistsPlatform(ctx, platformID)
	if err != nil {
		logger.Error("检查平台是否存在失败", slog.Any("error", err))
		return nil, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		logger.Warn("平台不存在")
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformID, ErrResourceNotFound)
	}

	modelIDs := make([]uint, len(updateItems))
	for i, item := range updateItems {
		modelIDs[i] = item.ID
	}

	models, err := s.modelControlRepo.ListModelsByIDs(ctx, modelIDs)
	if err != nil {
		logger.Error("批量查询模型失败", slog.Any("error", err))
		return nil, err
	}

	if len(models) != len(modelIDs) {
		foundIDs := make(map[uint]struct{}, len(models))
		for _, model := range models {
			foundIDs[model.ID] = struct{}{}
		}

		var missingIDs []uint
		for _, id := range modelIDs {
			if _, ok := foundIDs[id]; !ok {
				missingIDs = append(missingIDs, id)
			}
		}

		return nil, fmt.Errorf("以下模型不存在：%v: %w", missingIDs, ErrResourceNotFound)
	}

	modelByID := make(map[uint]*types.Model, len(models))
	for _, model := range models {
		if model.PlatformID != platformID {
			logger.Warn("模型不属于指定平台",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Uint64("model_platform_id", uint64(model.PlatformID)),
				slog.Uint64("expected_platform_id", uint64(platformID)))
			return nil, fmt.Errorf("模型 ID %d 不属于平台 ID %d: %w", model.ID, platformID, ErrResourceNotBelong)
		}
		modelByID[model.ID] = model
	}

	updatedModels := make([]*types.Model, 0, len(updateItems))
	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		for i := range updateItems {
			item := updateItems[i]
			itemLogger := logger.With(slog.Uint64("model_id", uint64(item.ID)))

			existingModel, ok := modelByID[item.ID]
			if !ok {
				return fmt.Errorf("未找到 ID 为 %d 的模型：%w", item.ID, ErrResourceNotFound)
			}

			if len(item.APIKeys) > 0 {
				apiKeyIDs := extractAPIKeyIDs(item.APIKeys)
				validKeys, innerErr := s.modelControlRepo.ListAPIKeysByPlatformAndIDs(txCtx, existingModel.PlatformID, apiKeyIDs)
				if innerErr != nil {
					itemLogger.Error("校验模型关联密钥失败", slog.Any("error", innerErr))
					return fmt.Errorf("校验模型关联密钥失败：%w", innerErr)
				}
				if len(validKeys) != len(apiKeyIDs) {
					return fmt.Errorf("部分 API 密钥不存在或不属于平台 ID %d：%w", existingModel.PlatformID, ErrResourceNotBelong)
				}

				if innerErr = s.modelControlRepo.ReplaceModelAPIKeys(txCtx, item.ID, validKeys); innerErr != nil {
					itemLogger.Error("更新模型密钥关联失败", slog.Any("error", innerErr))
					return fmt.Errorf("更新模型 ID %d 的密钥关联失败：%w", item.ID, innerErr)
				}
				itemLogger.Debug("成功更新模型密钥关联", slog.Int("api_key_count", len(validKeys)))
			}

			updates := make(map[string]interface{})
			if item.Name != "" && item.Name != existingModel.Name {
				updates["name"] = item.Name
			}
			if item.Alias != "" && item.Alias != existingModel.Alias {
				updates["alias"] = item.Alias
			}

			if len(updates) > 0 {
				rowsAffected, innerErr := s.modelControlRepo.UpdateModelFields(txCtx, item.ID, updates)
				if innerErr != nil {
					itemLogger.Error("更新模型字段失败", slog.Any("error", innerErr))
					return fmt.Errorf("更新模型 ID %d 的字段失败：%w", item.ID, innerErr)
				}
				if rowsAffected == 0 {
					itemLogger.Warn("模型更新无影响行")
				}
			}

			updatedModel, innerErr := s.modelControlRepo.GetModelWithAPIKeys(txCtx, item.ID)
			if innerErr != nil {
				itemLogger.Error("获取更新后的模型失败", slog.Any("error", innerErr))
				return innerErr
			}

			updatedModels = append(updatedModels, updatedModel)
		}

		return nil
	})
	if err != nil {
		logger.Error("批量更新模型事务失败", slog.Any("error", err))
		_ = s.logModelBatchUpdateAudit(ctx, platformID, "failed", fmt.Sprintf("批量更新模型失败：%v", err))
		return nil, fmt.Errorf("批量更新模型失败：%w", err)
	}

	logger.Info("成功批量更新模型", slog.Int("updated_count", len(updatedModels)))
	_ = s.logModelBatchUpdateAudit(ctx, platformID, "success", fmt.Sprintf("批量更新模型成功，更新数量 %d", len(updatedModels)))
	return updatedModels, nil
}

func (s *service) logModelBatchUpdateAudit(ctx context.Context, platformID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "model.batch_update",
		Resource:   "platform",
		ResourceID: platformID,
		Result:     result,
		Detail:     detail,
	})
}
