package provider

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/health"
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
