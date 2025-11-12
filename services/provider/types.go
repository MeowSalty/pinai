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
