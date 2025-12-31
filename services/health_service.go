package services

import (
	"context"
	"fmt"

	"github.com/MeowSalty/pinai/database/query"
	dbtypes "github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/handlers/health/types"
	"gorm.io/gorm"
)

// HealthServiceInterface 定义健康服务接口
type HealthServiceInterface interface {
	// GetResourceHealth 获取指定资源的详细健康状态信息
	GetResourceHealth(ctx context.Context, resourceType dbtypes.ResourceType, id uint) (*types.HealthStatusResponse, error)

	// GetPlatformsHealthOverview 获取所有平台的健康概览
	GetPlatformsHealthOverview(ctx context.Context) (*types.HealthOverviewResponse, error)

	// GetModelsHealthOverview 获取所有模型的健康概览
	GetModelsHealthOverview(ctx context.Context) (*types.HealthOverviewResponse, error)

	// GetPlatformResourcesHealth 获取指定平台下所有资源的健康状态
	GetPlatformResourcesHealth(ctx context.Context, platformID uint) (*types.PlatformResourcesHealthResponse, error)
}

// HealthService 健康服务实现
type HealthService struct{}

// NewHealthService 创建健康服务实例
func NewHealthService() HealthServiceInterface {
	return &HealthService{}
}

// GetResourceHealth 获取指定资源的详细健康状态信息
//
// 参数：
//
//	ctx - 上下文
//	resourceType - 资源类型
//	id - 资源 ID
//
// 返回值：
//
//	*types.HealthStatusResponse - 健康状态响应
//	error - 错误信息
func (s *HealthService) GetResourceHealth(ctx context.Context, resourceType dbtypes.ResourceType, id uint) (*types.HealthStatusResponse, error) {
	q := query.Q
	h := q.Health
	healthInfo, err := h.WithContext(ctx).
		Where(h.ResourceType.Eq(int8(resourceType)), h.ResourceID.Eq(id)).
		First()
	if err != nil {
		return nil, fmt.Errorf("查询健康信息失败：%w", err)
	}

	resourceName, err := s.getResourceName(ctx, resourceType, id)
	if err != nil {
		return nil, fmt.Errorf("获取资源名称失败：%w", err)
	}

	return &types.HealthStatusResponse{
		ResourceType:    healthInfo.ResourceType,
		ResourceID:      healthInfo.ResourceID,
		ResourceName:    resourceName,
		Status:          healthInfo.Status,
		LastError:       healthInfo.LastError,
		LastCheckAt:     healthInfo.LastCheckAt,
		LastSuccessAt:   healthInfo.LastSuccessAt,
		RetryCount:      healthInfo.RetryCount,
		NextAvailableAt: healthInfo.NextAvailableAt,
		SuccessCount:    healthInfo.SuccessCount,
		ErrorCount:      healthInfo.ErrorCount,
	}, nil
}

// getResourceName 获取资源名称
//
// 参数：
//
//	ctx - 上下文
//	resourceType - 资源类型
//	id - 资源 ID
//
// 返回值：
//
//	string - 资源名称
//	error - 错误信息
func (s *HealthService) getResourceName(ctx context.Context, resourceType dbtypes.ResourceType, id uint) (string, error) {
	q := query.Q

	switch resourceType {
	case dbtypes.ResourceTypePlatform:
		p := q.Platform
		platform, err := p.WithContext(ctx).Where(p.ID.Eq(id)).First()
		if err != nil {
			return "", fmt.Errorf("查询平台信息失败：%w", err)
		}
		return platform.Name, nil
	case dbtypes.ResourceTypeModel:
		m := q.Model
		model, err := m.WithContext(ctx).Where(m.ID.Eq(id)).First()
		if err != nil {
			return "", fmt.Errorf("查询模型信息失败：%w", err)
		}
		return model.Name, nil
	case dbtypes.ResourceTypeAPIKey:
		a := q.APIKey
		apiKey, err := a.WithContext(ctx).Where(a.ID.Eq(id)).First()
		if err != nil {
			return "", fmt.Errorf("查询 API 密钥信息失败：%w", err)
		}
		if len(apiKey.Value) > 8 {
			return apiKey.Value[:8] + "...", nil
		}
		return apiKey.Value, nil
	default:
		return "", fmt.Errorf("不支持的资源类型：%d", resourceType)
	}
}

// getHealthOverview 获取指定资源类型的健康概览
//
// 参数：
//
//	ctx - 上下文
//	resourceType - 资源类型
//
// 返回值：
//
//	*types.HealthOverviewResponse - 健康概览响应
//	error - 错误信息
func (s *HealthService) getHealthOverview(ctx context.Context, resourceType dbtypes.ResourceType) (*types.HealthOverviewResponse, error) {
	q := query.Q
	h := q.Health
	var stats []struct {
		Status dbtypes.HealthStatus
		Count  int
	}
	err := h.WithContext(ctx).
		Select(h.Status, h.Status.Count().As("count")).
		Where(h.ResourceType.Eq(int8(resourceType))).
		Group(h.Status).
		Scan(&stats)
	if err != nil {
		return nil, fmt.Errorf("查询健康统计信息失败：%w", err)
	}

	overview := &types.HealthOverviewResponse{
		Stats: make([]types.HealthOverviewStat, 0, len(stats)),
	}

	for _, stat := range stats {
		overview.Total += stat.Count
		resources, err := s.getHealthResources(ctx, resourceType, stat.Status)
		if err != nil {
			return nil, fmt.Errorf("获取健康资源列表失败：%w", err)
		}

		overview.Stats = append(overview.Stats, types.HealthOverviewStat{
			Status:    stat.Status,
			Count:     stat.Count,
			Resources: resources,
		})
	}

	return overview, nil
}

