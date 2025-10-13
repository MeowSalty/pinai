package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// StatsOverviewResponse 定义了全局概览数据的响应结构
type StatsOverviewResponse struct {
	TotalRequests    int64   `json:"total_requests"` // 总请求量
	SuccessRate      float64 `json:"success_rate"`   // 成功率
	AvgFirstByteTime float64 `json:"avg_first_byte"` // 平均首字时间 (微秒)
}

// StatsRealtimeResponse 定义了实时数据的响应结构
type StatsRealtimeResponse struct {
	RPM float64 `json:"rpm"` // 每分钟请求数
}

// ModelRankItem 定义了模型排名项
type ModelRankItem struct {
	ModelName    string  `json:"model_name"`    // 模型名称
	RequestCount int64   `json:"request_count"` // 请求数量
	SuccessRate  float64 `json:"success_rate"`  // 成功率
	Percentage   float64 `json:"percentage"`    // 占比
}

// PlatformRankItem 定义了平台排名项
type PlatformRankItem struct {
	PlatformName string  `json:"platform_name"` // 平台名称
	RequestCount int64   `json:"request_count"` // 请求数量
	SuccessRate  float64 `json:"success_rate"`  // 成功率
	Percentage   float64 `json:"percentage"`    // 占比
}

// ModelRankResponse 定义了模型排名响应结构
type ModelRankResponse struct {
	TotalRequests int64           `json:"total_requests"` // 总请求量
	Models        []ModelRankItem `json:"models"`         // 模型排名列表
}

// PlatformRankResponse 定义了平台排名响应结构
type PlatformRankResponse struct {
	TotalRequests int64              `json:"total_requests"` // 总请求量
	Platforms     []PlatformRankItem `json:"platforms"`      // 平台排名列表
}

// ListRequestLogsOptions 定义了获取请求状态列表的筛选选项
type ListRequestLogsOptions struct {
	// 时间范围筛选
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Success     *bool      `json:"success,omitempty"`
	RequestType *string    `json:"request_type,omitempty"`
	ModelName   *string    `json:"model_name,omitempty"`
	Page        int        `json:"page"`
	PageSize    int        `json:"page_size"`
}

// StatsServiceInterface 定义统计服务接口
type StatsServiceInterface interface {
	// GetOverview 获取全局概览数据
	GetOverview(ctx context.Context, duration time.Duration) (*StatsOverviewResponse, error)

	// GetRealtime 获取实时数据
	GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error)

	// ListRequestLogs 获取请求状态列表
	ListRequestLogs(ctx context.Context, opts ListRequestLogsOptions) ([]*types.RequestLog, int64, error)

	// GetModelRank 获取模型排名前 10
	GetModelRank(ctx context.Context, duration time.Duration) (*ModelRankResponse, error)

	// GetPlatformRank 获取平台排名前 10
	GetPlatformRank(ctx context.Context, duration time.Duration) (*PlatformRankResponse, error)
}

// statsService 是 StatsServiceInterface 接口的具体实现
type statsService struct{}

// NewStatsService 创建一个新的统计服务实例
func NewStatsService() StatsServiceInterface {
	return &statsService{}
}

