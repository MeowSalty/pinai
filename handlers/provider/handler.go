package provider

import (
	"github.com/MeowSalty/pinai/services/health"
	"github.com/MeowSalty/pinai/services/provider"
)

// Handler 结构体封装了所有供应方相关的处理函数
type Handler struct {
	service       provider.Service
	healthService health.Service
}

// NewHandler 创建一个新的 ProviderHandler 实例
//
// 参数：
//   - service: provider.Service 服务接口实例
//   - healthService: health.Service 健康服务接口实例
//
// 返回值：
//   - *ProviderHandler: ProviderHandler 实例指针
func NewHandler(service provider.Service, healthService health.Service) *Handler {
	return &Handler{service: service, healthService: healthService}
}
