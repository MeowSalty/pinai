package types

import (
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

// HealthStatusResponse 用于表示单个资源的详细健康状态
type HealthStatusResponse struct {
	ID              uint               `json:"id"`
	ResourceType    types.ResourceType `json:"resource_type"`
	ResourceID      uint               `json:"resource_id"`
	ResourceName    string             `json:"resource_name"`
	Status          types.HealthStatus `json:"status"`
	LastError       string             `json:"last_error,omitempty"`
	LastCheckAt     time.Time          `json:"last_check_at"`
	LastSuccessAt   *time.Time         `json:"last_success_at,omitempty"`
	RetryCount      int                `json:"retry_count"`
	NextAvailableAt *time.Time         `json:"next_available_at,omitempty"`
	SuccessCount    int                `json:"success_count"`
	ErrorCount      int                `json:"error_count"`
}

// HealthResourceInfo 包含在概览中显示的资源基本信息
type HealthResourceInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// HealthOverviewStat 表示一种健康状态的统计信息
type HealthOverviewStat struct {
	Status    types.HealthStatus   `json:"status"`
	Count     int                  `json:"count"`
	Resources []HealthResourceInfo `json:"resources"`
}

// HealthOverviewResponse 用于表示一类资源的健康状态概览
type HealthOverviewResponse struct {
	Total int                  `json:"total"`
	Stats []HealthOverviewStat `json:"stats"`
}

// PlatformResourcesHealthResponse 用于表示特定平台下所有资源的健康状态
type PlatformResourcesHealthResponse struct {
	PlatformID   uint                   `json:"platform_id"`
	PlatformName string                 `json:"platform_name"`
	Models       HealthOverviewResponse `json:"models"`
	APIKeys      HealthOverviewResponse `json:"api_keys"`
}
