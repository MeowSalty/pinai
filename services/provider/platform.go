package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// CreatePlatform 实现创建平台
func (s *service) CreatePlatform(ctx context.Context, platform types.Platform) (*types.Platform, error) {
	s.logger.Debug("开始创建平台", slog.String("platform_name", platform.Name))

	platform.ID = 0
	if err := query.Q.Platform.WithContext(ctx).Create(&platform); err != nil {
		s.logger.Error("创建平台失败", slog.String("platform_name", platform.Name), slog.Any("error", err))
		return nil, fmt.Errorf("创建平台失败：%w", err)
	}

	s.logger.Info("成功创建平台", slog.String("platform_name", platform.Name), slog.Uint64("platform_id", uint64(platform.ID)))
	return &platform, nil
}

// GetPlatforms 实现获取平台列表
func (s *service) GetPlatforms(ctx context.Context) ([]*types.Platform, error) {
	s.logger.Debug("开始获取平台列表")

	platforms, err := query.Q.Platform.WithContext(ctx).Find()
	if err != nil {
		s.logger.Error("获取平台列表失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取平台列表失败：%w", err)
	}

	s.logger.Info("成功获取平台列表", slog.Int("count", len(platforms)))
	return platforms, nil
}

// GetPlatform 实现获取指定平台详情
func (s *service) GetPlatform(ctx context.Context, id uint) (*types.Platform, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(id)))
	logger.Debug("开始获取平台详情")

	platform, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).First()
	if err != nil {
		logger.Error("获取平台详情失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取 ID 为 %d 的平台失败：%w", id, err)
	}

	logger.Info("成功获取平台详情", slog.String("platform_name", platform.Name))
	return platform, nil
}

// UpdatePlatform 实现更新平台信息
func (s *service) UpdatePlatform(ctx context.Context, id uint, platform types.Platform) (*types.Platform, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(id)))
	logger.Debug("开始更新平台")

	// 只更新非零值字段
	result, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).Updates(platform)
	if err != nil {
		logger.Error("更新平台失败", slog.Any("error", err))
		return nil, fmt.Errorf("更新 ID 为 %d 的平台失败：%w", id, err)
	}
	if result.RowsAffected == 0 {
		logger.Warn("平台不存在")
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台", id)
	}

	// 返回更新后的完整对象
	updatedPlatform, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).First()
	if err != nil {
		logger.Error("获取更新后的平台失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取更新后的 ID 为 %d 的平台失败：%w", id, err)
	}

	logger.Info("成功更新平台", slog.String("platform_name", updatedPlatform.Name))
	return updatedPlatform, nil
}