// getHealthResources 获取指定资源类型和健康状态的资源列表
//
// 参数：
//
//	ctx - 上下文
//	resourceType - 资源类型
//	status - 健康状态
//
// 返回值：
//
//	[]types.HealthResourceInfo - 资源信息列表
//	error - 错误信息
func (s *HealthService) getHealthResources(ctx context.Context, resourceType dbtypes.ResourceType, status dbtypes.HealthStatus) ([]types.HealthResourceInfo, error) {
	q := query.Q
	h := q.Health

	switch resourceType {
	case dbtypes.ResourceTypePlatform:
		p := q.Platform
		var resources []types.HealthResourceInfo
		err := h.WithContext(ctx).
			Select(p.ID, p.Name).
			LeftJoin(p, p.ID.EqCol(h.ResourceID)).
			Where(h.ResourceType.Eq(int8(resourceType)), h.Status.Eq(int8(status))).
			Scan(&resources)
		return resources, err
	case dbtypes.ResourceTypeModel:
		m := q.Model
		var resources []types.HealthResourceInfo
		err := h.WithContext(ctx).
			Select(m.ID, m.Name).
			LeftJoin(m, m.ID.EqCol(h.ResourceID)).
			Where(h.ResourceType.Eq(int8(resourceType)), h.Status.Eq(int8(status))).
			Scan(&resources)
		return resources, err
	case dbtypes.ResourceTypeAPIKey:
		a := q.APIKey
		type apiKeyResult struct {
			ID    uint
			Value string
		}
		var apiKeys []apiKeyResult
		var resources []types.HealthResourceInfo
		err := h.WithContext(ctx).
			Select(a.ID, a.Value).
			LeftJoin(a, a.ID.EqCol(h.ResourceID)).
			Where(h.ResourceType.Eq(int8(resourceType)), h.Status.Eq(int8(status))).
			Scan(&apiKeys)
		if err != nil {
			return nil, err
		}

		for _, key := range apiKeys {
			name := key.Value
			if len(name) > 8 {
				name = name[:8] + "..."
			}
			resources = append(resources, types.HealthResourceInfo{ID: key.ID, Name: name})
		}
		return resources, nil
	default:
		return nil, fmt.Errorf("不支持的资源类型：%d", resourceType)
	}
}

// GetPlatformsHealthOverview 获取所有平台的健康概览
//
// 参数：
//
//	ctx - 上下文
//
// 返回值：
//
//	*types.HealthOverviewResponse - 健康概览响应
//	error - 错误信息
func (s *HealthService) GetPlatformsHealthOverview(ctx context.Context) (*types.HealthOverviewResponse, error) {
	return s.getHealthOverview(ctx, dbtypes.ResourceTypePlatform)
}

// GetModelsHealthOverview 获取所有模型的健康概览
//
// 参数：
//
//	ctx - 上下文
//
// 返回值：
//
//	*types.HealthOverviewResponse - 健康概览响应
//	error - 错误信息
func (s *HealthService) GetModelsHealthOverview(ctx context.Context) (*types.HealthOverviewResponse, error) {
	return s.getHealthOverview(ctx, dbtypes.ResourceTypeModel)
}

// GetPlatformResourcesHealth 获取指定平台下所有资源的健康状态
//
// 参数：
//
//	ctx - 上下文
//	platformID - 平台 ID
//
// 返回值：
//
//	*types.PlatformResourcesHealthResponse - 平台资源健康响应
//	error - 错误信息
func (s *HealthService) GetPlatformResourcesHealth(ctx context.Context, platformID uint) (*types.PlatformResourcesHealthResponse, error) {
	q := query.Q
	p := q.Platform
	platform, err := p.WithContext(ctx).Where(p.ID.Eq(platformID)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("平台不存在：%w", err)
		}
		return nil, fmt.Errorf("查询平台信息失败：%w", err)
	}

	modelsHealth, err := s.getHealthOverviewForPlatform(ctx, platformID, dbtypes.ResourceTypeModel)
	if err != nil {
		return nil, fmt.Errorf("获取模型健康信息失败：%w", err)
	}

	apiKeysHealth, err := s.getHealthOverviewForPlatform(ctx, platformID, dbtypes.ResourceTypeAPIKey)
	if err != nil {
		return nil, fmt.Errorf("获取 API 密钥健康信息失败：%w", err)
	}

	return &types.PlatformResourcesHealthResponse{
		PlatformID:   platform.ID,
		PlatformName: platform.Name,
		Models:       *modelsHealth,
		APIKeys:      *apiKeysHealth,
	}, nil
}

