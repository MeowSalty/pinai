package stats

import (
	"context"
)

// GetRealtime 实现获取实时数据的业务逻辑
//
// 通过采集器获取 API 接口的实时调用数据
func (s *service) GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error) {
	collector := GetCollector()

	// 获取过去 1 分钟的请求数 (RPM)
	rpm := collector.GetRPM()

	// 获取当前活动连接数
	activeConnections := collector.GetActiveConnections()

	return &StatsRealtimeResponse{
		RPM:               rpm,
		ActiveConnections: activeConnections,
	}, nil
}
