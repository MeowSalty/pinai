package portal

import (
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/health"
	coreHealth "github.com/MeowSalty/portal/routing/health"
)

// healthStorageAdapter 适配器，将内部 health.Storage 转换为 portal 需要的 health.Storage 接口
type healthStorageAdapter struct {
	storage *health.Storage
}

// Get 实现 portal health.Storage 接口的 Get 方法
func (a *healthStorageAdapter) Get(resourceType coreHealth.ResourceType, resourceID uint) (*coreHealth.Health, error) {
	// 将 portal 库的 ResourceType 转换为内部 health 包的 ResourceType
	internalResourceType := convertResourceTypeToInternal(resourceType)
	internalResult, err := a.storage.Get(internalResourceType, resourceID)
	if err != nil || internalResult == nil {
		return nil, err
	}

	// 将内部 health.Health 转换为 portal 库的 health.Health
	return &coreHealth.Health{
		ResourceType:    resourceType,
		ResourceID:      internalResult.ResourceID,
		Status:          coreHealth.HealthStatus(internalResult.Status),
		RetryCount:      internalResult.RetryCount,
		NextAvailableAt: internalResult.NextAvailableAt,
		BackoffDuration: internalResult.BackoffDuration,
		LastError:       internalResult.LastError,
		LastErrorCode:   internalResult.LastErrorCode,
		LastCheckAt:     internalResult.LastCheckAt,
		LastSuccessAt:   internalResult.LastSuccessAt,
		SuccessCount:    internalResult.SuccessCount,
		ErrorCount:      internalResult.ErrorCount,
		CreatedAt:       internalResult.CreatedAt,
		UpdatedAt:       internalResult.UpdatedAt,
	}, nil
}

// Set 实现 portal health.Storage 接口的 Set 方法
func (a *healthStorageAdapter) Set(status *coreHealth.Health) error {
	// 将 portal 库的 health.Health 转换为内部 health 包的 health.Health 类型
	internalStatus := &types.Health{
		ResourceType:    convertResourceTypeToInternal(status.ResourceType),
		ResourceID:      status.ResourceID,
		Status:          types.HealthStatus(status.Status),
		RetryCount:      status.RetryCount,
		NextAvailableAt: status.NextAvailableAt,
		BackoffDuration: status.BackoffDuration,
		LastError:       status.LastError,
		LastErrorCode:   status.LastErrorCode,
		LastCheckAt:     status.LastCheckAt,
		LastSuccessAt:   status.LastSuccessAt,
		SuccessCount:    status.SuccessCount,
		ErrorCount:      status.ErrorCount,
		CreatedAt:       status.CreatedAt,
		UpdatedAt:       status.UpdatedAt,
	}

	return a.storage.Set(internalStatus)
}

// Delete 实现 portal health.Storage 接口的 Delete 方法
func (a *healthStorageAdapter) Delete(resourceType coreHealth.ResourceType, resourceID uint) error {
	// 将 portal 库的 ResourceType 转换为内部 health 包的 ResourceType
	internalResourceType := convertResourceTypeToInternal(resourceType)
	return a.storage.Delete(internalResourceType, resourceID)
}

// convertResourceTypeToInternal 将 portal 库的 ResourceType 转换为内部 health 包的 ResourceType
func convertResourceTypeToInternal(portalType coreHealth.ResourceType) types.ResourceType {
	// 直接类型转换，因为它们应该有相同的值定义
	return types.ResourceType(portalType)
}
