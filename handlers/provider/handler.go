package provider

import (
	"github.com/MeowSalty/pinai/services/provider"
)

// Handler 结构体封装了所有供应方相关的处理函数
type Handler struct {
	service provider.Service
}

// HealthUpdateRequest 健康状态更新请求
type HealthUpdateRequest struct {
	Enabled *bool `json:"enabled" binding:"required"` // true=启用；false=禁用
}

// NewHandler 创建一个新的 ProviderHandler 实例
//
// 参数：
//   - service: provider.Service 服务接口实例
//
// 返回值：
//   - *ProviderHandler: ProviderHandler 实例指针
func NewHandler(service provider.Service) *Handler {
	return &Handler{service: service}
}
