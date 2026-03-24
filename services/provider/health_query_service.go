package provider

import (
	"context"
	"fmt"

	"github.com/MeowSalty/pinai/database/types"
)

// GetResourceHealthStatus 获取单个资源健康状态；无记录时返回 Unknown。
func (s *service) GetResourceHealthStatus(resourceType types.ResourceType, resourceID uint) (types.HealthStatus, error) {
	if s.healthReader == nil {
		return types.HealthStatusUnknown, fmt.Errorf("获取资源健康状态失败：健康读取器未初始化")
	}

	health, err := s.healthReader.Get(resourceType, resourceID)
	if err != nil {
		return types.HealthStatusUnknown, fmt.Errorf("获取资源健康状态失败：%w", err)
	}
	if health == nil {
		return types.HealthStatusUnknown, nil
	}

	return health.Status, nil
}

// CountResourceHealthByPlatform 获取密钥和模型按平台维度的健康计数。
func (s *service) CountResourceHealthByPlatform(ctx context.Context) (keyCounts, modelCounts map[uint]PlatformStatusCount, err error) {
	if s.healthReader == nil {
		return nil, nil, fmt.Errorf("获取资源健康计数失败：健康读取器未初始化")
	}

	keyMap, modelMap, err := s.GetResourcePlatformMaps(ctx)
	if err != nil {
		return nil, nil, err
	}

	keyCounts = s.healthReader.CountByPlatform(types.ResourceTypeAPIKey, keyMap)
	modelCounts = s.healthReader.CountByPlatform(types.ResourceTypeModel, modelMap)
	return keyCounts, modelCounts, nil
}
