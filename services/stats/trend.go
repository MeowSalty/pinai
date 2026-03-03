package stats

import (
	"time"
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

// ceilToHour 将时间向上取整到整点（若已是整点则保持不变）
func ceilToHour(t time.Time) time.Time {
	u := t
	if u.Minute() == 0 && u.Second() == 0 && u.Nanosecond() == 0 {
		return u
	}
	return u.Truncate(time.Hour).Add(time.Hour)
}
