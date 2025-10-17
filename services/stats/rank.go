package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

// modelRankResult 定义模型排名查询结果结构
type modelRankResult struct {
	ModelName    string `gorm:"column:original_model_name"`
	RequestCount int64  `gorm:"column:request_count"`
	SuccessCount int64  `gorm:"column:success_count"`
}

// GetModelRank 获取模型排名前 10
func (s *service) GetModelRank(ctx context.Context, duration time.Duration) (*ModelRankResponse, error) {
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

	// 使用数据库聚合查询一次性获取所有模型的统计数据
	// SELECT
	//   original_model_name,
	//   COUNT(*) as request_count,
	//   SUM(CASE WHEN success = true THEN 1 ELSE 0 END) as success_count
	// FROM request_logs
	// WHERE timestamp >= ?
	// GROUP BY original_model_name
	// ORDER BY request_count DESC
	// LIMIT 10
	var results []modelRankResult
	err = r.WithContext(ctx).
		UnderlyingDB().
		Select("original_model_name, COUNT(*) as request_count, SUM(CASE WHEN success = ? THEN 1 ELSE 0 END) as success_count", true).
		Where("timestamp >= ?", startTime).
		Group("original_model_name").
		Order("request_count DESC").
		Limit(10).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("获取模型排名失败：%w", err)
	}

	// 构建响应结果
	modelRankItems := make([]ModelRankItem, 0, len(results))
	for _, result := range results {
		var successRate float64
		if result.RequestCount > 0 {
			successRate = float64(result.SuccessCount) / float64(result.RequestCount)
		}

		var percentage float64
		if totalRequests > 0 {
			percentage = float64(result.RequestCount) / float64(totalRequests)
		}

		modelRankItems = append(modelRankItems, ModelRankItem{
			ModelName:    result.ModelName,
			RequestCount: result.RequestCount,
			SuccessRate:  successRate,
			Percentage:   percentage,
		})
	}

	return &ModelRankResponse{
		TotalRequests: totalRequests,
		Models:        modelRankItems,
	}, nil
}

// platformRankResult 定义平台排名查询结果结构
type platformRankResult struct {
	PlatformID   uint   `gorm:"column:platform_id"`
	PlatformName string `gorm:"column:platform_name"`
	RequestCount int64  `gorm:"column:request_count"`
	SuccessCount int64  `gorm:"column:success_count"`
}

// GetPlatformRank 获取平台排名前 10
func (s *service) GetPlatformRank(ctx context.Context, duration time.Duration) (*PlatformRankResponse, error) {
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

	// 使用数据库聚合查询和 JOIN 一次性获取所有平台的统计数据
	// SELECT
	//   r.platform_id,
	//   p.name as platform_name,
	//   COUNT(*) as request_count,
	//   SUM(CASE WHEN r.success = true THEN 1 ELSE 0 END) as success_count
	// FROM request_logs r
	// LEFT JOIN platforms p ON r.platform_id = p.id
	// WHERE r.timestamp >= ?
	// GROUP BY r.platform_id, p.name
	// ORDER BY request_count DESC
	// LIMIT 10
	var results []platformRankResult
	err = r.WithContext(ctx).
		UnderlyingDB().
		Table("request_logs r").
		Select("r.platform_id, p.name as platform_name, COUNT(*) as request_count, SUM(CASE WHEN r.success = ? THEN 1 ELSE 0 END) as success_count", true).
		Joins("LEFT JOIN platforms p ON r.platform_id = p.id").
		Where("r.timestamp >= ?", startTime).
		Group("r.platform_id, p.name").
		Order("request_count DESC").
		Limit(10).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("获取平台排名失败：%w", err)
	}

	// 构建响应结果
	platformRankItems := make([]PlatformRankItem, 0, len(results))
	for _, result := range results {
		var successRate float64
		if result.RequestCount > 0 {
			successRate = float64(result.SuccessCount) / float64(result.RequestCount)
		}

		var percentage float64
		if totalRequests > 0 {
			percentage = float64(result.RequestCount) / float64(totalRequests)
		}

		platformRankItems = append(platformRankItems, PlatformRankItem{
			PlatformName: result.PlatformName,
			RequestCount: result.RequestCount,
			SuccessRate:  successRate,
			Percentage:   percentage,
		})
	}

	return &PlatformRankResponse{
		TotalRequests: totalRequests,
		Platforms:     platformRankItems,
	}, nil
}
