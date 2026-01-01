package services

import (
	"context"
	"log/slog"

	"github.com/MeowSalty/pinai/services/health"
	"github.com/MeowSalty/pinai/services/portal"
	"github.com/MeowSalty/pinai/services/provider"
	"github.com/MeowSalty/pinai/services/stats"
)

// Services 持有所有服务实例的结构体
type Services struct {
	HealthService   health.Service
	HealthStorage   *health.Storage // 健康状态存储，供外部需要时访问
	PortalService   portal.Service
	ProviderService provider.Service
	StatsService    stats.Service
}

// NewServices 初始化所有服务并返回 Services 实例
//
// 该函数负责初始化应用所需的所有服务，并将日志记录器正确传递给各服务。
//
// 参数：
//
//	ctx - 上下文，用于服务的初始化
//	logger - 日志记录器，用于记录服务初始化和运行过程中的日志信息
//	modelMapping - 模型映射规则字符串，格式为 "key1:value1,key2:value2"
//
// 返回值：
//
//	*Services - 包含所有服务实例的结构体
//	error - 初始化过程中可能出现的错误
func NewServices(ctx context.Context, logger *slog.Logger, modelMapping string) (*Services, error) {
	// 初始化健康服务（内部会创建 Storage）
	healthService, err := health.NewService(ctx, logger.WithGroup("health"))
	if err != nil {
		return nil, err
	}

	// 从健康服务获取共享的 Storage 实例
	healthStorage := healthService.GetStorage()

	// 使用共享的 Storage 创建 Portal 服务
	portalService, err := portal.New(ctx, logger.WithGroup("portal"), modelMapping, healthStorage)
	if err != nil {
		return nil, err
	}

	// 初始化供应商服务
	providerService := provider.New(logger.WithGroup("provider"))

	// 初始化统计服务
	statsService := stats.New(logger.WithGroup("stats"))

	return &Services{
		HealthService:   healthService,
		HealthStorage:   healthStorage,
		PortalService:   portalService,
		ProviderService: providerService,
		StatsService:    statsService,
	}, nil
}
