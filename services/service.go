package services

import (
	"context"
	"log/slog"
)

// Services 持有所有服务实例的结构体
type Services struct {
	HealthService    HealthServiceInterface
	AIGatewayService PortalService
	ProviderService  ProviderService
}

// NewServices 初始化所有服务并返回 Services 实例
// 参数：
//
//	ctx - 上下文，用于 AI 网关服务的初始化
//	logger - 日志记录器，用于 AI 网关服务的初始化
//
// 返回值：
//
//	*Services - 包含所有服务实例的结构体
//	error - 初始化过程中可能出现的错误
func NewServices(ctx context.Context, logger *slog.Logger) (*Services, error) {
	// 初始化健康服务
	healthService := NewHealthService()

	// 初始化 AI 网关服务
	aiGatewayService, err := NewPortalService(ctx, logger.WithGroup("portal"))
	if err != nil {
		return nil, err
	}

	// 初始化供应商服务
	providerService := NewProviderService()

	return &Services{
		HealthService:    healthService,
		AIGatewayService: aiGatewayService,
		ProviderService:  providerService,
	}, nil
}
