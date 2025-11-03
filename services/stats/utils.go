package stats

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

// calculateAvgFirstByteTimeWithPercentile 使用百分位数过滤法计算平均首字时间
//
// 该方法通过百分位数过滤异常值（过快或过慢的请求），提供更准确的平均首字时间。
// 例如：lowerPercentile=0.1, upperPercentile=0.9 会过滤掉最快的 10% 和最慢的 10%。
//
// 参数：
//   - ctx: 上下文
//   - lowerPercentile: 下限百分位 (0.0-1.0)，用于过滤异常低值
//   - upperPercentile: 上限百分位 (0.0-1.0)，用于过滤异常高值
//
// 返回：
//   - float64: 过滤后的平均首字时间（纳秒）
//   - error: 如果查询失败则返回错误
func (s *service) calculateAvgFirstByteTimeWithPercentile(ctx context.Context, lowerPercentile, upperPercentile float64) (float64, error) {
	// 参数验证
	if lowerPercentile < 0 || lowerPercentile >= 1 || upperPercentile <= 0 || upperPercentile > 1 {
		s.logger.ErrorContext(ctx, "百分位参数无效",
			"lower_percentile", lowerPercentile,
			"upper_percentile", upperPercentile,
		)
		return 0, fmt.Errorf("百分位参数无效：lowerPercentile=%f, upperPercentile=%f", lowerPercentile, upperPercentile)
	}
	if lowerPercentile >= upperPercentile {
		s.logger.ErrorContext(ctx, "百分位参数顺序错误",
			"lower_percentile", lowerPercentile,
			"upper_percentile", upperPercentile,
		)
		return 0, fmt.Errorf("下限百分位 (%f) 必须小于上限百分位 (%f)", lowerPercentile, upperPercentile)
	}

	r := query.Q.RequestLog

	// 获取过去 24 小时内有效的首字时间数据
	// 注意：这里使用指针是因为数据库字段可能为 NULL
	var firstByteTimes []*time.Duration
	err := r.WithContext(ctx).
		Select(r.FirstByteTime).
		Where(r.FirstByteTime.IsNotNull()).
		Where(r.Timestamp.Gte(time.Now().Add(-24 * time.Hour))).
		Scan(&firstByteTimes)
	if err != nil {
		s.logger.ErrorContext(ctx, "获取首字时间数据失败", "error", err)
		return 0, fmt.Errorf("获取首字时间数据失败：%w", err)
	}

	// 无数据时返回 0
	if len(firstByteTimes) == 0 {
		s.logger.DebugContext(ctx, "无首字时间数据")
		return 0, nil
	}

	// 转换为纳秒值，过滤 nil 值和无效值
	durations := make([]uint64, 0, len(firstByteTimes))
	for _, d := range firstByteTimes {
		if d != nil && *d > 0 {
			durations = append(durations, uint64(d.Nanoseconds()))
		}
	}

	// 过滤后无有效数据
	if len(durations) == 0 {
		s.logger.DebugContext(ctx, "过滤后无有效首字时间数据")
		return 0, nil
	}

	// 数据量太少时不进行百分位过滤，直接计算平均值
	if len(durations) < 10 {
		avg := calculateAverage(durations)
		s.logger.DebugContext(ctx, "数据量较少，直接计算平均值",
			"data_count", len(durations),
			"avg_ns", avg,
		)
		return avg, nil
	}

	// 对持续时间进行排序（升序）
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	// 计算百分位索引
	lowerIndex := int(float64(len(durations)) * lowerPercentile)
	upperIndex := int(float64(len(durations)) * upperPercentile)

	// 确保索引在有效范围内
	// upperIndex 应该向上取整以包含更多数据点
	if lowerIndex < 0 {
		lowerIndex = 0
	}
	if upperIndex > len(durations) {
		upperIndex = len(durations)
	}

	// 确保至少有一个数据点
	if lowerIndex >= upperIndex {
		upperIndex = lowerIndex + 1
	}

	// 提取过滤后的数据（去除极端值）
	filteredDurations := durations[lowerIndex:upperIndex]

	if len(filteredDurations) == 0 {
		s.logger.WarnContext(ctx, "百分位过滤后无数据")
		return 0, nil
	}

	// 计算平均值
	avg := calculateAverage(filteredDurations)

	s.logger.DebugContext(ctx, "成功计算平均首字时间",
		"original_count", len(durations),
		"filtered_count", len(filteredDurations),
		"avg_ns", avg,
		"lower_percentile", lowerPercentile,
		"upper_percentile", upperPercentile,
	)

	return avg, nil
}

// calculateAverage 计算 uint64 切片的平均值
//
// 提取为独立函数便于复用和测试
func calculateAverage(values []uint64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum uint64
	for _, v := range values {
		sum += v
	}

	return float64(sum) / float64(len(values))
}
