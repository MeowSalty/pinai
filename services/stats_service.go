package services

import (
	"context"
	"fmt"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

// StatsOverviewResponse 定义了全局概览数据的响应结构
type StatsOverviewResponse struct {
	TotalRequests    int64   `json:"total_requests"` // 总请求量
	SuccessRate      float64 `json:"success_rate"`   // 成功率
	AvgFirstByteTime float64 `json:"avg_first_byte"` // 平均首字时间 (毫秒)
	RPM              float64 `json:"rpm"`            // 每分钟请求数
}

// StatsServiceInterface 定义统计服务接口
type StatsServiceInterface interface {
	// GetOverview 获取全局概览数据
	GetOverview(ctx context.Context) (*StatsOverviewResponse, error)
}

// statsService 是 StatsServiceInterface 接口的具体实现
type statsService struct{}

// NewStatsService 创建一个新的统计服务实例
func NewStatsService() StatsServiceInterface {
	return &statsService{}
}

// GetOverview 实现获取全局概览数据的业务逻辑
func (s *statsService) GetOverview(ctx context.Context) (*StatsOverviewResponse, error) {
	q := query.Q
	r := q.RequestStat

	// 获取总请求数
	totalRequests, err := r.WithContext(ctx).Count()
	if err != nil {
		return nil, fmt.Errorf("获取总请求数失败：%w", err)
	}

	// 获取成功请求数
	successRequests, err := r.WithContext(ctx).Where(r.Success.Is(true)).Count()
	if err != nil {
		return nil, fmt.Errorf("获取成功请求数失败：%w", err)
	}

	// 计算成功率
	var successRate float64
	if totalRequests > 0 {
		successRate = float64(successRequests) / float64(totalRequests)
	}

	// 计算平均首字时间
	type avgFirstByteResult struct {
		AvgFirstByteTime float64 `gorm:"column:avg_first_byte"`
	}
	var avgResult avgFirstByteResult
	err = r.WithContext(ctx).
		Select(r.FirstByteTime.Avg().As("avg_first_byte")).
		Where(r.FirstByteTime.IsNotNull()).
		Scan(&avgResult)
	if err != nil {
		return nil, fmt.Errorf("计算平均首字时间失败：%w", err)
	}

	// 计算过去 1 分钟的请求数 (RPM)
	oneMinuteAgo := time.Now().Add(-time.Minute)
	recentRequests, err := r.WithContext(ctx).
		Where(r.Timestamp.Gte(oneMinuteAgo)).
		Count()
	if err != nil {
		return nil, fmt.Errorf("计算 RPM 失败：%w", err)
	}

	return &StatsOverviewResponse{
		TotalRequests:    totalRequests,
		SuccessRate:      successRate,
		AvgFirstByteTime: avgResult.AvgFirstByteTime / float64(time.Millisecond), // 转换为毫秒
		RPM:              float64(recentRequests),
	}, nil
}
