package stats

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

const dashboardTopN = 5

// dashboardLogRow 定义仪表盘聚合查询的原始行结构
type dashboardLogRow struct {
	Timestamp         time.Time `gorm:"column:timestamp"`
	Success           bool      `gorm:"column:success"`
	FirstByteTime     *int64    `gorm:"column:first_byte_time"`
	OriginalModelName string    `gorm:"column:original_model_name"`
	PlatformID        uint      `gorm:"column:platform_id"`
	PromptTokens      *int      `gorm:"column:prompt_tokens"`
	CompletionTokens  *int      `gorm:"column:completion_tokens"`
	TotalTokens       *int      `gorm:"column:total_tokens"`
}

type dashboardCallAgg struct {
	RequestCount int64
	SuccessCount int64
}

type dashboardUsageAgg struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}

// GetDashboard 获取仪表盘所有数据（单次查询优化版本）
func (s *service) GetDashboard(ctx context.Context, trendRange TrendRange) (*DashboardResponse, error) {
	if trendRange == "" {
		trendRange = TrendRange24h
	}

	cfg, ok := trendRangeConfigs[trendRange]
	if !ok {
		return nil, fmt.Errorf("无效的时间范围参数，可选值：24h, 7d, 30d")
	}

	now := time.Now().UTC()
	bucketEnd := ceilToHour(now)
	bucketStart := bucketEnd.Add(-cfg.Granularity * time.Duration(cfg.Points))

	s.logger.InfoContext(ctx, "开始聚合仪表盘数据",
		"range", trendRange,
		"granularity", cfg.Label,
		"bucket_start", bucketStart,
		"bucket_end", bucketEnd,
	)

	var rows []dashboardLogRow
	err := query.Q.RequestLog.WithContext(ctx).
		UnderlyingDB().
		Table("request_logs").
		Select("timestamp, success, first_byte_time, original_model_name, platform_id, prompt_tokens, completion_tokens, total_tokens").
		Where("timestamp >= ? AND timestamp < ?", bucketStart, bucketEnd).
		Order("timestamp ASC").
		Scan(&rows).Error
	if err != nil {
		s.logger.ErrorContext(ctx, "查询仪表盘原始数据失败", "error", err)
		return nil, fmt.Errorf("查询仪表盘数据失败：%w", err)
	}

	platformNameMap, err := s.loadPlatformNameMap(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "加载平台名称映射失败", "error", err)
		return nil, fmt.Errorf("加载平台名称失败：%w", err)
	}

	points := make([]TrendDataPoint, cfg.Points)
	for i := 0; i < cfg.Points; i++ {
		points[i] = TrendDataPoint{
			Timestamp: bucketStart.Add(cfg.Granularity * time.Duration(i+1)),
		}
	}

	modelCallAgg := make(map[string]*dashboardCallAgg)
	platformCallAgg := make(map[uint]*dashboardCallAgg)
	modelUsageAgg := make(map[string]*dashboardUsageAgg)
	platformUsageAgg := make(map[uint]*dashboardUsageAgg)
	firstByteDurations := make([]uint64, 0, len(rows))

	var (
		overview      DashboardOverview
		successCount  int64
		trendSummary  TrendSummary
		totalUsageTok int64
	)

	for _, row := range rows {
		overview.TotalRequests++
		if row.Success {
			successCount++
		}

		if row.FirstByteTime != nil && *row.FirstByteTime > 0 {
			firstByteDurations = append(firstByteDurations, uint64(*row.FirstByteTime))
		}

		promptTokens := ptrIntToInt64(row.PromptTokens)
		completionTokens := ptrIntToInt64(row.CompletionTokens)
		totalTokens := ptrIntToInt64(row.TotalTokens)
		if row.TotalTokens == nil && (promptTokens > 0 || completionTokens > 0) {
			totalTokens = promptTokens + completionTokens
		}

		overview.TotalPromptTokens += promptTokens
		overview.TotalCompletionTokens += completionTokens
		overview.TotalTokens += totalTokens

		modelName := row.OriginalModelName
		if modelName == "" {
			modelName = "unknown"
		}

		if _, exists := modelCallAgg[modelName]; !exists {
			modelCallAgg[modelName] = &dashboardCallAgg{}
		}
		modelCallAgg[modelName].RequestCount++
		if row.Success {
			modelCallAgg[modelName].SuccessCount++
		}

		if _, exists := platformCallAgg[row.PlatformID]; !exists {
			platformCallAgg[row.PlatformID] = &dashboardCallAgg{}
		}
		platformCallAgg[row.PlatformID].RequestCount++
		if row.Success {
			platformCallAgg[row.PlatformID].SuccessCount++
		}

		if row.TotalTokens != nil || row.PromptTokens != nil || row.CompletionTokens != nil {
			if _, exists := modelUsageAgg[modelName]; !exists {
				modelUsageAgg[modelName] = &dashboardUsageAgg{}
			}
			modelUsageAgg[modelName].PromptTokens += promptTokens
			modelUsageAgg[modelName].CompletionTokens += completionTokens
			modelUsageAgg[modelName].TotalTokens += totalTokens

			if _, exists := platformUsageAgg[row.PlatformID]; !exists {
				platformUsageAgg[row.PlatformID] = &dashboardUsageAgg{}
			}
			platformUsageAgg[row.PlatformID].PromptTokens += promptTokens
			platformUsageAgg[row.PlatformID].CompletionTokens += completionTokens
			platformUsageAgg[row.PlatformID].TotalTokens += totalTokens

			totalUsageTok += totalTokens
		}

		idx := int(row.Timestamp.UTC().Sub(bucketStart) / cfg.Granularity)
		if idx >= 0 && idx < len(points) {
			points[idx].RequestCount++
			points[idx].TotalTokens += totalTokens
		}
	}

	overview.SuccessRate = s.calculateSuccessRate(overview.TotalRequests, successCount)
	overview.AvgFirstByteTime = s.calculateAverageWithPercentile(firstByteDurations, percentileLower, percentileUpper)

	for _, p := range points {
		trendSummary.TotalRequests += p.RequestCount
		trendSummary.TotalTokens += p.TotalTokens
	}
	if len(points) > 0 {
		trendSummary.AvgRequestsPerPoint = float64(trendSummary.TotalRequests) / float64(len(points))
		trendSummary.AvgTokensPerPoint = float64(trendSummary.TotalTokens) / float64(len(points))
	}

	ranks := DashboardRanks{
		ModelCall:     buildModelCallRankItems(modelCallAgg, overview.TotalRequests),
		PlatformCall:  buildPlatformCallRankItems(platformCallAgg, platformNameMap, overview.TotalRequests),
		ModelUsage:    buildModelUsageRankItems(modelUsageAgg, totalUsageTok),
		PlatformUsage: buildPlatformUsageRankItems(platformUsageAgg, platformNameMap, totalUsageTok),
	}

	resp := &DashboardResponse{
		Range:    string(trendRange),
		Overview: overview,
		Ranks:    ranks,
		Trend: &TrendResponse{
			Range:       string(trendRange),
			Granularity: cfg.Label,
			DataPoints:  points,
			Summary:     trendSummary,
		},
	}

	s.logger.InfoContext(ctx, "成功聚合仪表盘数据",
		"range", trendRange,
		"rows", len(rows),
		"total_requests", resp.Overview.TotalRequests,
		"total_tokens", resp.Overview.TotalTokens,
	)

	return resp, nil
}

