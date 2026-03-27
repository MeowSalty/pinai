package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// UpdateKeyHealthEnabled 更新密钥健康状态（控制面最小写路径应用服务）。
func (s *service) UpdateKeyHealthEnabled(ctx context.Context, keyID uint, enabled bool) (types.HealthStatus, error) {
	logger := s.logger.With(
		slog.String("operation", "update_key_health"),
		slog.Uint64("key_id", uint64(keyID)),
		slog.Bool("enabled", enabled),
	)

	if s.keyControlRepo == nil {
		return types.HealthStatusUnknown, fmt.Errorf("更新密钥健康状态失败：密钥控制仓储未初始化")
	}
	if s.controlTx == nil {
		return types.HealthStatusUnknown, fmt.Errorf("更新密钥健康状态失败：事务执行器未初始化")
	}

	if _, err := s.keyControlRepo.GetAPIKey(ctx, keyID); err != nil {
		logger.Warn("密钥不存在或查询失败", slog.Any("error", err))
		return types.HealthStatusUnknown, fmt.Errorf("查询密钥失败：%w", err)
	}

	status := types.HealthStatusUnavailable
	auditResult := "success"
	auditDetail := "禁用密钥健康状态"

	err := s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		if enabled {
			if innerErr := s.keyControlRepo.EnableAPIKeyHealth(txCtx, keyID); innerErr != nil {
				return innerErr
			}
			status = types.HealthStatusUnknown
			auditDetail = "启用密钥健康状态"
			return nil
		}

		if innerErr := s.keyControlRepo.DisableAPIKeyHealth(txCtx, keyID); innerErr != nil {
			return innerErr
		}
		return nil
	})
	if err != nil {
		auditResult = "failed"
		auditDetail = fmt.Sprintf("更新密钥健康状态失败：%v", err)
		logger.Error("更新密钥健康状态失败", slog.Any("error", err))
		_ = s.logKeyControlAudit(ctx, keyID, auditResult, auditDetail)
		return types.HealthStatusUnknown, fmt.Errorf("更新密钥健康状态失败：%w", err)
	}

	logger.Info("更新密钥健康状态成功", slog.Int("status", int(status)))
	_ = s.logKeyControlAudit(ctx, keyID, auditResult, auditDetail)
	return status, nil
}

func (s *service) logKeyControlAudit(ctx context.Context, keyID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "key.health.update",
		Resource:   "key",
		ResourceID: keyID,
		Result:     result,
		Detail:     detail,
	})
}
