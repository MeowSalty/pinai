package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

// GetOverview 实现获取全局概览数据的业务逻辑
func (s *service) GetOverview(ctx context.Context, duration time.Duration) (*StatsOverviewResponse, error) {
	q := query.Q
	r := q.RequestLog

	// 设置默认时间范围为 24 小时
	if duration == 0 {
		duration = 24 * time.Hour
	}

	startTime := time.Now().Add(-duration)

	// 获取总请求数
	totalRequests, err := r.WithContext(ctx).
		Where(r.Timestamp.Gte(startTime)).
		Count()
	if err != nil {
		return nil, fmt.Errorf("获取总请求数失败：%w", err)
	}

	// 获取成功请求数
	successRequests, err := r.WithContext(ctx).
		Where(r.Success.Is(true)).
		Where(r.Timestamp.Gte(startTime)).
		Count()
	if err != nil {
		return nil, fmt.Errorf("获取成功请求数失败：%w", err)
	}

	// 计算成功率
	var successRate float64
	if totalRequests > 0 {
		successRate = float64(successRequests) / float64(totalRequests)
	}

	// 使用百分位数过滤法计算平均首字时间
	avgFirstByteTime, err := s.calculateAvgFirstByteTimeWithPercentile(ctx, 0.1, 0.9)
	if err != nil {
		return nil, fmt.Errorf("计算平均首字时间失败：%w", err)
	}

	return &StatsOverviewResponse{
		TotalRequests:    totalRequests,
		SuccessRate:      successRate,
		AvgFirstByteTime: avgFirstByteTime,
	}, nil
}
