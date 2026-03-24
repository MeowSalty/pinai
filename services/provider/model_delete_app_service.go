package provider

import (
	"context"
	"fmt"
	"log/slog"
)

// deleteModelApp 以应用服务方式实现单个模型删除。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) deleteModelApp(ctx context.Context, modelID uint) error {
	logger := s.logger.With(
		slog.String("operation", "model_delete"),
		slog.Uint64("model_id", uint64(modelID)),
	)
	logger.Debug("开始删除模型")

	if s.modelControlRepo == nil {
		return fmt.Errorf("删除模型失败：模型控制仓储未初始化")
	}
	if s.controlTx == nil {
		return fmt.Errorf("删除模型失败：事务执行器未初始化")
	}

	model, err := s.modelControlRepo.GetModel(ctx, modelID)
	if err != nil {
		logger.Warn("查询模型失败", slog.Any("error", err))
		_ = s.logModelDeleteAudit(ctx, modelID, "failed", fmt.Sprintf("查询模型失败：%v", err))
		return err
	}

	var relationCount int
	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		apiKeys, innerErr := s.modelControlRepo.ListAPIKeysByModel(txCtx, modelID)
		if innerErr != nil {
			return fmt.Errorf("查询模型关联密钥失败：%w", innerErr)
		}

		relationCount = len(apiKeys)
		if relationCount > 0 {
			if innerErr := s.modelControlRepo.ClearModelAPIKeyRelations(txCtx, modelID); innerErr != nil {
				return fmt.Errorf("清理模型与密钥关联关系失败：%w", innerErr)
			}
		}

		rowsAffected, innerErr := s.modelControlRepo.DeleteModelByID(txCtx, modelID)
		if innerErr != nil {
			return fmt.Errorf("删除模型失败：%w", innerErr)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("未找到 ID 为 %d 的模型：%w", modelID, ErrResourceNotFound)
		}

		return nil
	})
	if err != nil {
		logger.Error("删除模型事务失败", slog.Any("error", err))
		_ = s.logModelDeleteAudit(ctx, modelID, "failed", fmt.Sprintf("删除模型失败：%v", err))
		return fmt.Errorf("删除模型失败：%w", err)
	}

	logger.Info("成功删除模型",
		slog.String("model_name", model.Name),
		slog.Uint64("platform_id", uint64(model.PlatformID)),
		slog.Int("api_key_relation_count", relationCount),
	)
	_ = s.logModelDeleteAudit(ctx, modelID, "success", fmt.Sprintf("删除模型成功，platform_id=%d，关联密钥数=%d", model.PlatformID, relationCount))
	return nil
}

func (s *service) logModelDeleteAudit(ctx context.Context, modelID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "model.delete",
		Resource:   "model",
		ResourceID: modelID,
		Result:     result,
		Detail:     detail,
	})
}
