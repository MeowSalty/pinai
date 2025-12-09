package provider

import (
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// service 是 Service 接口的具体实现
type service struct {
	logger *slog.Logger
}

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
