package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// keyControlQueryRepository 是基于 database/query 的密钥控制面仓储实现。
type keyControlQueryRepository struct {
	logger *slog.Logger
}

// NewKeyControlQueryRepository 创建密钥控制面仓储实现。
func NewKeyControlQueryRepository(logger *slog.Logger) KeyControlRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &keyControlQueryRepository{
		logger: logger.WithGroup("key_control_repo"),
	}
}

// ExistsPlatform 检查平台是否存在。
func (r *keyControlQueryRepository) ExistsPlatform(ctx context.Context, platformID uint) (bool, error) {
	q := queryFromContextOrDefault(ctx)
	count, err := q.Platform.WithContext(ctx).Where(q.Platform.ID.Eq(platformID)).Count()
	if err != nil {
		r.logger.Error("查询平台是否存在失败", slog.Uint64("platform_id", uint64(platformID)), slog.Any("error", err))
		return false, fmt.Errorf("查询平台是否存在失败：%w", err)
	}

	return count > 0, nil
}

// CreateAPIKey 创建密钥。
func (r *keyControlQueryRepository) CreateAPIKey(ctx context.Context, key *types.APIKey) error {
	if key == nil {
		return fmt.Errorf("创建 API 密钥失败：密钥参数不能为空")
	}

	q := queryFromContextOrDefault(ctx)
	if err := q.APIKey.WithContext(ctx).Create(key); err != nil {
		r.logger.Error("创建 API 密钥失败",
			slog.Uint64("platform_id", uint64(key.PlatformID)),
			slog.Any("error", err))
		return fmt.Errorf("创建 API 密钥失败：%w", err)
	}

	return nil
}
