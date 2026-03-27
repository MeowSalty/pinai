package bootstrap

import (
	"context"
	"log/slog"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	"github.com/MeowSalty/pinai/internal/app/health"
	"github.com/MeowSalty/pinai/internal/app/provider"
	"github.com/MeowSalty/pinai/internal/infra/portal"
	"github.com/MeowSalty/pinai/internal/app/stats"
)

// Services 持有启动阶段装配得到的服务实例。
type Services struct {
	HealthService   health.Service
	GatewayService  gateway.Service
	ProviderService provider.Service
	StatsService    stats.Service
	StatsCollector  *stats.Collector
}

// NewServices 初始化应用所需服务并返回聚合结果。
func NewServices(ctx context.Context, logger *slog.Logger, modelMapping string) (*Services, error) {
	// 初始化共享健康存储
	healthStorage, err := health.NewStorage(ctx, logger.WithGroup("health_storage"))
	if err != nil {
		return nil, err
	}

	// 基于共享存储初始化健康服务
	healthService, err := health.NewService(healthStorage, logger.WithGroup("health"))
	if err != nil {
		return nil, err
	}

	// 使用共享的 Storage 创建 Portal 服务
	portalService, err := portal.New(ctx, logger.WithGroup("portal"), modelMapping, healthStorage)
	if err != nil {
		return nil, err
	}

	// 初始化网关应用服务
	gatewayService := gateway.New(portalService, logger.WithGroup("gateway_app"))

	// 初始化供应商服务
	providerService := provider.New(logger.WithGroup("provider"), healthStorage)

	// 初始化统计服务（主路径：装配阶段显式创建并注入采集器）
	statsLogger := logger.WithGroup("stats")
	statsCollector := stats.NewCollector(statsLogger.WithGroup("collector"))
	// 保留全局采集器兼容层，保障仍依赖全局入口的历史调用点行为不变。
	stats.SetGlobalCollector(statsCollector)
	statsService := stats.NewWithCollector(statsLogger, statsCollector)

	return &Services{
		HealthService:   healthService,
		GatewayService:  gatewayService,
		ProviderService: providerService,
		StatsService:    statsService,
		StatsCollector:  statsCollector,
	}, nil
}
