package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

// modelCallRankResult 定义模型调用排名查询结果结构
type modelCallRankResult struct {
	ModelName    string `gorm:"column:original_model_name"`
	RequestCount int64  `gorm:"column:request_count"`
	SuccessCount int64  `gorm:"column:success_count"`
}

// GetModelCallRank 获取模型调用排名前 5
func (s *service) GetModelCallRank(ctx context.Context, duration time.Duration) (*ModelCallRankResponse, error) {
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
	var results []modelCallRankResult
	err = r.WithContext(ctx).
		UnderlyingDB().
		Select("original_model_name, COUNT(*) as request_count, SUM(CASE WHEN success = ? THEN 1 ELSE 0 END) as success_count", true).
		Where("timestamp >= ?", startTime).
		Group("original_model_name").
		Order("request_count DESC").
		Limit(5).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("获取模型排名失败：%w", err)
	}

	// 构建响应结果
	modelCallRankItems := make([]ModelCallRankItem, 0, len(results))
	for _, result := range results {
		var successRate float64
		if result.RequestCount > 0 {
			successRate = float64(result.SuccessCount) / float64(result.RequestCount)
		}

		var percentage float64
		if totalRequests > 0 {
			percentage = float64(result.RequestCount) / float64(totalRequests)
		}

		modelCallRankItems = append(modelCallRankItems, ModelCallRankItem{
			ModelName:    result.ModelName,
			RequestCount: result.RequestCount,
			SuccessRate:  successRate,
			Percentage:   percentage,
		})
	}

	return &ModelCallRankResponse{
		TotalRequests: totalRequests,
		Models:        modelCallRankItems,
	}, nil
}

// platformCallRankResult 定义平台调用排名查询结果结构
type platformCallRankResult struct {
	PlatformID   uint   `gorm:"column:platform_id"`
	PlatformName string `gorm:"column:platform_name"`
	RequestCount int64  `gorm:"column:request_count"`
	SuccessCount int64  `gorm:"column:success_count"`
}

// GetPlatformCallRank 获取平台调用排名前 5
func (s *service) GetPlatformCallRank(ctx context.Context, duration time.Duration) (*PlatformCallRankResponse, error) {
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
	var results []platformCallRankResult
	err = r.WithContext(ctx).
		UnderlyingDB().
		Table("request_logs r").
		Select("r.platform_id, p.name as platform_name, COUNT(*) as request_count, SUM(CASE WHEN r.success = ? THEN 1 ELSE 0 END) as success_count", true).
		Joins("LEFT JOIN platforms p ON r.platform_id = p.id").
		Where("r.timestamp >= ?", startTime).
		Group("r.platform_id, p.name").
		Order("request_count DESC").
		Limit(5).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("获取平台排名失败：%w", err)
	}

	// 构建响应结果
	platformCallRankItems := make([]PlatformCallRankItem, 0, len(results))
	for _, result := range results {
		var successRate float64
		if result.RequestCount > 0 {
			successRate = float64(result.SuccessCount) / float64(result.RequestCount)
		}

		var percentage float64
		if totalRequests > 0 {
			percentage = float64(result.RequestCount) / float64(totalRequests)
		}

		platformCallRankItems = append(platformCallRankItems, PlatformCallRankItem{
			PlatformName: result.PlatformName,
			RequestCount: result.RequestCount,
			SuccessRate:  successRate,
			Percentage:   percentage,
		})
	}

	return &PlatformCallRankResponse{
		TotalRequests: totalRequests,
		Platforms:     platformCallRankItems,
	}, nil
}

// modelUsageRankResult 定义模型用量排名查询结果结构
type modelUsageRankResult struct {
	ModelName        string `gorm:"column:original_model_name"`
	TotalTokens      int64  `gorm:"column:total_tokens"`
	PromptTokens     int64  `gorm:"column:prompt_tokens"`
	CompletionTokens int64  `gorm:"column:completion_tokens"`
}

