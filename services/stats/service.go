package stats

import (
	"context"
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

// New 创建一个新的统计服务实例
func New() Service {
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

	// GetModelRank 获取模型排名前 10
	GetModelRank(ctx context.Context, duration time.Duration) (*ModelRankResponse, error)

	// GetPlatformRank 获取平台排名前 10
	GetPlatformRank(ctx context.Context, duration time.Duration) (*PlatformRankResponse, error)
}
