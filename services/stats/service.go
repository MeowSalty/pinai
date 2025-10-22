package stats

import (
	"context"
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

// New 创建一个新的统计服务实例
func New() Service {
	// 初始化全局采集器
	InitCollector()
	return &service{}
}

// Service 定义统计服务接口
type Service interface {
	// GetOverview 获取全局概览数据
	GetOverview(ctx context.Context, duration time.Duration) (*StatsOverviewResponse, error)

	// GetRealtime 获取实时数据
	GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error)

	// ListRequestLogs 获取请求状态列表
	ListRequestLogs(ctx context.Context, opts ListRequestLogsOptions) ([]*types.RequestLog, int64, error)

	// GetModelCallRank 获取模型调用排名前 5
	GetModelCallRank(ctx context.Context, duration time.Duration) (*ModelCallRankResponse, error)

	// GetPlatformCallRank 获取平台调用排名前 5
	GetPlatformCallRank(ctx context.Context, duration time.Duration) (*PlatformCallRankResponse, error)

	// GetModelUsageRank 获取模型用量排名前 5
	GetModelUsageRank(ctx context.Context, duration time.Duration) (*ModelUsageRankResponse, error)

	// GetPlatformUsageRank 获取平台用量排名前 5
	GetPlatformUsageRank(ctx context.Context, duration time.Duration) (*PlatformUsageRankResponse, error)
}
