package provider

import (
	"context"
	"fmt"
	"log/slog"
)

// deleteEndpointApp 以应用服务方式实现单个端点删除。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) deleteEndpointApp(ctx context.Context, endpointID uint) error {
	logger := s.logger.With(
		slog.String("operation", "endpoint_delete"),
		slog.Uint64("endpoint_id", uint64(endpointID)),
	)
	logger.Debug("开始删除端点")

	if s.endpointControlRepo == nil {
		return fmt.Errorf("删除端点失败：端点控制仓储未初始化")
	}
	if s.controlTx == nil {
		return fmt.Errorf("删除端点失败：事务执行器未初始化")
	}

	endpoint, err := s.endpointControlRepo.GetEndpoint(ctx, endpointID)
	if err != nil {
		logger.Warn("查询端点失败", slog.Any("error", err))
		_ = s.logEndpointDeleteAudit(ctx, endpointID, "failed", fmt.Sprintf("查询端点失败：%v", err))
		return err
	}

	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		rowsAffected, innerErr := s.endpointControlRepo.DeleteEndpointByID(txCtx, endpointID)
		if innerErr != nil {
			return fmt.Errorf("删除端点失败：%w", innerErr)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("未找到 ID 为 %d 的端点：%w", endpointID, ErrResourceNotFound)
		}

		if endpoint.IsDefault {
			count, countErr := s.endpointControlRepo.CountDefaultEndpointsByPlatform(txCtx, endpoint.PlatformID)
			if countErr != nil {
				return fmt.Errorf("查询默认端点失败：%w", countErr)
			}
			if count == 0 {
				latest, latestErr := s.endpointControlRepo.GetLatestEndpointByPlatform(txCtx, endpoint.PlatformID)
				if latestErr != nil {
					return fmt.Errorf("查询平台最新端点失败：%w", latestErr)
				}
				if latest != nil {
					setRows, setErr := s.endpointControlRepo.SetEndpointDefault(txCtx, latest.ID, true)
					if setErr != nil {
						return fmt.Errorf("设置默认端点失败：%w", setErr)
					}
					if setRows == 0 {
						return fmt.Errorf("未找到 ID 为 %d 的端点：%w", latest.ID, ErrResourceNotFound)
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		logger.Error("删除端点失败", slog.Any("error", err))
		_ = s.logEndpointDeleteAudit(ctx, endpointID, "failed", fmt.Sprintf("删除端点失败：%v", err))
		return fmt.Errorf("删除端点失败：%w", err)
	}

	logger.Info("成功删除端点", slog.Uint64("platform_id", uint64(endpoint.PlatformID)))
	_ = s.logEndpointDeleteAudit(ctx, endpointID, "success", fmt.Sprintf("删除端点成功，platform_id=%d", endpoint.PlatformID))
	return nil
}

func (s *service) logEndpointDeleteAudit(ctx context.Context, endpointID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "endpoint.delete",
		Resource:   "endpoint",
		ResourceID: endpointID,
		Result:     result,
		Detail:     detail,
	})
}
