package provider

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/internal/app/health"
	"gorm.io/gorm"
)

// modelControlQueryRepository 是基于 database/query 的模型控制面仓储实现。
type modelControlQueryRepository struct {
	healthStorage *health.Storage
	logger        *slog.Logger
}

// NewModelControlQueryRepository 创建模型控制面仓储实现。
func NewModelControlQueryRepository(healthStorage *health.Storage, logger *slog.Logger) ModelControlRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &modelControlQueryRepository{
		healthStorage: healthStorage,
		logger:        logger.WithGroup("model_control_repo"),
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

// ListModelsByIDs 按 ID 列表查询模型。
func (r *modelControlQueryRepository) ListModelsByIDs(ctx context.Context, modelIDs []uint) ([]*types.Model, error) {
	if len(modelIDs) == 0 {
		return []*types.Model{}, nil
	}

	q := queryFromContextOrDefault(ctx)
	models, err := q.Model.WithContext(ctx).
		Where(q.Model.ID.In(modelIDs...)).
		Find()
	if err != nil {
		r.logger.Error("批量查询模型失败", slog.Any("model_ids", modelIDs), slog.Any("error", err))
		return nil, fmt.Errorf("批量查询模型失败：%w", err)
	}

	return models, nil
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

// GetModel 查询模型详情。
func (r *modelControlQueryRepository) GetModel(ctx context.Context, modelID uint) (*types.Model, error) {
	q := queryFromContextOrDefault(ctx)
	model, err := q.Model.WithContext(ctx).Where(q.Model.ID.Eq(modelID)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的模型：%w", modelID, ErrResourceNotFound)
		}
		r.logger.Error("查询模型失败", slog.Uint64("model_id", uint64(modelID)), slog.Any("error", err))
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}

	return model, nil
}

// GetModelWithAPIKeys 查询模型详情并预加载关联密钥。
func (r *modelControlQueryRepository) GetModelWithAPIKeys(ctx context.Context, modelID uint) (*types.Model, error) {
	q := queryFromContextOrDefault(ctx)
	model, err := q.Model.WithContext(ctx).
		Preload(q.Model.APIKeys).
		Where(q.Model.ID.Eq(modelID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的模型：%w", modelID, ErrResourceNotFound)
		}
		r.logger.Error("查询模型失败", slog.Uint64("model_id", uint64(modelID)), slog.Any("error", err))
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}

	return model, nil
}

// ReplaceModelAPIKeys 替换模型与密钥的关联关系。
func (r *modelControlQueryRepository) ReplaceModelAPIKeys(ctx context.Context, modelID uint, apiKeys []*types.APIKey) error {
	if len(apiKeys) == 0 {
		return nil
	}

	q := queryFromContextOrDefault(ctx)
	if err := q.Model.APIKeys.Model(&types.Model{ID: modelID}).Replace(apiKeys...); err != nil {
		r.logger.Error("更新模型与密钥的关联关系失败",
			slog.Uint64("model_id", uint64(modelID)),
			slog.Any("error", err))
		return fmt.Errorf("更新模型与密钥的关联关系失败：%w", err)
	}

	return nil
}

// UpdateModelFields 更新模型字段。
func (r *modelControlQueryRepository) UpdateModelFields(ctx context.Context, modelID uint, updates map[string]interface{}) (int64, error) {
	if len(updates) == 0 {
		return 0, nil
	}

	q := queryFromContextOrDefault(ctx)
	result, err := q.Model.WithContext(ctx).Where(q.Model.ID.Eq(modelID)).Updates(updates)
	if err != nil {
		r.logger.Error("更新模型字段失败",
			slog.Uint64("model_id", uint64(modelID)),
			slog.Any("updates", updates),
			slog.Any("error", err))
		return 0, fmt.Errorf("更新模型字段失败：%w", err)
	}

	return result.RowsAffected, nil
}

// ListAPIKeysByModel 查询模型关联的密钥列表。
func (r *modelControlQueryRepository) ListAPIKeysByModel(ctx context.Context, modelID uint) ([]*types.APIKey, error) {
	q := queryFromContextOrDefault(ctx)
	apiKeys, err := q.Model.APIKeys.Model(&types.Model{ID: modelID}).Find()
	if err != nil {
		r.logger.Error("查询模型关联的密钥失败",
			slog.Uint64("model_id", uint64(modelID)),
			slog.Any("error", err))
		return nil, fmt.Errorf("查询模型关联的密钥失败：%w", err)
	}

	return apiKeys, nil
}

// ClearModelAPIKeyRelations 清理模型与密钥的关联关系。
func (r *modelControlQueryRepository) ClearModelAPIKeyRelations(ctx context.Context, modelID uint) error {
	q := queryFromContextOrDefault(ctx)
	if err := q.Model.APIKeys.Model(&types.Model{ID: modelID}).Clear(); err != nil {
		r.logger.Error("清理模型与密钥的关联关系失败",
			slog.Uint64("model_id", uint64(modelID)),
			slog.Any("error", err))
		return fmt.Errorf("清理模型与密钥的关联关系失败：%w", err)
	}

	return nil
}

// AppendModelAPIKeys 恢复模型与密钥的关联关系。
func (r *modelControlQueryRepository) AppendModelAPIKeys(ctx context.Context, modelID uint, apiKeys []*types.APIKey) error {
	if len(apiKeys) == 0 {
		return nil
	}

	q := queryFromContextOrDefault(ctx)
	if err := q.Model.APIKeys.Model(&types.Model{ID: modelID}).Append(apiKeys...); err != nil {
		r.logger.Error("恢复模型与密钥的关联关系失败",
			slog.Uint64("model_id", uint64(modelID)),
			slog.Any("error", err))
		return fmt.Errorf("恢复模型与密钥的关联关系失败：%w", err)
	}

	return nil
}

// DeleteModelByID 删除指定模型。
func (r *modelControlQueryRepository) DeleteModelByID(ctx context.Context, modelID uint) (int64, error) {
	q := queryFromContextOrDefault(ctx)
	result, err := q.Model.WithContext(ctx).Where(q.Model.ID.Eq(modelID)).Delete()
	if err != nil {
		r.logger.Error("删除模型失败",
			slog.Uint64("model_id", uint64(modelID)),
			slog.Any("error", err))
		return 0, fmt.Errorf("删除模型失败：%w", err)
	}

	return result.RowsAffected, nil
}

// DeleteModelsByIDs 按 ID 列表批量删除模型。
func (r *modelControlQueryRepository) DeleteModelsByIDs(ctx context.Context, modelIDs []uint) (int64, error) {
	if len(modelIDs) == 0 {
		return 0, nil
	}

	q := queryFromContextOrDefault(ctx)
	result, err := q.Model.WithContext(ctx).Where(q.Model.ID.In(modelIDs...)).Delete()
	if err != nil {
		r.logger.Error("批量删除模型失败",
			slog.Any("model_ids", modelIDs),
			slog.Any("error", err))
		return 0, fmt.Errorf("批量删除模型失败：%w", err)
	}

	return result.RowsAffected, nil
}

// EnableModelHealth 启用模型健康状态（删除健康记录，恢复为 Unknown）。
func (r *modelControlQueryRepository) EnableModelHealth(ctx context.Context, modelID uint) error {
	if r.healthStorage == nil {
		return fmt.Errorf("启用模型健康状态失败：健康状态存储未初始化")
	}

	if err := r.healthStorage.Delete(types.ResourceTypeModel, modelID); err != nil {
		r.logger.Error("启用模型健康状态失败", slog.Uint64("model_id", uint64(modelID)), slog.Any("error", err))
		return fmt.Errorf("启用模型健康状态失败：%w", err)
	}

	return nil
}

// DisableModelHealth 禁用模型健康状态（写入 Unavailable 状态）。
func (r *modelControlQueryRepository) DisableModelHealth(ctx context.Context, modelID uint) error {
	if r.healthStorage == nil {
		return fmt.Errorf("禁用模型健康状态失败：健康状态存储未初始化")
	}

	now := time.Now()
	healthRecord := &types.Health{
		ResourceType:    types.ResourceTypeModel,
		ResourceID:      modelID,
		Status:          types.HealthStatusUnavailable,
		LastError:       "手动禁用",
		LastCheckAt:     now,
		RetryCount:      0,
		BackoffDuration: 0,
	}

	if err := r.healthStorage.Set(healthRecord); err != nil {
		r.logger.Error("禁用模型健康状态失败", slog.Uint64("model_id", uint64(modelID)), slog.Any("error", err))
		return fmt.Errorf("禁用模型健康状态失败：%w", err)
	}

	return nil
}
