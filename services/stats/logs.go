package stats

import (
	"context"
	"fmt"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// ListRequestLogs 实现获取请求状态列表的业务逻辑
func (s *service) ListRequestLogs(ctx context.Context, opts ListRequestLogsOptions) ([]*types.RequestLog, int64, error) {
	s.logger.InfoContext(ctx, "开始获取请求日志列表",
		"page", opts.Page,
		"page_size", opts.PageSize,
	)

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
		s.logger.ErrorContext(ctx, "获取请求日志列表失败", "error", err)
		return nil, 0, fmt.Errorf("获取请求状态列表失败：%w", err)
	}

	s.logger.InfoContext(ctx, "成功获取请求日志列表",
		"count", count,
		"result_size", len(result),
	)

	return result, count, nil
}
