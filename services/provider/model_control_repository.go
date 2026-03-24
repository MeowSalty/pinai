package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// modelControlQueryRepository 是基于 database/query 的模型控制面仓储实现。
type modelControlQueryRepository struct {
	logger *slog.Logger
}

// NewModelControlQueryRepository 创建模型控制面仓储实现。
func NewModelControlQueryRepository(logger *slog.Logger) ModelControlRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &modelControlQueryRepository{
		logger: logger.WithGroup("model_control_repo"),
	}
}

// ExistsPlatform 检查平台是否存在。
func (r *modelControlQueryRepository) ExistsPlatform(ctx context.Context, platformID uint) (bool, error) {
	q := queryFromContextOrDefault(ctx)
	count, err := q.Platform.WithContext(ctx).Where(q.Platform.ID.Eq(platformID)).Count()
	if err != nil {
		r.logger.Error("查询平台是否存在失败", slog.Uint64("platform_id", uint64(platformID)), slog.Any("error", err))
		return false, fmt.Errorf("查询平台是否存在失败：%w", err)
	}

	return count > 0, nil
}

// ListAPIKeysByPlatformAndIDs 查询指定平台下给定 ID 列表的密钥。
func (r *modelControlQueryRepository) ListAPIKeysByPlatformAndIDs(ctx context.Context, platformID uint, apiKeyIDs []uint) ([]*types.APIKey, error) {
	if len(apiKeyIDs) == 0 {
		return []*types.APIKey{}, nil
	}

	q := queryFromContextOrDefault(ctx)
	apiKeys, err := q.APIKey.WithContext(ctx).
		Where(q.APIKey.ID.In(apiKeyIDs...), q.APIKey.PlatformID.Eq(platformID)).
		Find()
	if err != nil {
		r.logger.Error("查询平台密钥失败",
			slog.Uint64("platform_id", uint64(platformID)),
			slog.Any("api_key_ids", apiKeyIDs),
			slog.Any("error", err))
		return nil, fmt.Errorf("查询平台密钥失败：%w", err)
	}

	return apiKeys, nil
}

// CreateModel 创建模型记录及其关联关系。
func (r *modelControlQueryRepository) CreateModel(ctx context.Context, model *types.Model) error {
	if model == nil {
		return fmt.Errorf("创建模型失败：模型参数不能为空")
	}

	q := queryFromContextOrDefault(ctx)
	if err := q.Model.WithContext(ctx).Create(model); err != nil {
		r.logger.Error("创建模型失败",
			slog.String("model_name", model.Name),
			slog.Uint64("platform_id", uint64(model.PlatformID)),
			slog.Any("error", err))
		return fmt.Errorf("创建模型失败：%w", err)
	}

	return nil
}
