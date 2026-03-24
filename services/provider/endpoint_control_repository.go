package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gen/field"
	"gorm.io/gorm"
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

// GetEndpoint 根据 ID 获取端点。
func (r *endpointControlQueryRepository) GetEndpoint(ctx context.Context, endpointID uint) (*types.Endpoint, error) {
	q := queryFromContextOrDefault(ctx)
	endpoint, err := q.Endpoint.WithContext(ctx).Where(q.Endpoint.ID.Eq(endpointID)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的端点：%w", endpointID, ErrResourceNotFound)
		}

		r.logger.Error("查询端点失败",
			slog.Uint64("endpoint_id", uint64(endpointID)),
			slog.Any("error", err))
		return nil, fmt.Errorf("查询端点失败：%w", err)
	}

	return endpoint, nil
}

// UpdateEndpointFields 按字段集合更新端点并返回影响行数。
func (r *endpointControlQueryRepository) UpdateEndpointFields(ctx context.Context, endpointID uint, updates types.Endpoint, fieldNames []string) (int64, error) {
	q := queryFromContextOrDefault(ctx)

	selectExprs := make([]field.Expr, 0, len(fieldNames))
	for _, name := range fieldNames {
		expr, ok := q.Endpoint.GetFieldByName(name)
		if !ok {
			continue
		}
		selectExprs = append(selectExprs, expr)
	}

	result, err := q.Endpoint.WithContext(ctx).
		Select(selectExprs...).
		Where(q.Endpoint.ID.Eq(endpointID)).
		Updates(updates)
	if err != nil {
		r.logger.Error("更新端点失败",
			slog.Uint64("endpoint_id", uint64(endpointID)),
			slog.Any("error", err))
		return 0, fmt.Errorf("更新端点失败：%w", err)
	}

	return result.RowsAffected, nil
}

// DeleteEndpointByID 根据 ID 删除端点。
func (r *endpointControlQueryRepository) DeleteEndpointByID(ctx context.Context, endpointID uint) (int64, error) {
	q := queryFromContextOrDefault(ctx)
	result, err := q.Endpoint.WithContext(ctx).Where(q.Endpoint.ID.Eq(endpointID)).Delete()
	if err != nil {
		r.logger.Error("删除端点失败",
			slog.Uint64("endpoint_id", uint64(endpointID)),
			slog.Any("error", err))
		return 0, fmt.Errorf("删除端点失败：%w", err)
	}

	return result.RowsAffected, nil
}

// GetLatestEndpointByPlatform 获取平台最新创建的端点（按 ID 倒序）。
func (r *endpointControlQueryRepository) GetLatestEndpointByPlatform(ctx context.Context, platformID uint) (*types.Endpoint, error) {
	q := queryFromContextOrDefault(ctx)
	endpoint, err := q.Endpoint.WithContext(ctx).
		Where(q.Endpoint.PlatformID.Eq(platformID)).
		Order(q.Endpoint.ID.Desc()).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		r.logger.Error("查询平台最新端点失败",
			slog.Uint64("platform_id", uint64(platformID)),
			slog.Any("error", err))
		return nil, fmt.Errorf("查询平台最新端点失败：%w", err)
	}

	return endpoint, nil
}

// SetEndpointDefault 设置端点默认状态。
func (r *endpointControlQueryRepository) SetEndpointDefault(ctx context.Context, endpointID uint, isDefault bool) (int64, error) {
	q := queryFromContextOrDefault(ctx)
	result, err := q.Endpoint.WithContext(ctx).
		Where(q.Endpoint.ID.Eq(endpointID)).
		Updates(types.Endpoint{IsDefault: isDefault})
	if err != nil {
		r.logger.Error("更新端点默认状态失败",
			slog.Uint64("endpoint_id", uint64(endpointID)),
			slog.Bool("is_default", isDefault),
			slog.Any("error", err))
		return 0, fmt.Errorf("更新端点默认状态失败：%w", err)
	}

	return result.RowsAffected, nil
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
