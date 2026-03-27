package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// addEndpointToPlatformApp 以应用服务方式实现单个端点新增。
// 事务边界在应用服务层声明，仓储仅负责数据访问。
func (s *service) addEndpointToPlatformApp(ctx context.Context, platformID uint, endpoint types.Endpoint) (*types.Endpoint, error) {
	logger := s.logger.With(
		slog.String("operation", "endpoint_create"),
		slog.Uint64("platform_id", uint64(platformID)),
	)
	logger.Debug("开始为平台添加端点")

	if s.endpointControlRepo == nil {
		return nil, fmt.Errorf("创建端点失败：端点控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("创建端点失败：事务执行器未初始化")
	}

	exists, err := s.endpointControlRepo.ExistsPlatform(ctx, platformID)
	if err != nil {
		logger.Error("检查平台是否存在失败", slog.Any("error", err))
		_ = s.logEndpointCreateAudit(ctx, platformID, 0, "failed", fmt.Sprintf("检查平台是否存在失败：%v", err))
		return nil, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		logger.Warn("平台不存在")
		err = fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformID, ErrResourceNotFound)
		_ = s.logEndpointCreateAudit(ctx, platformID, 0, "failed", err.Error())
		return nil, err
	}

	endpoint.ID = 0
	endpoint.PlatformID = platformID

	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		if innerErr := s.endpointControlRepo.CreateEndpoint(txCtx, &endpoint); innerErr != nil {
			return innerErr
		}

		if endpoint.IsDefault {
			count, innerErr := s.endpointControlRepo.CountDefaultEndpointsByPlatform(txCtx, platformID)
			if innerErr != nil {
				return fmt.Errorf("默认端点校验失败：%w", innerErr)
			}
			if count > 1 {
				return fmt.Errorf("平台 ID %d 存在多个默认端点：%w", platformID, ErrDefaultConflict)
			}
		}

		return nil
	})
	if err != nil {
		logger.Error("创建端点失败", slog.Any("error", err))
		_ = s.logEndpointCreateAudit(ctx, platformID, 0, "failed", fmt.Sprintf("创建端点失败：%v", err))
		return nil, fmt.Errorf("创建端点失败：%w", err)
	}

	logger.Info("成功为平台添加端点", slog.Uint64("endpoint_id", uint64(endpoint.ID)))
	_ = s.logEndpointCreateAudit(ctx, platformID, endpoint.ID, "success", fmt.Sprintf("创建端点成功，platform_id=%d", platformID))
	return &endpoint, nil
}

func (s *service) logEndpointCreateAudit(ctx context.Context, platformID, endpointID uint, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     "endpoint.create",
		Resource:   "endpoint",
		ResourceID: endpointID,
		Result:     result,
		Detail:     detail,
	})
}