// loadPlatformNameMap 加载平台 ID 到名称的映射
func (s *service) loadPlatformNameMap(ctx context.Context) (map[uint]string, error) {
	platforms, err := query.Q.Platform.WithContext(ctx).
		Select(query.Q.Platform.ID, query.Q.Platform.Name).
		Find()
	if err != nil {
		return nil, err
	}

	nameMap := make(map[uint]string, len(platforms))
	for _, p := range platforms {
		nameMap[p.ID] = p.Name
	}

	return nameMap, nil
}

// calculateAverageWithPercentile 使用百分位数过滤法计算平均值
func (s *service) calculateAverageWithPercentile(values []uint64, lowerPercentile, upperPercentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	if len(values) < 10 {
		return calculateAverage(values)
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	lowerIndex := int(float64(len(values)) * lowerPercentile)
	upperIndex := int(float64(len(values)) * upperPercentile)

	if lowerIndex < 0 {
		lowerIndex = 0
	}
	if upperIndex > len(values) {
		upperIndex = len(values)
	}
	if lowerIndex >= upperIndex {
		upperIndex = lowerIndex + 1
	}
	if upperIndex > len(values) {
		upperIndex = len(values)
	}

	filteredValues := values[lowerIndex:upperIndex]
	if len(filteredValues) == 0 {
		return 0
	}

	return calculateAverage(filteredValues)
}

// ptrIntToInt64 将 *int 安全转换为 int64
func ptrIntToInt64(v *int) int64 {
	if v == nil {
		return 0
	}
	return int64(*v)
}

func buildModelCallRankItems(agg map[string]*dashboardCallAgg, totalRequests int64) []ModelCallRankItem {
	items := make([]ModelCallRankItem, 0, len(agg))
	for modelName, stat := range agg {
		var successRate float64
		if stat.RequestCount > 0 {
			successRate = float64(stat.SuccessCount) / float64(stat.RequestCount)
		}

		var percentage float64
		if totalRequests > 0 {
			percentage = float64(stat.RequestCount) / float64(totalRequests)
		}

		items = append(items, ModelCallRankItem{
			ModelName:    modelName,
			RequestCount: stat.RequestCount,
			SuccessRate:  successRate,
			Percentage:   percentage,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].RequestCount == items[j].RequestCount {
			return items[i].ModelName < items[j].ModelName
		}
		return items[i].RequestCount > items[j].RequestCount
	})

	if len(items) > dashboardTopN {
		items = items[:dashboardTopN]
	}

	return items
}

func buildPlatformCallRankItems(agg map[uint]*dashboardCallAgg, platformNameMap map[uint]string, totalRequests int64) []PlatformCallRankItem {
	items := make([]PlatformCallRankItem, 0, len(agg))
	for platformID, stat := range agg {
		platformName := platformNameMap[platformID]
		if platformName == "" {
			platformName = fmt.Sprintf("平台#%d", platformID)
		}

		var successRate float64
		if stat.RequestCount > 0 {
			successRate = float64(stat.SuccessCount) / float64(stat.RequestCount)
		}

		var percentage float64
		if totalRequests > 0 {
			percentage = float64(stat.RequestCount) / float64(totalRequests)
		}

		items = append(items, PlatformCallRankItem{
			PlatformName: platformName,
			RequestCount: stat.RequestCount,
			SuccessRate:  successRate,
			Percentage:   percentage,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].RequestCount == items[j].RequestCount {
			return items[i].PlatformName < items[j].PlatformName
		}
		return items[i].RequestCount > items[j].RequestCount
	})

	if len(items) > dashboardTopN {
		items = items[:dashboardTopN]
	}

	return items
}

func buildModelUsageRankItems(agg map[string]*dashboardUsageAgg, totalTokens int64) []ModelUsageRankItem {
	items := make([]ModelUsageRankItem, 0, len(agg))
	for modelName, stat := range agg {
		var percentage float64
		if totalTokens > 0 {
			percentage = float64(stat.TotalTokens) / float64(totalTokens)
		}

		items = append(items, ModelUsageRankItem{
			ModelName:        modelName,
			TotalTokens:      stat.TotalTokens,
			PromptTokens:     stat.PromptTokens,
			CompletionTokens: stat.CompletionTokens,
			Percentage:       percentage,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].TotalTokens == items[j].TotalTokens {
			return items[i].ModelName < items[j].ModelName
		}
		return items[i].TotalTokens > items[j].TotalTokens
	})

	if len(items) > dashboardTopN {
		items = items[:dashboardTopN]
	}

	return items
}

func buildPlatformUsageRankItems(agg map[uint]*dashboardUsageAgg, platformNameMap map[uint]string, totalTokens int64) []PlatformUsageRankItem {
	items := make([]PlatformUsageRankItem, 0, len(agg))
	for platformID, stat := range agg {
		platformName := platformNameMap[platformID]
		if platformName == "" {
			platformName = fmt.Sprintf("平台#%d", platformID)
		}

		var percentage float64
		if totalTokens > 0 {
			percentage = float64(stat.TotalTokens) / float64(totalTokens)
		}

		items = append(items, PlatformUsageRankItem{
			PlatformName:     platformName,
			TotalTokens:      stat.TotalTokens,
			PromptTokens:     stat.PromptTokens,
			CompletionTokens: stat.CompletionTokens,
			Percentage:       percentage,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].TotalTokens == items[j].TotalTokens {
			return items[i].PlatformName < items[j].PlatformName
		}
		return items[i].TotalTokens > items[j].TotalTokens
	})

	if len(items) > dashboardTopN {
		items = items[:dashboardTopN]
	}

	return items
}