// getHealthOverviewForPlatform 获取指定平台下特定资源类型的健康概览
//
// 参数：
//
//	ctx - 上下文
//	platformID - 平台 ID
//	resourceType - 资源类型
//
// 返回值：
//
//	*types.HealthOverviewResponse - 健康概览响应
//	error - 错误信息
func (s *HealthService) getHealthOverviewForPlatform(ctx context.Context, platformID uint, resourceType dbtypes.ResourceType) (*types.HealthOverviewResponse, error) {
	overview := &types.HealthOverviewResponse{
		Stats: make([]types.HealthOverviewStat, 0),
	}

	stats, err := s.getPlatformResourceStats(ctx, platformID, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取平台资源统计信息失败：%w", err)
	}

	for _, stat := range stats {
		overview.Total += stat.Count
		resources, err := s.getPlatformResources(ctx, platformID, resourceType, stat.Status)
		if err != nil {
			return nil, fmt.Errorf("获取平台资源列表失败：%w", err)
		}

		overview.Stats = append(overview.Stats, types.HealthOverviewStat{
			Status:    stat.Status,
			Count:     stat.Count,
			Resources: resources,
		})
	}

	return overview, nil
}

// getPlatformResourceStats 获取平台下特定资源类型的统计信息
//
// 参数：
//
//	ctx - 上下文
//	platformID - 平台 ID
//	resourceType - 资源类型
//
// 返回值：
//
//	[]struct{Status dbtypes.HealthStatus; Count int} - 统计信息
//	error - 错误信息
func (s *HealthService) getPlatformResourceStats(ctx context.Context, platformID uint, resourceType dbtypes.ResourceType) ([]struct {
	Status dbtypes.HealthStatus
	Count  int
}, error) {
	q := query.Q

	var stats []struct {
		Status dbtypes.HealthStatus
		Count  int
	}

	switch resourceType {
	case dbtypes.ResourceTypeModel:
		h := q.Health
		m := q.Model
		err := h.WithContext(ctx).
			Select(h.Status, h.Status.Count().As("count")).
			Join(m, m.ID.EqCol(h.ResourceID)).
			Where(h.ResourceType.Eq(int8(resourceType)), m.PlatformID.Eq(platformID)).
			Group(h.Status).
			Scan(&stats)
		return stats, err
	case dbtypes.ResourceTypeAPIKey:
		h := q.Health
		a := q.APIKey
		err := h.WithContext(ctx).
			Select(h.Status, h.Status.Count().As("count")).
			Join(a, a.ID.EqCol(h.ResourceID)).
			Where(h.ResourceType.Eq(int8(resourceType)), a.PlatformID.Eq(platformID)).
			Group(h.Status).
			Scan(&stats)
		return stats, err
	default:
		return nil, fmt.Errorf("不支持的资源类型：%d", resourceType)
	}
}

// getPlatformResources 获取平台下特定资源类型和健康状态的资源列表
//
// 参数：
//
//	ctx - 上下文
//	platformID - 平台 ID
//	resourceType - 资源类型
//	status - 健康状态
//
// 返回值：
//
//	[]types.HealthResourceInfo - 资源信息列表
//	error - 错误信息
func (s *HealthService) getPlatformResources(ctx context.Context, platformID uint, resourceType dbtypes.ResourceType, status dbtypes.HealthStatus) ([]types.HealthResourceInfo, error) {
	q := query.Q
	h := q.Health

	switch resourceType {
	case dbtypes.ResourceTypeModel:
		m := q.Model
		var resources []types.HealthResourceInfo
		err := h.WithContext(ctx).
			Select(m.ID, m.Name).
			Join(m, m.ID.EqCol(h.ResourceID)).
			Where(h.ResourceType.Eq(int8(resourceType)), m.PlatformID.Eq(platformID), h.Status.Eq(int8(status))).
			Scan(&resources)
		return resources, err
	case dbtypes.ResourceTypeAPIKey:
		a := q.APIKey
		type apiKeyResult struct {
			ID    uint
			Value string
		}
		var apiKeys []apiKeyResult
		var resources []types.HealthResourceInfo
		err := h.WithContext(ctx).
			Select(a.ID, a.Value).
			Join(a, a.ID.EqCol(h.ResourceID)).
			Where(h.ResourceType.Eq(int8(resourceType)), a.PlatformID.Eq(platformID), h.Status.Eq(int8(status))).
			Scan(&apiKeys)
		if err != nil {
			return nil, err
		}

		for _, key := range apiKeys {
			name := key.Value
			if len(name) > 8 {
				name = name[:8] + "..."
			}
			resources = append(resources, types.HealthResourceInfo{ID: key.ID, Name: name})
		}
		return resources, nil
	default:
		return nil, fmt.Errorf("不支持的资源类型：%d", resourceType)
	}
}
