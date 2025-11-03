package stats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

const (
	// defaultDuration 默认统计时间范围
	defaultDuration = 24 * time.Hour

	// percentileLower 百分位数下限，用于过滤异常低值
	percentileLower = 0.1

	// percentileUpper 百分位数上限，用于过滤异常高值
	percentileUpper = 0.9
)

// GetOverview 实现获取全局概览数据的业务逻辑
//
// 该方法通过并发查询优化性能，获取指定时间范围内的统计数据：
//   - 总请求数和成功率
//   - 平均首字时间（使用百分位数过滤异常值）
//   - Token 使用统计
//
// 参数：
//   - ctx: 上下文，用于控制请求生命周期和传递日志记录器
//   - duration: 统计时间范围，0 值将使用默认的 24 小时
//
// 返回：
//   - *StatsOverviewResponse: 包含所有统计数据的响应对象
//   - error: 如果查询失败则返回错误
func (s *service) GetOverview(ctx context.Context, duration time.Duration) (*StatsOverviewResponse, error) {
	// 设置默认时间范围
	if duration == 0 {
		duration = defaultDuration
	}

	startTime := time.Now().Add(-duration)

	s.logger.InfoContext(ctx, "开始获取全局概览数据",
		"duration", duration,
		"start_time", startTime,
	)

	// 使用 WaitGroup 和 errgroup 模式并发执行独立查询
	var (
		totalRequests         int64
		successRequests       int64
		avgFirstByteTime      float64
		totalPromptTokens     int64
		totalCompletionTokens int64
		totalTokens           int64

		wg      sync.WaitGroup
		errChan = make(chan error, 3)
	)

	// 并发查询 1: 获取请求统计（总数和成功数）
	wg.Add(1)
	go func() {
		defer wg.Done()
		total, success, err := s.getRequestCounts(ctx, startTime)
		if err != nil {
			errChan <- fmt.Errorf("获取请求统计失败：%w", err)
			return
		}
		totalRequests = total
		successRequests = success
	}()

	// 并发查询 2: 计算平均首字时间
	wg.Add(1)
	go func() {
		defer wg.Done()
		avg, err := s.calculateAvgFirstByteTimeWithPercentile(ctx, percentileLower, percentileUpper)
		if err != nil {
			errChan <- fmt.Errorf("计算平均首字时间失败：%w", err)
			return
		}
		avgFirstByteTime = avg
	}()

	// 并发查询 3: 获取 Token 统计（使用数据库聚合）
	wg.Add(1)
	go func() {
		defer wg.Done()
		prompt, completion, total, err := s.getTokenStats(ctx, startTime)
		if err != nil {
			errChan <- fmt.Errorf("获取 Token 统计失败：%w", err)
			return
		}
		totalPromptTokens = prompt
		totalCompletionTokens = completion
		totalTokens = total
	}()

	// 等待所有查询完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误发生
	if err := <-errChan; err != nil {
		s.logger.ErrorContext(ctx, "获取全局概览数据失败", "error", err)
		return nil, err
	}

	// 计算成功率，避免除零错误
	successRate := s.calculateSuccessRate(totalRequests, successRequests)

	response := &StatsOverviewResponse{
		TotalRequests:         totalRequests,
		SuccessRate:           successRate,
		AvgFirstByteTime:      avgFirstByteTime,
		TotalPromptTokens:     totalPromptTokens,
		TotalCompletionTokens: totalCompletionTokens,
		TotalTokens:           totalTokens,
	}

	s.logger.InfoContext(ctx, "成功获取全局概览数据",
		"total_requests", totalRequests,
		"success_rate", successRate,
		"avg_first_byte_time", avgFirstByteTime,
		"total_tokens", totalTokens,
	)

	return response, nil
}

// getRequestCounts 获取总请求数和成功请求数
//
// 使用单次数据库查询同时获取总数和成功数，提高查询效率
func (s *service) getRequestCounts(ctx context.Context, startTime time.Time) (total, success int64, err error) {
	r := query.Q.RequestLog

	// 获取总请求数
	total, err = r.WithContext(ctx).
		Where(r.Timestamp.Gte(startTime)).
		Count()
	if err != nil {
		s.logger.ErrorContext(ctx, "获取总请求数失败", "error", err, "start_time", startTime)
		return 0, 0, err
	}

	// 如果总数为 0，无需查询成功数
	if total == 0 {
		s.logger.DebugContext(ctx, "无请求数据", "start_time", startTime)
		return 0, 0, nil
	}

	// 获取成功请求数
	success, err = r.WithContext(ctx).
		Where(r.Timestamp.Gte(startTime)).
		Where(r.Success.Is(true)).
		Count()
	if err != nil {
		s.logger.ErrorContext(ctx, "获取成功请求数失败", "error", err, "start_time", startTime)
		return 0, 0, err
	}

	s.logger.DebugContext(ctx, "成功获取请求统计",
		"total_requests", total,
		"success_requests", success,
		"start_time", startTime,
	)

	return total, success, nil
}

// getTokenStats 使用数据库聚合函数计算 Token 统计
//
// 相比在应用层遍历所有记录，使用数据库聚合可以：
//  1. 减少网络传输的数据量
//  2. 利用数据库优化的聚合算法
//  3. 降低内存使用
//
// 注意：由于 GORM/Gen 的限制，这里使用原始 SQL 查询
func (s *service) getTokenStats(ctx context.Context, startTime time.Time) (prompt, completion, total int64, err error) {
	r := query.Q.RequestLog

	// 定义聚合结果结构
	type tokenSum struct {
		PromptTokens     *int64
		CompletionTokens *int64
		TotalTokens      *int64
	}

	var result tokenSum

	// 使用 Select 和 Scan 进行聚合查询
	// 注意：需要处理 NULL 值的情况
	err = r.WithContext(ctx).
		Select(
			r.PromptTokens.Sum().As("prompt_tokens"),
			r.CompletionTokens.Sum().As("completion_tokens"),
			r.TotalTokens.Sum().As("total_tokens"),
		).
		Where(r.Timestamp.Gte(startTime)).
		Scan(&result)

	if err != nil {
		s.logger.ErrorContext(ctx, "获取 Token 统计失败", "error", err, "start_time", startTime)
		return 0, 0, 0, err
	}

	// 处理 NULL 值
	if result.PromptTokens != nil {
		prompt = *result.PromptTokens
	}
	if result.CompletionTokens != nil {
		completion = *result.CompletionTokens
	}
	if result.TotalTokens != nil {
		total = *result.TotalTokens
	}

	s.logger.DebugContext(ctx, "成功获取 Token 统计",
		"prompt_tokens", prompt,
		"completion_tokens", completion,
		"total_tokens", total,
		"start_time", startTime,
	)

	return prompt, completion, total, nil
}

// calculateSuccessRate 计算成功率
//
// 提取成功率计算逻辑，便于测试和复用
func (s *service) calculateSuccessRate(total, success int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(success) / float64(total)
}
