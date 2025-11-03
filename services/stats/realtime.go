package stats

import (
	"context"
)

// GetRealtime 实现获取实时数据的业务逻辑
//
// 通过采集器获取 API 接口的实时调用数据
func (s *service) GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error) {
	s.logger.DebugContext(ctx, "开始获取实时数据")

	collector := GetCollector()

	// 获取过去 1 分钟的请求数 (RPM)
	rpm := collector.GetRPM()

	// 获取当前活动连接数
	activeConnections := collector.GetActiveConnections()

	s.logger.InfoContext(ctx, "成功获取实时数据",
		"rpm", rpm,
		"active_connections", activeConnections,
	)

	return &StatsRealtimeResponse{
		RPM:               rpm,
		ActiveConnections: activeConnections,
	}, nil
}
