package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// UpdatePlatformHealthEnabled 更新平台健康状态（控制面最小写路径应用服务）。
func (s *service) UpdatePlatformHealthEnabled(ctx context.Context, platformID uint, enabled bool) (types.HealthStatus, error) {
	logger := s.logger.With(
		slog.String("operation", "update_platform_health"),
		slog.Uint64("platform_id", uint64(platformID)),
		slog.Bool("enabled", enabled),
	)

	if s.platformControlRepo == nil {
		return types.HealthStatusUnknown, fmt.Errorf("更新平台健康状态失败：平台控制仓储未初始化")
	}
	if s.controlTx == nil {
		return types.HealthStatusUnknown, fmt.Errorf("更新平台健康状态失败：事务执行器未初始化")
	}

	exists, err := s.platformControlRepo.ExistsPlatform(ctx, platformID)
	if err != nil {
		logger.Error("检查平台是否存在失败", slog.Any("error", err))
		return types.HealthStatusUnknown, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		logger.Warn("平台不存在")
		return types.HealthStatusUnknown, fmt.Errorf("平台不存在：%w", ErrResourceNotFound)
	}

	status := types.HealthStatusUnavailable
	auditResult := "success"
	auditDetail := "禁用平台健康状态"

	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		if enabled {
			if innerErr := s.platformControlRepo.EnablePlatformHealth(txCtx, platformID); innerErr != nil {
				return innerErr
			}
			status = types.HealthStatusUnknown
			auditDetail = "启用平台健康状态"
			return nil
		}

		if innerErr := s.platformControlRepo.DisablePlatformHealth(txCtx, platformID); innerErr != nil {
			return innerErr
		}
		return nil
	})
	if err != nil {
		auditResult = "failed"
		auditDetail = fmt.Sprintf("更新平台健康状态失败：%v", err)
		logger.Error("更新平台健康状态失败", slog.Any("error", err))
		_ = s.logControlAudit(ctx, platformID, auditResult, auditDetail)
		return types.HealthStatusUnknown, fmt.Errorf("更新平台健康状态失败：%w", err)
	}

	logger.Info("更新平台健康状态成功", slog.Int("status", int(status)))
	_ = s.logControlAudit(ctx, platformID, auditResult, auditDetail)
	return status, nil
}

func (s *service) logControlAudit(ctx context.Context, platformID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "platform.health.update",
		Resource:   "platform",
		ResourceID: platformID,
		Result:     result,
		Detail:     detail,
	})
}
