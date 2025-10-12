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

// ListRequestLogsOptions 定义了获取请求状态列表的筛选选项
type ListRequestLogsOptions struct {
	// 时间范围筛选
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`

	// 结果状态筛选
	Success *bool `json:"success,omitempty"`

	// 请求类型筛选
	RequestType *string `json:"request_type,omitempty"`

	// 模型名称筛选
	ModelName *string `json:"model_name,omitempty"`

	// 分页参数
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// StatsServiceInterface 定义统计服务接口
type StatsServiceInterface interface {
	// GetOverview 获取全局概览数据
	GetOverview(ctx context.Context, duration time.Duration) (*StatsOverviewResponse, error)

	// GetRealtime 获取实时数据
	GetRealtime(ctx context.Context) (*StatsRealtimeResponse, error)

	// ListRequestLogs 获取请求状态列表
	ListRequestLogs(ctx context.Context, opts ListRequestLogsOptions) ([]*types.RequestLog, int64, error)
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
