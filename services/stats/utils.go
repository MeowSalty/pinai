package stats

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

// calculateAvgFirstByteTimeWithPercentile 使用百分位数过滤法计算平均首字时间
// lowerPercentile 和 upperPercentile 分别指定要过滤的下限和上限百分位（0.0-1.0）
func (s *service) calculateAvgFirstByteTimeWithPercentile(ctx context.Context, lowerPercentile, upperPercentile float64) (float64, error) {
	q := query.Q
	r := q.RequestLog

	// 获取过去 24 小时内有效的首字时间数据
	var firstByteTimes []*time.Duration
	err := r.WithContext(ctx).
		Select(r.FirstByteTime).
		Where(r.FirstByteTime.IsNotNull()).
		Where(r.Timestamp.Gte(time.Now().Add(-24 * time.Hour))).
		Scan(&firstByteTimes)
	if err != nil {
		return 0, fmt.Errorf("获取首字时间数据失败：%w", err)
	}

	if len(firstByteTimes) == 0 {
		return 0, nil
	}

	durations := make([]uint64, 0, len(firstByteTimes))
	for _, d := range firstByteTimes {
		if d != nil {
			durations = append(durations, uint64(d.Nanoseconds()))
		}
	}

	if len(durations) == 0 {
		return 0, nil
	}

	// 对持续时间进行排序
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	// 计算百分位索引
	lowerIndex := int(float64(len(durations)) * lowerPercentile)
	upperIndex := int(float64(len(durations)) * upperPercentile)

	// 确保索引在有效范围内
	if lowerIndex < 0 {
		lowerIndex = 0
	}
	if upperIndex >= len(durations) {
		upperIndex = len(durations) - 1
	}

	// 提取过滤后的数据（去除极端值）
	filteredDurations := durations[lowerIndex:upperIndex]

	if len(filteredDurations) == 0 {
		return 0, nil
	}

	// 计算平均值
	var sum uint64
	for _, d := range filteredDurations {
		sum += d
	}

	return float64(sum) / float64(len(filteredDurations)), nil
}