// GetOverview 实现获取全局概览数据的业务逻辑
func (s *statsService) GetOverview(ctx context.Context, duration time.Duration) (*StatsOverviewResponse, error) {
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

// GetRealtime 实现获取实时数据的业务逻辑
func (s *statsService) GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error) {
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

// ListRequestLogs 实现获取请求状态列表的业务逻辑
func (s *statsService) ListRequestLogs(ctx context.Context, opts ListRequestLogsOptions) ([]*types.RequestLog, int64, error) {
	q := query.Q
	r := q.RequestLog

	// 构建查询条件
	queryBuilder := r.WithContext(ctx)

	// 时间范围筛选
	if opts.StartTime != nil {
		queryBuilder = queryBuilder.Where(r.Timestamp.Gte(*opts.StartTime))
	}
	if opts.EndTime != nil {
		queryBuilder = queryBuilder.Where(r.Timestamp.Lte(*opts.EndTime))
	}

	// 结果状态筛选
	if opts.Success != nil {
		queryBuilder = queryBuilder.Where(r.Success.Is(*opts.Success))
	}

	// 请求类型筛选
	if opts.RequestType != nil {
		queryBuilder = queryBuilder.Where(r.RequestType.Eq(*opts.RequestType))
	}

	// 模型名称筛选
	if opts.ModelName != nil {
		queryBuilder = queryBuilder.Where(r.ModelName.Eq(*opts.ModelName))
	}

	// 计算偏移量
	offset := (opts.Page - 1) * opts.PageSize

	// 添加排序条件，按时间倒序排列以确保最新数据在前
	queryBuilder = queryBuilder.Order(r.Timestamp.Desc())

	// 执行分页查询
	result, count, err := queryBuilder.FindByPage(offset, opts.PageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("获取请求状态列表失败：%w", err)
	}

	return result, count, nil
}

// calculateAvgFirstByteTimeWithPercentile 使用百分位数过滤法计算平均首字时间
// lowerPercentile 和 upperPercentile 分别指定要过滤的下限和上限百分位（0.0-1.0）
func (s *statsService) calculateAvgFirstByteTimeWithPercentile(ctx context.Context, lowerPercentile, upperPercentile float64) (float64, error) {
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

// GetModelRank 获取模型排名前 10
func (s *statsService) GetModelRank(ctx context.Context, duration time.Duration) (*ModelRankResponse, error) {
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

	// 获取所有模型名称
	var modelNames []string
	err = r.WithContext(ctx).
		Where(r.Timestamp.Gte(startTime)).
		Distinct(r.OriginalModelName).
		Pluck(r.OriginalModelName, &modelNames)
	if err != nil {
		return nil, fmt.Errorf("获取模型名称列表失败：%w", err)
	}

	// 计算每个模型的统计数据
	modelRankItems := make([]ModelRankItem, 0, len(modelNames))
	for _, modelName := range modelNames {
		// 获取该模型的总请求数
		modelTotal, err := r.WithContext(ctx).
			Where(r.Timestamp.Gte(startTime)).
			Where(r.OriginalModelName.Eq(modelName)).
			Count()
		if err != nil {
			return nil, fmt.Errorf("获取模型 %s 请求数失败：%w", modelName, err)
		}

		// 获取该模型的成功请求数
		modelSuccess, err := r.WithContext(ctx).
			Where(r.Timestamp.Gte(startTime)).
			Where(r.OriginalModelName.Eq(modelName)).
			Where(r.Success.Is(true)).
			Count()
		if err != nil {
			return nil, fmt.Errorf("获取模型 %s 成功请求数失败：%w", modelName, err)
		}

		var successRate float64
		if modelTotal > 0 {
			successRate = float64(modelSuccess) / float64(modelTotal)
		}

		var percentage float64
		if totalRequests > 0 {
			percentage = float64(modelTotal) / float64(totalRequests)
		}

		modelRankItems = append(modelRankItems, ModelRankItem{
			ModelName:    modelName,
			RequestCount: modelTotal,
			SuccessRate:  successRate,
			Percentage:   percentage,
		})
	}

	// 按请求数量降序排序
	sort.Slice(modelRankItems, func(i, j int) bool {
		return modelRankItems[i].RequestCount > modelRankItems[j].RequestCount
	})

	// 只取前 10 个
	if len(modelRankItems) > 10 {
		modelRankItems = modelRankItems[:10]
	}

	return &ModelRankResponse{
		TotalRequests: totalRequests,
		Models:        modelRankItems,
	}, nil
}

// GetPlatformRank 获取平台排名前 10
func (s *statsService) GetPlatformRank(ctx context.Context, duration time.Duration) (*PlatformRankResponse, error) {
	q := query.Q
	r := q.RequestLog
	p := q.Platform

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

	// 获取所有记录并手动统计
	logs, err := r.WithContext(ctx).
		Where(r.Timestamp.Gte(startTime)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("获取请求日志失败：%w", err)
	}

	// 手动统计每个平台的数据
	platformStats := make(map[uint]*struct {
		total   int64
		success int64
	})

	for _, log := range logs {
		platformID := log.ChannelInfo.PlatformID
		if _, exists := platformStats[platformID]; !exists {
			platformStats[platformID] = &struct {
				total   int64
				success int64
			}{}
		}
		platformStats[platformID].total++
		if log.Success {
			platformStats[platformID].success++
		}
	}

	// 获取平台名称映射
	platformIDs := make([]uint, 0, len(platformStats))
	for platformID := range platformStats {
		platformIDs = append(platformIDs, platformID)
	}

	platforms, err := p.WithContext(ctx).
		Where(p.ID.In(platformIDs...)).
		Find()
	if err != nil {
		return nil, fmt.Errorf("获取平台信息失败：%w", err)
	}

	platformNames := make(map[uint]string)
	for _, platform := range platforms {
		platformNames[platform.ID] = platform.Name
	}

	// 构建排名列表
	platformRankItems := make([]PlatformRankItem, 0, len(platformStats))
	for platformID, stats := range platformStats {
		var successRate float64
		if stats.total > 0 {
			successRate = float64(stats.success) / float64(stats.total)
		}

		var percentage float64
		if totalRequests > 0 {
			percentage = float64(stats.total) / float64(totalRequests)
		}

		platformRankItems = append(platformRankItems, PlatformRankItem{
			PlatformName: platformNames[platformID],
			RequestCount: stats.total,
			SuccessRate:  successRate,
			Percentage:   percentage,
		})
	}

	// 按请求数量降序排序
	sort.Slice(platformRankItems, func(i, j int) bool {
		return platformRankItems[i].RequestCount > platformRankItems[j].RequestCount
	})

	// 只取前 10 个
	if len(platformRankItems) > 10 {
		platformRankItems = platformRankItems[:10]
	}

	return &PlatformRankResponse{
		TotalRequests: totalRequests,
		Platforms:     platformRankItems,
	}, nil
}
