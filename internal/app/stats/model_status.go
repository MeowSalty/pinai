package stats

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/MeowSalty/pinai/database/query"
)

type modelStatusLogRow struct {
	Timestamp         time.Time `gorm:"column:timestamp"`
	Success           bool      `gorm:"column:success"`
	OriginalModelName string    `gorm:"column:original_model_name"`
}

type modelStatusAgg struct {
	item *ModelStatusItem
}

// GetModelStatus 获取模型状态监控数据。
func (s *service) GetModelStatus(ctx context.Context, trendRange TrendRange, modelName *string) (*ModelStatusResponse, error) {
	start := time.Now()
	logger := s.logger.With("operation", "get_model_status")

	if trendRange == "" {
		trendRange = TrendRange24h
	}

	cfg, ok := trendRangeConfigs[trendRange]
	if !ok {
		return nil, fmt.Errorf("无效的时间范围参数，可选值：24h, 7d, 30d")
	}

	now := time.Now()
	bucketEnd := ceilToHour(now)
	bucketStart := bucketEnd.Add(-cfg.Granularity * time.Duration(cfg.Points))

	logger.DebugContext(ctx, "开始聚合模型状态数据",
		"range", trendRange,
		"granularity", cfg.Label,
		"bucket_start", bucketStart,
		"bucket_end", bucketEnd,
	)

	db := query.Q.RequestLog.WithContext(ctx).
		UnderlyingDB().
		Table("request_logs").
		Select("timestamp, success, original_model_name").
		Where("timestamp >= ? AND timestamp < ?", bucketStart, bucketEnd)

	if modelName != nil && *modelName != "" {
		db = db.Where("original_model_name = ?", *modelName)
		logger = logger.With("model_name", *modelName)
	}

	var rows []modelStatusLogRow
	if err := db.Order("timestamp ASC").Scan(&rows).Error; err != nil {
		logger.ErrorContext(ctx, "查询模型状态原始数据失败",
			"error", err,
			"error_type", "database_error",
			"latency_ms", time.Since(start).Milliseconds(),
		)
		return nil, fmt.Errorf("查询模型状态数据失败：%w", err)
	}

	aggMap := make(map[string]*modelStatusAgg)

	for _, row := range rows {
		name := row.OriginalModelName
		if name == "" {
			name = "unknown"
		}

		agg, exists := aggMap[name]
		if !exists {
			points := make([]ModelStatusPoint, cfg.Points)
			for i := 0; i < cfg.Points; i++ {
				points[i] = ModelStatusPoint{
					Timestamp: bucketStart.Add(cfg.Granularity * time.Duration(i+1)),
				}
			}

			agg = &modelStatusAgg{
				item: &ModelStatusItem{
					ModelName: name,
					Points:    points,
				},
			}
			aggMap[name] = agg
		}

		agg.item.TotalRequests++
		if row.Success {
			agg.item.SuccessCount++
		}

		idx := int(row.Timestamp.Sub(bucketStart) / cfg.Granularity)
		if idx >= 0 && idx < len(agg.item.Points) {
			agg.item.Points[idx].RequestCount++
			if row.Success {
				agg.item.Points[idx].SuccessCount++
			}
		}
	}

	models := make([]ModelStatusItem, 0, len(aggMap))
	for _, agg := range aggMap {
		models = append(models, *agg.item)
	}

	sort.Slice(models, func(i, j int) bool {
		if models[i].TotalRequests == models[j].TotalRequests {
			return models[i].ModelName < models[j].ModelName
		}
		return models[i].TotalRequests > models[j].TotalRequests
	})

	resp := &ModelStatusResponse{
		Range:       string(trendRange),
		Granularity: cfg.Label,
		WindowStart: bucketStart,
		WindowEnd:   bucketEnd,
		Models:      models,
	}

	logger.DebugContext(ctx, "成功聚合模型状态数据",
		"rows", len(rows),
		"models", len(models),
		"latency_ms", time.Since(start).Milliseconds(),
	)

	return resp, nil
}
