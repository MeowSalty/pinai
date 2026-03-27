package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// UpdateModelHealthEnabled 更新模型健康状态（控制面最小写路径应用服务）。
func (s *service) UpdateModelHealthEnabled(ctx context.Context, modelID uint, enabled bool) (types.HealthStatus, error) {
	logger := s.logger.With(
		slog.String("operation", "update_model_health"),
		slog.Uint64("model_id", uint64(modelID)),
		slog.Bool("enabled", enabled),
	)

	if s.modelControlRepo == nil {
		return types.HealthStatusUnknown, fmt.Errorf("更新模型健康状态失败：模型控制仓储未初始化")
	}
	if s.controlTx == nil {
		return types.HealthStatusUnknown, fmt.Errorf("更新模型健康状态失败：事务执行器未初始化")
	}

	if _, err := s.modelControlRepo.GetModel(ctx, modelID); err != nil {
		logger.Warn("模型不存在或查询失败", slog.Any("error", err))
		return types.HealthStatusUnknown, fmt.Errorf("查询模型失败：%w", err)
	}

	status := types.HealthStatusUnavailable
	auditResult := "success"
	auditDetail := "禁用模型健康状态"

	err := s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		if enabled {
			if innerErr := s.modelControlRepo.EnableModelHealth(txCtx, modelID); innerErr != nil {
				return innerErr
			}
			status = types.HealthStatusUnknown
			auditDetail = "启用模型健康状态"
			return nil
		}

		if innerErr := s.modelControlRepo.DisableModelHealth(txCtx, modelID); innerErr != nil {
			return innerErr
		}
		return nil
	})
	if err != nil {
		auditResult = "failed"
		auditDetail = fmt.Sprintf("更新模型健康状态失败：%v", err)
		logger.Error("更新模型健康状态失败", slog.Any("error", err))
		_ = s.logModelControlAudit(ctx, modelID, auditResult, auditDetail)
		return types.HealthStatusUnknown, fmt.Errorf("更新模型健康状态失败：%w", err)
	}

	logger.Info("更新模型健康状态成功", slog.Int("status", int(status)))
	_ = s.logModelControlAudit(ctx, modelID, auditResult, auditDetail)
	return status, nil
}

func (s *service) logModelControlAudit(ctx context.Context, modelID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "model.health.update",
		Resource:   "model",
		ResourceID: modelID,
		Result:     result,
		Detail:     detail,
	})
}
