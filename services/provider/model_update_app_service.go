package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// updateModelApp 以应用服务方式实现单个模型更新。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) updateModelApp(ctx context.Context, modelID uint, model types.Model) (*types.Model, error) {
	logger := s.logger.With(
		slog.String("operation", "model_update"),
		slog.Uint64("model_id", uint64(modelID)),
	)
	logger.Debug("开始更新模型")

	if s.modelControlRepo == nil {
		return nil, fmt.Errorf("更新模型失败：模型控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("更新模型失败：事务执行器未初始化")
	}

	existingModel, err := s.modelControlRepo.GetModel(ctx, modelID)
	if err != nil {
		logger.Warn("查询模型失败", slog.Any("error", err))
		_ = s.logModelUpdateAudit(ctx, modelID, "failed", fmt.Sprintf("查询模型失败：%v", err))
		return nil, err
	}

	var validKeys []*types.APIKey
	if len(model.APIKeys) > 0 {
		apiKeyIDs := extractAPIKeyIDs(model.APIKeys)
		validKeys, err = s.modelControlRepo.ListAPIKeysByPlatformAndIDs(ctx, existingModel.PlatformID, apiKeyIDs)
		if err != nil {
			logger.Error("校验模型关联密钥失败", slog.Any("error", err))
			_ = s.logModelUpdateAudit(ctx, modelID, "failed", fmt.Sprintf("校验模型关联密钥失败：%v", err))
			return nil, fmt.Errorf("校验模型关联密钥失败：%w", err)
		}
		if len(validKeys) != len(apiKeyIDs) {
			logger.Warn("部分 API 密钥不存在或不属于指定平台", slog.Uint64("platform_id", uint64(existingModel.PlatformID)))
			err = fmt.Errorf("部分 API 密钥不存在或不属于平台 ID %d：%w", existingModel.PlatformID, ErrResourceNotBelong)
			_ = s.logModelUpdateAudit(ctx, modelID, "failed", err.Error())
			return nil, err
		}
	}

	updates := make(map[string]interface{})
	if model.Name != "" && model.Name != existingModel.Name {
		updates["name"] = model.Name
	}
	if model.Alias != "" && model.Alias != existingModel.Alias {
		updates["alias"] = model.Alias
	}

	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		if len(validKeys) > 0 {
			if innerErr := s.modelControlRepo.ReplaceModelAPIKeys(txCtx, modelID, validKeys); innerErr != nil {
				return fmt.Errorf("更新模型密钥关联失败：%w", innerErr)
			}
		}

		if len(updates) > 0 {
			rowsAffected, innerErr := s.modelControlRepo.UpdateModelFields(txCtx, modelID, updates)
			if innerErr != nil {
				return fmt.Errorf("更新模型字段失败：%w", innerErr)
			}
			if rowsAffected == 0 {
				logger.Warn("模型更新无影响行")
			}
		}

		return nil
	})
	if err != nil {
		logger.Error("更新模型事务失败", slog.Any("error", err))
		_ = s.logModelUpdateAudit(ctx, modelID, "failed", fmt.Sprintf("更新模型失败：%v", err))
		return nil, fmt.Errorf("更新模型失败：%w", err)
	}

	updatedModel, err := s.modelControlRepo.GetModelWithAPIKeys(ctx, modelID)
	if err != nil {
		logger.Error("获取更新后的模型失败", slog.Any("error", err))
		_ = s.logModelUpdateAudit(ctx, modelID, "failed", fmt.Sprintf("获取更新后的模型失败：%v", err))
		return nil, err
	}

	logger.Info("成功更新模型",
		slog.String("model_name", updatedModel.Name),
		slog.Uint64("platform_id", uint64(updatedModel.PlatformID)),
		slog.Bool("api_keys_updated", len(validKeys) > 0),
		slog.Int("updated_field_count", len(updates)),
	)
	_ = s.logModelUpdateAudit(ctx, modelID, "success", fmt.Sprintf("更新模型成功，platform_id=%d，更新字段数=%d，是否更新密钥=%t", updatedModel.PlatformID, len(updates), len(validKeys) > 0))

	return updatedModel, nil
}

func (s *service) logModelUpdateAudit(ctx context.Context, modelID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "model.update",
		Resource:   "model",
		ResourceID: modelID,
		Result:     result,
		Detail:     detail,
	})
}
