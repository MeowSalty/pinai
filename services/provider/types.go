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
	APIKey   types.APIKey   `json:"apiKey"`
}