// GetModelUsageRank 获取模型用量排名前 5
func (s *service) GetModelUsageRank(ctx context.Context, duration time.Duration) (*ModelUsageRankResponse, error) {
	q := query.Q
	r := q.RequestLog

	// 设置默认时间范围为 24 小时
	if duration == 0 {
		duration = 24 * time.Hour
	}

	startTime := time.Now().Add(-duration)

	// 获取总 Token 数
	var totalTokensSum struct {
		Total int64 `gorm:"column:total"`
	}
	err := r.WithContext(ctx).
		UnderlyingDB().
		Select("COALESCE(SUM(total_tokens), 0) as total").
		Where("timestamp >= ? AND total_tokens IS NOT NULL", startTime).
		Scan(&totalTokensSum).Error
	if err != nil {
		return nil, fmt.Errorf("获取总 Token 数失败：%w", err)
	}

	// 使用数据库聚合查询获取模型用量统计
	// SELECT
	//   original_model_name,
	//   COALESCE(SUM(total_tokens), 0) as total_tokens,
	//   COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
	//   COALESCE(SUM(completion_tokens), 0) as completion_tokens
	// FROM request_logs
	// WHERE timestamp >= ? AND total_tokens IS NOT NULL
	// GROUP BY original_model_name
	// ORDER BY total_tokens DESC
	// LIMIT 5
	var results []modelUsageRankResult
	err = r.WithContext(ctx).
		UnderlyingDB().
		Select("original_model_name, COALESCE(SUM(total_tokens), 0) as total_tokens, COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(completion_tokens), 0) as completion_tokens").
		Where("timestamp >= ? AND total_tokens IS NOT NULL", startTime).
		Group("original_model_name").
		Order("total_tokens DESC").
		Limit(5).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("获取模型用量排名失败：%w", err)
	}

	// 构建响应结果
	modelUsageRankItems := make([]ModelUsageRankItem, 0, len(results))
	for _, result := range results {
		var percentage float64
		if totalTokensSum.Total > 0 {
			percentage = float64(result.TotalTokens) / float64(totalTokensSum.Total)
		}

		modelUsageRankItems = append(modelUsageRankItems, ModelUsageRankItem{
			ModelName:        result.ModelName,
			TotalTokens:      result.TotalTokens,
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
			Percentage:       percentage,
		})
	}

	return &ModelUsageRankResponse{
		TotalTokens: totalTokensSum.Total,
		Models:      modelUsageRankItems,
	}, nil
}

// platformUsageRankResult 定义平台用量排名查询结果结构
type platformUsageRankResult struct {
	PlatformID       uint   `gorm:"column:platform_id"`
	PlatformName     string `gorm:"column:platform_name"`
	TotalTokens      int64  `gorm:"column:total_tokens"`
	PromptTokens     int64  `gorm:"column:prompt_tokens"`
	CompletionTokens int64  `gorm:"column:completion_tokens"`
}

// GetPlatformUsageRank 获取平台用量排名前 5
func (s *service) GetPlatformUsageRank(ctx context.Context, duration time.Duration) (*PlatformUsageRankResponse, error) {
	q := query.Q
	r := q.RequestLog

	// 设置默认时间范围为 24 小时
	if duration == 0 {
		duration = 24 * time.Hour
	}

	startTime := time.Now().Add(-duration)

	// 获取总 Token 数
	var totalTokensSum struct {
		Total int64 `gorm:"column:total"`
	}
	err := r.WithContext(ctx).
		UnderlyingDB().
		Select("COALESCE(SUM(total_tokens), 0) as total").
		Where("timestamp >= ? AND total_tokens IS NOT NULL", startTime).
		Scan(&totalTokensSum).Error
	if err != nil {
		return nil, fmt.Errorf("获取总 Token 数失败：%w", err)
	}

	// 使用数据库聚合查询和 JOIN 获取平台用量统计
	// SELECT
	//   r.platform_id,
	//   p.name as platform_name,
	//   COALESCE(SUM(r.total_tokens), 0) as total_tokens,
	//   COALESCE(SUM(r.prompt_tokens), 0) as prompt_tokens,
	//   COALESCE(SUM(r.completion_tokens), 0) as completion_tokens
	// FROM request_logs r
	// LEFT JOIN platforms p ON r.platform_id = p.id
	// WHERE r.timestamp >= ? AND r.total_tokens IS NOT NULL
	// GROUP BY r.platform_id, p.name
	// ORDER BY total_tokens DESC
	// LIMIT 5
	var results []platformUsageRankResult
	err = r.WithContext(ctx).
		UnderlyingDB().
		Table("request_logs r").
		Select("r.platform_id, p.name as platform_name, COALESCE(SUM(r.total_tokens), 0) as total_tokens, COALESCE(SUM(r.prompt_tokens), 0) as prompt_tokens, COALESCE(SUM(r.completion_tokens), 0) as completion_tokens").
		Joins("LEFT JOIN platforms p ON r.platform_id = p.id").
		Where("r.timestamp >= ? AND r.total_tokens IS NOT NULL", startTime).
		Group("r.platform_id, p.name").
		Order("total_tokens DESC").
		Limit(5).
		Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("获取平台用量排名失败：%w", err)
	}

	// 构建响应结果
	platformUsageRankItems := make([]PlatformUsageRankItem, 0, len(results))
	for _, result := range results {
		var percentage float64
		if totalTokensSum.Total > 0 {
			percentage = float64(result.TotalTokens) / float64(totalTokensSum.Total)
		}

		platformUsageRankItems = append(platformUsageRankItems, PlatformUsageRankItem{
			PlatformName:     result.PlatformName,
			TotalTokens:      result.TotalTokens,
			PromptTokens:     result.PromptTokens,
			CompletionTokens: result.CompletionTokens,
			Percentage:       percentage,
		})
	}

	return &PlatformUsageRankResponse{
		TotalTokens: totalTokensSum.Total,
		Platforms:   platformUsageRankItems,
	}, nil
}
