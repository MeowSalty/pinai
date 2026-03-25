package provider

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/health"
	"gorm.io/gorm"
)

// keyControlQueryRepository 是基于 database/query 的密钥控制面仓储实现。
type keyControlQueryRepository struct {
	healthStorage *health.Storage
	logger        *slog.Logger
}

// NewKeyControlQueryRepository 创建密钥控制面仓储实现。
func NewKeyControlQueryRepository(healthStorage *health.Storage, logger *slog.Logger) KeyControlRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &keyControlQueryRepository{
		healthStorage: healthStorage,
		logger:        logger.WithGroup("key_control_repo"),
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

// GetAPIKey 查询密钥详情。
func (r *keyControlQueryRepository) GetAPIKey(ctx context.Context, keyID uint) (*types.APIKey, error) {
	q := queryFromContextOrDefault(ctx)
	apiKey, err := q.APIKey.WithContext(ctx).Where(q.APIKey.ID.Eq(keyID)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的 API 密钥：%w", keyID, ErrResourceNotFound)
		}
		r.logger.Error("查询 API 密钥失败", slog.Uint64("key_id", uint64(keyID)), slog.Any("error", err))
		return nil, fmt.Errorf("查询 API 密钥失败：%w", err)
	}

	return apiKey, nil
}

// UpdateAPIKey 更新密钥字段并返回影响行数。
func (r *keyControlQueryRepository) UpdateAPIKey(ctx context.Context, keyID uint, updates types.APIKey) (int64, error) {
	q := queryFromContextOrDefault(ctx)
	result, err := q.APIKey.WithContext(ctx).Where(q.APIKey.ID.Eq(keyID)).Updates(updates)
	if err != nil {
		r.logger.Error("更新 API 密钥失败", slog.Uint64("key_id", uint64(keyID)), slog.Any("error", err))
		return 0, fmt.Errorf("更新 API 密钥失败：%w", err)
	}

	return result.RowsAffected, nil
}

// ListModelsByAPIKey 查询密钥关联的模型列表。
func (r *keyControlQueryRepository) ListModelsByAPIKey(ctx context.Context, keyID uint) ([]*types.Model, error) {
	q := queryFromContextOrDefault(ctx)
	models, err := q.APIKey.Models.Model(&types.APIKey{ID: keyID}).Find()
	if err != nil {
		r.logger.Error("查询密钥关联模型失败", slog.Uint64("key_id", uint64(keyID)), slog.Any("error", err))
		return nil, fmt.Errorf("查询密钥关联模型失败：%w", err)
	}

	return models, nil
}

// ClearAPIKeyModelRelations 清理密钥与模型的关联关系。
func (r *keyControlQueryRepository) ClearAPIKeyModelRelations(ctx context.Context, keyID uint) error {
	q := queryFromContextOrDefault(ctx)
	if err := q.APIKey.Models.Model(&types.APIKey{ID: keyID}).Clear(); err != nil {
		r.logger.Error("清理密钥与模型关联关系失败", slog.Uint64("key_id", uint64(keyID)), slog.Any("error", err))
		return fmt.Errorf("清理密钥与模型关联关系失败：%w", err)
	}

	return nil
}

// AppendAPIKeyModels 恢复密钥与模型关联关系。
func (r *keyControlQueryRepository) AppendAPIKeyModels(ctx context.Context, keyID uint, models []*types.Model) error {
	if len(models) == 0 {
		return nil
	}

	q := queryFromContextOrDefault(ctx)
	if err := q.APIKey.Models.Model(&types.APIKey{ID: keyID}).Append(models...); err != nil {
		r.logger.Error("恢复密钥与模型关联关系失败", slog.Uint64("key_id", uint64(keyID)), slog.Any("error", err))
		return fmt.Errorf("恢复密钥与模型关联关系失败：%w", err)
	}

	return nil
}

// DeleteAPIKeyByID 删除指定密钥并返回影响行数。
func (r *keyControlQueryRepository) DeleteAPIKeyByID(ctx context.Context, keyID uint) (int64, error) {
	q := queryFromContextOrDefault(ctx)
	result, err := q.APIKey.WithContext(ctx).Where(q.APIKey.ID.Eq(keyID)).Delete()
	if err != nil {
		r.logger.Error("删除 API 密钥失败", slog.Uint64("key_id", uint64(keyID)), slog.Any("error", err))
		return 0, fmt.Errorf("删除 API 密钥失败：%w", err)
	}

	return result.RowsAffected, nil
}

// EnableAPIKeyHealth 启用密钥健康状态（删除健康记录，恢复为 Unknown）。
func (r *keyControlQueryRepository) EnableAPIKeyHealth(ctx context.Context, keyID uint) error {
	if r.healthStorage == nil {
		return fmt.Errorf("启用密钥健康状态失败：健康状态存储未初始化")
	}

	if err := r.healthStorage.Delete(types.ResourceTypeAPIKey, keyID); err != nil {
		r.logger.Error("启用密钥健康状态失败", slog.Uint64("key_id", uint64(keyID)), slog.Any("error", err))
		return fmt.Errorf("启用密钥健康状态失败：%w", err)
	}

	return nil
}

// DisableAPIKeyHealth 禁用密钥健康状态（写入 Unavailable 状态）。
func (r *keyControlQueryRepository) DisableAPIKeyHealth(ctx context.Context, keyID uint) error {
	if r.healthStorage == nil {
		return fmt.Errorf("禁用密钥健康状态失败：健康状态存储未初始化")
	}

	now := time.Now()
	healthRecord := &types.Health{
		ResourceType:    types.ResourceTypeAPIKey,
		ResourceID:      keyID,
		Status:          types.HealthStatusUnavailable,
		LastError:       "手动禁用",
		LastCheckAt:     now,
		RetryCount:      0,
		BackoffDuration: 0,
	}

	if err := r.healthStorage.Set(healthRecord); err != nil {
		r.logger.Error("禁用密钥健康状态失败", slog.Uint64("key_id", uint64(keyID)), slog.Any("error", err))
		return fmt.Errorf("禁用密钥健康状态失败：%w", err)
	}

	return nil
}
