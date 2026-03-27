package stats

import (
	"context"
	"fmt"
	"time"
)

// GetRealtime 实现获取实时数据的业务逻辑
//
// 通过采集器获取 API 接口的实时调用数据
func (s *service) GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error) {
	start := time.Now()
	logger := s.logger.With("operation", "get_realtime")

	logger.DebugContext(ctx, "开始获取实时数据")

	collector := s.collector
	if collector == nil {
		err := fmt.Errorf("统计采集器未注入，无法获取实时数据")
		logger.ErrorContext(ctx, "获取实时数据失败", "error", err)
		return nil, err
	}

	// 获取过去 1 分钟的请求数 (RPM)
	rpm := collector.GetRPM()

	// 获取当前活动连接数
	activeConnections := collector.GetActiveConnections()

	logger.DebugContext(ctx, "成功获取实时数据",
		"rpm", rpm,
		"active_connections", activeConnections,
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return &StatsRealtimeResponse{
		RPM:               rpm,
		ActiveConnections: activeConnections,
	}, nil
}
