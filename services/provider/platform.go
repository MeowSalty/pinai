package provider

import (
	"context"
	"fmt"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// GetPlatforms 实现获取平台列表
func (s *service) GetPlatforms(ctx context.Context) ([]*types.Platform, error) {
	platforms, err := query.Q.Platform.WithContext(ctx).Find()
	if err != nil {
		return nil, fmt.Errorf("获取平台列表失败：%w", err)
	}
	return platforms, nil
}

// GetPlatform 实现获取指定平台详情
func (s *service) GetPlatform(ctx context.Context, id uint) (*types.Platform, error) {
	platform, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).First()
	if err != nil {
		return nil, fmt.Errorf("获取 ID 为 %d 的平台失败：%w", id, err)
	}
	return platform, nil
}

// UpdatePlatform 实现更新平台信息
func (s *service) UpdatePlatform(ctx context.Context, id uint, platform types.Platform) (*types.Platform, error) {
	// 只更新非零值字段
	result, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).Updates(platform)
	if err != nil {
		return nil, fmt.Errorf("更新 ID 为 %d 的平台失败：%w", id, err)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台", id)
	}

	// 返回更新后的完整对象
	updatedPlatform, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).First()
	if err != nil {
		return nil, fmt.Errorf("获取更新后的 ID 为 %d 的平台失败：%w", id, err)
	}
	return updatedPlatform, nil
}
