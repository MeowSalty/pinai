package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// endpointControlQueryRepository 是基于 database/query 的端点控制面仓储实现。
type endpointControlQueryRepository struct {
	logger *slog.Logger
}

// NewEndpointControlQueryRepository 创建端点控制面仓储实现。
func NewEndpointControlQueryRepository(logger *slog.Logger) EndpointControlRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &endpointControlQueryRepository{
		logger: logger.WithGroup("endpoint_control_repo"),
	}
}

// ExistsPlatform 检查平台是否存在。
func (r *endpointControlQueryRepository) ExistsPlatform(ctx context.Context, platformID uint) (bool, error) {
	q := queryFromContextOrDefault(ctx)
	count, err := q.Platform.WithContext(ctx).Where(q.Platform.ID.Eq(platformID)).Count()
	if err != nil {
		r.logger.Error("查询平台是否存在失败", slog.Uint64("platform_id", uint64(platformID)), slog.Any("error", err))
		return false, fmt.Errorf("查询平台是否存在失败：%w", err)
	}

	return count > 0, nil
}

// CreateEndpoint 创建端点。
func (r *endpointControlQueryRepository) CreateEndpoint(ctx context.Context, endpoint *types.Endpoint) error {
	if endpoint == nil {
		return fmt.Errorf("创建端点失败：端点参数不能为空")
	}

	q := queryFromContextOrDefault(ctx)
	if err := q.Endpoint.WithContext(ctx).Create(endpoint); err != nil {
		r.logger.Error("创建端点失败",
			slog.Uint64("platform_id", uint64(endpoint.PlatformID)),
			slog.Any("error", err))
		return fmt.Errorf("创建端点失败：%w", err)
	}

	return nil
}

// CountDefaultEndpointsByPlatform 统计平台默认端点数量。
func (r *endpointControlQueryRepository) CountDefaultEndpointsByPlatform(ctx context.Context, platformID uint) (int64, error) {
	q := queryFromContextOrDefault(ctx)
	count, err := q.Endpoint.WithContext(ctx).
		Where(q.Endpoint.PlatformID.Eq(platformID), q.Endpoint.IsDefault.Is(true)).
		Count()
	if err != nil {
		r.logger.Error("统计平台默认端点数量失败",
			slog.Uint64("platform_id", uint64(platformID)),
			slog.Any("error", err))
		return 0, fmt.Errorf("统计平台默认端点数量失败：%w", err)
	}

	return count, nil
}
