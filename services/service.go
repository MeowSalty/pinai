package services

import (
	"context"
	"log/slog"

	"github.com/MeowSalty/pinai/services/provider"
)

// Services 持有所有服务实例的结构体
type Services struct {
	HealthService    HealthServiceInterface
	AIGatewayService PortalService
	ProviderService  provider.Service
	StatsService     StatsServiceInterface
}

// NewServices 初始化所有服务并返回 Services 实例
//
// 该函数负责初始化应用所需的所有服务，并将日志记录器正确传递给各服务。
//
// 参数：
//
//	ctx - 上下文，用于服务的初始化
//	logger - 日志记录器，用于记录服务初始化和运行过程中的日志信息
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
	providerService := provider.New()

	// 初始化统计服务
	statsService := NewStatsService()

	return &Services{
		HealthService:    healthService,
		AIGatewayService: aiGatewayService,
		ProviderService:  providerService,
		StatsService:     statsService,
	}, nil
}
