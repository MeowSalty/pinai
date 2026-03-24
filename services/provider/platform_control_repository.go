package provider

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/health"
	"gorm.io/gorm"
)

// platformControlQueryRepository 是基于 database/query 的平台控制面仓储实现。
type platformControlQueryRepository struct {
	healthStorage *health.Storage
	logger        *slog.Logger
}

// NewPlatformControlQueryRepository 创建平台控制面仓储实现。
func NewPlatformControlQueryRepository(healthStorage *health.Storage, logger *slog.Logger) PlatformControlRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &platformControlQueryRepository{
		healthStorage: healthStorage,
		logger:        logger.WithGroup("platform_control_repo"),
	}
}

// ExistsPlatform 检查平台是否存在。
func (r *platformControlQueryRepository) ExistsPlatform(ctx context.Context, platformID uint) (bool, error) {
	count, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(platformID)).Count()
	if err != nil {
		r.logger.Error("查询平台是否存在失败", slog.Uint64("platform_id", uint64(platformID)), slog.Any("error", err))
		return false, fmt.Errorf("查询平台是否存在失败：%w", err)
	}

	return count > 0, nil
}

// CreatePlatform 创建平台。
func (r *platformControlQueryRepository) CreatePlatform(ctx context.Context, platform *types.Platform) error {
	if platform == nil {
		return fmt.Errorf("创建平台失败：平台参数不能为空")
	}

	if err := query.Q.Platform.WithContext(ctx).Create(platform); err != nil {
		r.logger.Error("创建平台失败", slog.String("platform_name", platform.Name), slog.Any("error", err))
		return fmt.Errorf("创建平台失败：%w", err)
	}

	return nil
}

// UpdatePlatform 更新平台信息并返回影响行数。
func (r *platformControlQueryRepository) UpdatePlatform(ctx context.Context, platformID uint, updates types.Platform) (int64, error) {
	result, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(platformID)).Updates(updates)
	if err != nil {
		r.logger.Error("更新平台失败", slog.Uint64("platform_id", uint64(platformID)), slog.Any("error", err))
		return 0, fmt.Errorf("更新平台失败：%w", err)
	}

	return result.RowsAffected, nil
}

// GetPlatform 获取平台详情（含端点信息）。
func (r *platformControlQueryRepository) GetPlatform(ctx context.Context, platformID uint) (*types.Platform, error) {
	platform, err := query.Q.Platform.WithContext(ctx).
		Preload(query.Q.Platform.Endpoints).
		Where(query.Q.Platform.ID.Eq(platformID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformID, ErrResourceNotFound)
		}
		r.logger.Error("查询平台失败", slog.Uint64("platform_id", uint64(platformID)), slog.Any("error", err))
		return nil, fmt.Errorf("查询平台失败：%w", err)
	}

	return platform, nil
}

// EnablePlatformHealth 启用平台健康状态（删除健康记录，恢复为 Unknown）。
func (r *platformControlQueryRepository) EnablePlatformHealth(ctx context.Context, platformID uint) error {
	if r.healthStorage == nil {
		return fmt.Errorf("启用平台健康状态失败：健康状态存储未初始化")
	}

	if err := r.healthStorage.Delete(types.ResourceTypePlatform, platformID); err != nil {
		r.logger.Error("启用平台健康状态失败", slog.Uint64("platform_id", uint64(platformID)), slog.Any("error", err))
		return fmt.Errorf("启用平台健康状态失败：%w", err)
	}

	return nil
}

// DisablePlatformHealth 禁用平台健康状态（写入 Unavailable 状态）。
func (r *platformControlQueryRepository) DisablePlatformHealth(ctx context.Context, platformID uint) error {
	if r.healthStorage == nil {
		return fmt.Errorf("禁用平台健康状态失败：健康状态存储未初始化")
	}

	now := time.Now()
	healthRecord := &types.Health{
		ResourceType:    types.ResourceTypePlatform,
		ResourceID:      platformID,
		Status:          types.HealthStatusUnavailable,
		LastError:       "手动禁用",
		LastCheckAt:     now,
		RetryCount:      0,
		BackoffDuration: 0,
	}

	if err := r.healthStorage.Set(healthRecord); err != nil {
		r.logger.Error("禁用平台健康状态失败", slog.Uint64("platform_id", uint64(platformID)), slog.Any("error", err))
		return fmt.Errorf("禁用平台健康状态失败：%w", err)
	}

	return nil
}
