package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

// GetRealtime 实现获取实时数据的业务逻辑
func (s *service) GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error) {
	q := query.Q
	r := q.RequestLog

	// 计算过去 1 分钟的请求数 (RPM)
	oneMinuteAgo := time.Now().Add(-time.Minute)
	recentRequests, err := r.WithContext(ctx).
		Where(r.Timestamp.Gte(oneMinuteAgo)).
		Count()
	if err != nil {
		return nil, fmt.Errorf("计算 RPM 失败：%w", err)
	}

	return &StatsRealtimeResponse{
		RPM: float64(recentRequests),
	}, nil
}