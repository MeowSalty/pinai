package provider

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/internal/app/health"
)

// service 是 Service 接口的具体实现
type service struct {
	logger              *slog.Logger
	healthReader        HealthReader
	platformControlRepo PlatformControlRepository
	modelControlRepo    ModelControlRepository
	keyControlRepo      KeyControlRepository
	endpointControlRepo EndpointControlRepository
	modelBatchTaskRepo  ModelBatchTaskRepository
	controlTx           ControlTx
	controlAudit        ControlAuditLogger

	workerMu         sync.Mutex
	workerCancel     context.CancelFunc
	workerDone       chan struct{}
	workerRunning    bool
	workerPollSecond int

	taskQueue         chan uint
	taskQueueSize     int
	taskStateMu       sync.RWMutex
	taskStateCache    map[uint]*ModelBatchTaskSummary
	taskEnqueued      map[uint]struct{}
	workerRecoverOnce sync.Once
}

// PlatformStatusCount 平台维度健康状态统计。
type PlatformStatusCount = health.StatusCount

// CreateRequest 定义了创建供应方的请求体
type CreateRequest struct {
	Platform types.Platform `json:"platform"`
	Models   []types.Model  `json:"models"`
	APIKeys  []types.APIKey `json:"apiKey"`
}

// BatchCreateModelsRequest 批量创建模型的请求体
type BatchCreateModelsRequest struct {
	Models []types.Model `json:"models" binding:"required,min=1,dive"`
}

// BatchCreateModelsResponse 批量创建模型的响应体
type BatchCreateModelsResponse struct {
	Models       []*types.Model `json:"models"`        // 创建成功的模型列表
	TotalCount   int            `json:"total_count"`   // 请求的模型总数
	CreatedCount int            `json:"created_count"` // 实际创建的模型数
}

// ModelUpdateItem 单个模型的更新项
type ModelUpdateItem struct {
	ID      uint           `json:"id" binding:"required"` // 必需：要更新的模型 ID
	Name    string         `json:"name,omitempty"`        // 可选：模型名称
	Alias   string         `json:"alias,omitempty"`       // 可选：模型别名
	APIKeys []types.APIKey `json:"api_keys,omitempty"`    // 可选：关联的 API 密钥
}

// BatchUpdateModelsRequest 批量更新模型的请求体
type BatchUpdateModelsRequest struct {
	Models []ModelUpdateItem `json:"models" binding:"required,min=1,dive"`
}

// BatchUpdateModelsResponse 批量更新模型的响应体
type BatchUpdateModelsResponse struct {
	Models       []*types.Model `json:"models"`        // 更新成功的模型列表
	TotalCount   int            `json:"total_count"`   // 请求的模型总数
	UpdatedCount int            `json:"updated_count"` // 实际更新的模型数
}

// BatchDeleteModelsRequest 批量删除模型的请求体
type BatchDeleteModelsRequest struct {
	ModelIDs []uint `json:"model_ids" binding:"required,min=1"` // 要删除的模型 ID 列表
}

// BatchDeleteModelsResponse 批量删除模型的响应体
type BatchDeleteModelsResponse struct {
	TotalCount   int `json:"total_count"`   // 请求删除的模型总数
	DeletedCount int `json:"deleted_count"` // 实际删除的模型数
}

// BatchTaskAcceptedResponse 表示异步任务已接受响应。
type BatchTaskAcceptedResponse struct {
	TaskID uint   `json:"task_id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// BatchTaskResult 表示任务结果详情。
type BatchTaskResult struct {
	TotalCount   int `json:"total_count"`
	CreatedCount int `json:"created_count,omitempty"`
	UpdatedCount int `json:"updated_count,omitempty"`
	DeletedCount int `json:"deleted_count,omitempty"`
}

// ModelBatchTaskSummary 表示模型批量任务查询结果。
type ModelBatchTaskSummary struct {
	ID           uint            `json:"id"`
	Type         string          `json:"type"`
	Status       string          `json:"status"`
	PlatformID   uint            `json:"platform_id"`
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
	StartedAt    *string         `json:"started_at,omitempty"`
	FinishedAt   *string         `json:"finished_at,omitempty"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

// BatchCreateEndpointsRequest 批量创建端点的请求体
type BatchCreateEndpointsRequest struct {
	Endpoints []types.Endpoint `json:"endpoints" binding:"required,min=1,dive"`
}

// BatchCreateEndpointsResponse 批量创建端点的响应体
type BatchCreateEndpointsResponse struct {
	Endpoints    []*types.Endpoint `json:"endpoints"`
	TotalCount   int               `json:"total_count"`
	CreatedCount int               `json:"created_count"`
}

// EndpointUpdateItem 单个端点的更新项
type EndpointUpdateItem struct {
	ID              uint               `json:"id" binding:"required"`
	EndpointType    *string            `json:"endpoint_type,omitempty"`
	EndpointVariant *string            `json:"endpoint_variant,omitempty"`
	Path            *string            `json:"path,omitempty"`
	CustomHeaders   *map[string]string `json:"custom_headers,omitempty"`
	IsDefault       *bool              `json:"is_default,omitempty"`
}

// BatchUpdateEndpointsRequest 批量更新端点的请求体
type BatchUpdateEndpointsRequest struct {
	Endpoints []EndpointUpdateItem `json:"endpoints" binding:"required,min=1,dive"`
}

// BatchUpdateEndpointsResponse 批量更新端点的响应体
type BatchUpdateEndpointsResponse struct {
	Endpoints    []*types.Endpoint `json:"endpoints"`
	TotalCount   int               `json:"total_count"`
	UpdatedCount int               `json:"updated_count"`
}
