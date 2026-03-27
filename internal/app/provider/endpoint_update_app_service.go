package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// updateEndpointApp 以应用服务方式实现单个端点更新。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) updateEndpointApp(ctx context.Context, endpointID uint, endpoint types.Endpoint) (*types.Endpoint, error) {
	logger := s.logger.With(
		slog.String("operation", "endpoint_update"),
		slog.Uint64("endpoint_id", uint64(endpointID)),
	)
	logger.Debug("开始更新端点")

	if s.endpointControlRepo == nil {
		return nil, fmt.Errorf("更新端点失败：端点控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("更新端点失败：事务执行器未初始化")
	}

	existing, err := s.endpointControlRepo.GetEndpoint(ctx, endpointID)
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			logger.Warn("端点不存在", slog.Any("error", err))
		} else {
			logger.Error("查询端点失败", slog.Any("error", err))
		}
		_ = s.logEndpointUpdateAudit(ctx, endpointID, "failed", fmt.Sprintf("查询端点失败：%v", err))
		return nil, err
	}

	payload := types.Endpoint{}
	selectFields := make([]string, 0, 5)
	if endpoint.EndpointType != "" {
		payload.EndpointType = endpoint.EndpointType
		selectFields = append(selectFields, "endpoint_type")
	}
	if endpoint.EndpointVariant != "" {
		payload.EndpointVariant = endpoint.EndpointVariant
		selectFields = append(selectFields, "endpoint_variant")
	}
	if endpoint.Path != "" {
		payload.Path = endpoint.Path
		selectFields = append(selectFields, "path")
	}
	if endpoint.CustomHeaders != nil {
		payload.CustomHeaders = endpoint.CustomHeaders
		selectFields = append(selectFields, "custom_headers")
	}
	if endpoint.IsDefault {
		payload.IsDefault = true
		selectFields = append(selectFields, "is_default")
	}

	var updatedEndpoint *types.Endpoint
	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		if len(selectFields) > 0 {
			rowsAffected, innerErr := s.endpointControlRepo.UpdateEndpointFields(txCtx, endpointID, payload, selectFields)
			if innerErr != nil {
				return fmt.Errorf("更新端点失败：%w", innerErr)
			}
			if rowsAffected == 0 {
				return fmt.Errorf("未找到 ID 为 %d 的端点：%w", endpointID, ErrResourceNotFound)
			}
		}

		if endpoint.IsDefault {
			count, innerErr := s.endpointControlRepo.CountDefaultEndpointsByPlatform(txCtx, existing.PlatformID)
			if innerErr != nil {
				return fmt.Errorf("默认端点校验失败：%w", innerErr)
			}
			if count > 1 {
				return fmt.Errorf("平台 ID %d 存在多个默认端点：%w", existing.PlatformID, ErrDefaultConflict)
			}
		}

		updated, innerErr := s.endpointControlRepo.GetEndpoint(txCtx, endpointID)
		if innerErr != nil {
			return fmt.Errorf("获取更新后的端点失败：%w", innerErr)
		}
		updatedEndpoint = updated
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrDefaultConflict) || errors.Is(err, ErrResourceNotFound) || errors.Is(err, ErrInvalidArgument) {
			logger.Warn("更新端点失败", slog.Any("error", err))
		} else {
			logger.Error("更新端点失败", slog.Any("error", err))
		}
		_ = s.logEndpointUpdateAudit(ctx, endpointID, "failed", fmt.Sprintf("更新端点失败：%v", err))
		return nil, err
	}

	logger.Info("成功更新端点", slog.Uint64("platform_id", uint64(updatedEndpoint.PlatformID)))
	_ = s.logEndpointUpdateAudit(ctx, endpointID, "success", fmt.Sprintf("更新端点成功，platform_id=%d", updatedEndpoint.PlatformID))
	return updatedEndpoint, nil
}

func (s *service) logEndpointUpdateAudit(ctx context.Context, endpointID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "endpoint.update",
		Resource:   "endpoint",
		ResourceID: endpointID,
		Result:     result,
		Detail:     detail,
	})
}
