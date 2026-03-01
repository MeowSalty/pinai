package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

// trendRangeConfig 定义趋势范围配置
type trendRangeConfig struct {
	Granularity time.Duration
	Points      int
	Label       string
}

var trendRangeConfigs = map[TrendRange]trendRangeConfig{
	TrendRange24h: {
		Granularity: 4 * time.Hour,
		Points:      7,
		Label:       "4h",
	},
	TrendRange7d: {
		Granularity: 24 * time.Hour,
		Points:      7,
		Label:       "1d",
	},
	TrendRange30d: {
		Granularity: 24 * time.Hour,
		Points:      30,
		Label:       "1d",
	},
}

// GetTrend 获取用量趋势数据
func (s *service) GetTrend(ctx context.Context, trendRange TrendRange) (*TrendResponse, error) {
	if trendRange == "" {
		trendRange = TrendRange24h
	}

	cfg, ok := trendRangeConfigs[trendRange]
	if !ok {
		return nil, fmt.Errorf("无效的时间范围参数，可选值：24h, 7d, 30d")
	}

	now := time.Now().UTC()
	bucketEnd := ceilToHour(now)
	// 数据桶区间采用左闭右开：[bucketStart, bucketEnd)
	// 每个点的时间戳表示该桶的结束时间
	bucketStart := bucketEnd.Add(-cfg.Granularity * time.Duration(cfg.Points))

	s.logger.InfoContext(ctx, "开始获取用量趋势数据",
		"range", trendRange,
		"granularity", cfg.Label,
		"bucket_start", bucketStart,
		"bucket_end", bucketEnd,
	)

	logs, err := query.Q.RequestLog.WithContext(ctx).
		Select(query.Q.RequestLog.Timestamp, query.Q.RequestLog.TotalTokens).
		Where(query.Q.RequestLog.Timestamp.Gte(bucketStart)).
		Where(query.Q.RequestLog.Timestamp.Lt(bucketEnd)).
		Order(query.Q.RequestLog.Timestamp.Asc()).
		Find()
	if err != nil {
		s.logger.ErrorContext(ctx, "查询趋势原始数据失败", "error", err)
		return nil, fmt.Errorf("查询趋势数据失败：%w", err)
	}

	points := make([]TrendDataPoint, cfg.Points)
	for i := 0; i < cfg.Points; i++ {
		points[i] = TrendDataPoint{
			Timestamp: bucketStart.Add(cfg.Granularity * time.Duration(i+1)),
		}
	}

	for _, log := range logs {
		idx := int(log.Timestamp.UTC().Sub(bucketStart) / cfg.Granularity)
		if idx < 0 || idx >= len(points) {
			continue
		}

		points[idx].RequestCount++
		if log.TotalTokens != nil {
			points[idx].TotalTokens += int64(*log.TotalTokens)
		}
	}

	var summary TrendSummary
	for _, p := range points {
		summary.TotalRequests += p.RequestCount
		summary.TotalTokens += p.TotalTokens
	}

	if len(points) > 0 {
		summary.AvgRequestsPerPoint = float64(summary.TotalRequests) / float64(len(points))
		summary.AvgTokensPerPoint = float64(summary.TotalTokens) / float64(len(points))
	}

	resp := &TrendResponse{
		Range:       string(trendRange),
		Granularity: cfg.Label,
		DataPoints:  points,
		Summary:     summary,
	}

	s.logger.InfoContext(ctx, "成功获取用量趋势数据",
		"range", trendRange,
		"points", len(points),
		"total_requests", summary.TotalRequests,
		"total_tokens", summary.TotalTokens,
	)

	return resp, nil
}

// ceilToHour 将时间向上取整到整点（若已是整点则保持不变）
func ceilToHour(t time.Time) time.Time {
	u := t.UTC()
	if u.Minute() == 0 && u.Second() == 0 && u.Nanosecond() == 0 {
		return u
	}
	return u.Truncate(time.Hour).Add(time.Hour)
}
