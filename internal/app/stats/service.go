package stats

import (
	"context"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

// New 创建一个新的统计服务实例
//
// 参数：
//   - logger: 日志记录器，用于记录服务运行状态和关键操作
//
// 返回值：
//   - Service: 统计服务实例
func New(logger *slog.Logger) Service {
	collector := NewCollector(logger.WithGroup("collector"))

	return NewWithCollector(logger, collector)
}

// NewWithCollector 创建一个使用显式采集器依赖的统计服务实例。
//
// 参数：
//   - logger: 日志记录器，用于记录服务运行状态和关键操作
//   - collector: 实时数据采集器；为 nil 时实时统计能力不可用
//
// 返回值：
//   - Service: 统计服务实例
func NewWithCollector(logger *slog.Logger, collector *Collector) Service {
	if collector == nil {
		logger.Warn("未显式提供采集器，实时统计能力不可用")
	}

	logger.Info("统计服务初始化完成")

	return &service{
		logger:    logger,
		collector: collector,
	}
}

// Service 定义统计服务接口
type Service interface {
	// GetDashboard 获取仪表盘所有数据（单次查询优化版本）
	GetDashboard(ctx context.Context, trendRange TrendRange) (*DashboardResponse, error)

	// GetModelStatus 获取模型状态监控数据
	GetModelStatus(ctx context.Context, trendRange TrendRange, modelName *string) (*ModelStatusResponse, error)

	// GetOverview 获取全局概览数据
	//
	// Deprecated: 请改用 GetDashboard 获取统一仪表盘数据。
	GetOverview(ctx context.Context, duration time.Duration) (*StatsOverviewResponse, error)

	// GetRealtime 获取实时数据
	GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error)

	// ListRequestLogs 获取请求状态列表
	ListRequestLogs(ctx context.Context, opts ListRequestLogsOptions) ([]*types.RequestLog, int64, error)

	// GetModelCallRank 获取模型调用排名前 5
	//
	// Deprecated: 请改用 GetDashboard 获取统一仪表盘数据。
	GetModelCallRank(ctx context.Context, duration time.Duration) (*ModelCallRankResponse, error)

	// GetPlatformCallRank 获取平台调用排名前 5
	//
	// Deprecated: 请改用 GetDashboard 获取统一仪表盘数据。
	GetPlatformCallRank(ctx context.Context, duration time.Duration) (*PlatformCallRankResponse, error)

	// GetModelUsageRank 获取模型用量排名前 5
	//
	// Deprecated: 请改用 GetDashboard 获取统一仪表盘数据。
	GetModelUsageRank(ctx context.Context, duration time.Duration) (*ModelUsageRankResponse, error)

	// GetPlatformUsageRank 获取平台用量排名前 5
	//
	// Deprecated: 请改用 GetDashboard 获取统一仪表盘数据。
	GetPlatformUsageRank(ctx context.Context, duration time.Duration) (*PlatformUsageRankResponse, error)
}
