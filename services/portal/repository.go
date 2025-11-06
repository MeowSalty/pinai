package portal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/portal/request"
	"github.com/MeowSalty/portal/routing"
	"github.com/MeowSalty/portal/routing/health"
)

// Repository 数据仓库实现
//
// 提供 portal 所需的数据访问接口
type Repository struct {
	logger *slog.Logger
}

// GetModelByID 根据 ID 获取模型信息
func (r *Repository) GetModelByID(ctx context.Context, id uint) (routing.Model, error) {
	repoLogger := r.logger.WithGroup("model_repository")
	repoLogger.Debug("根据 ID 获取模型信息", "model_id", id)

	q := query.Q

	// 预加载 APIKeys 关联数据
	dbModel, err := q.WithContext(ctx).Model.Preload(q.Model.APIKeys).Where(q.Model.ID.Eq(id)).First()
	if err != nil {
		repoLogger.Error("获取模型失败", "error", err, "model_id", id)
		return routing.Model{}, fmt.Errorf("获取模型失败：%w", err)
	}

	// 转换 APIKeys
	apiKeys := make([]routing.APIKey, len(dbModel.APIKeys))
	for i, dbKey := range dbModel.APIKeys {
		apiKeys[i] = routing.APIKey{
			ID:    dbKey.ID,
			Value: dbKey.Value,
		}
	}

	// 转换为 routing.Model 类型
	model := routing.Model{
		ID:         dbModel.ID,
		PlatformID: dbModel.PlatformID,
		Name:       dbModel.Name,
		Alias:      dbModel.Alias,
		APIKeys:    apiKeys,
	}

	repoLogger.Debug("模型信息获取成功", "model_id", id, "model_name", model.Name, "api_keys_count", len(apiKeys))
	return model, nil
}

// FindModelsByNameOrAlias 根据名称或别名查找模型
func (r *Repository) FindModelsByNameOrAlias(ctx context.Context, name string) ([]routing.Model, error) {
	repoLogger := r.logger.WithGroup("model_repository")
	repoLogger.Debug("根据名称或别名查找模型", "name", name)

	q := query.Q

	// 使用 GORM 查询模型（按名称或别名查找），预加载 APIKeys 关联数据
	dbModels, err := q.WithContext(ctx).Model.Preload(q.Model.APIKeys).Where(
		q.Model.Name.Eq(name),
	).Or(
		q.Model.Alias_.Eq(name),
	).Find()

	if err != nil {
		repoLogger.Error("查询模型失败", "error", err, "name", name)
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}

	// 转换为 routing.Model 类型
	models := make([]routing.Model, len(dbModels))
	for i, dbModel := range dbModels {
		// 转换 APIKeys
		apiKeys := make([]routing.APIKey, len(dbModel.APIKeys))
		for j, dbKey := range dbModel.APIKeys {
			apiKeys[j] = routing.APIKey{
				ID:    dbKey.ID,
				Value: dbKey.Value,
			}
		}

		models[i] = routing.Model{
			ID:         dbModel.ID,
			PlatformID: dbModel.PlatformID,
			Name:       dbModel.Name,
			Alias:      dbModel.Alias,
			APIKeys:    apiKeys,
		}
	}

	repoLogger.Debug("模型查询成功", "name", name, "found_count", len(models))
	return models, nil
}

// GetPlatformByID 根据 ID 获取平台信息
func (r *Repository) GetPlatformByID(ctx context.Context, id uint) (*routing.Platform, error) {
	repoLogger := r.logger.WithGroup("platform_repository")
	repoLogger.Debug("根据 ID 获取平台信息", "platform_id", id)

	q := query.Q

	dbPlatform, err := q.WithContext(ctx).Platform.Where(q.Platform.ID.Eq(id)).First()
	if err != nil {
		repoLogger.Error("获取平台失败", "error", err, "platform_id", id)
		return nil, fmt.Errorf("获取平台失败：%w", err)
	}

	// 转换为 core.Platform 类型
	platform := &routing.Platform{
		ID:      dbPlatform.ID,
		Name:    dbPlatform.Name,
		Format:  dbPlatform.Format,
		BaseURL: dbPlatform.BaseURL,
		RateLimit: routing.RateLimitConfig{
			RPM: dbPlatform.RateLimit.RPM,
			TPM: dbPlatform.RateLimit.TPM,
		},
	}

	repoLogger.Debug("平台信息获取成功", "platform_id", id, "platform_name", platform.Name)
	return platform, nil
}

// GetAllAPIKeysByPlatformID 根据平台 ID 获取所有 API 密钥
func (r *Repository) GetAllAPIKeysByPlatformID(ctx context.Context, platformID uint) ([]*routing.APIKey, error) {
	repoLogger := r.logger.WithGroup("api_key_repository")
	repoLogger.Debug("根据平台 ID 获取所有 API 密钥", "platform_id", platformID)

	q := query.Q

	dbKeys, err := q.WithContext(ctx).APIKey.Where(q.APIKey.PlatformID.Eq(platformID)).Find()
	if err != nil {
		repoLogger.Error("获取 API 密钥失败", "error", err, "platform_id", platformID)
		return nil, fmt.Errorf("获取 API 密钥失败：%w", err)
	}

	// 转换为 core.APIKey 类型
	keys := make([]*routing.APIKey, len(dbKeys))
	for i, dbKey := range dbKeys {
		keys[i] = &routing.APIKey{
			ID:    dbKey.ID,
			Value: dbKey.Value,
		}
	}

	repoLogger.Debug("API 密钥获取成功", "platform_id", platformID, "key_count", len(keys))
	return keys, nil
}

// GetAllHealth 获取所有健康状态
func (r *Repository) GetAllHealth(ctx context.Context) ([]*health.Health, error) {
	repoLogger := r.logger.WithGroup("health_repository")
	repoLogger.Debug("获取所有健康状态")

	q := query.Q

	dbHealths, err := q.WithContext(ctx).Health.Find()
	if err != nil {
		repoLogger.Error("获取健康状态失败", "error", err)
		return nil, fmt.Errorf("获取健康状态失败：%w", err)
	}

	// 转换为 core.Health 类型
	healths := make([]*health.Health, len(dbHealths))
	for i, dbHealth := range dbHealths {
		healths[i] = &health.Health{
			ID:              dbHealth.ID,
			ResourceType:    health.ResourceType(dbHealth.ResourceType),
			ResourceID:      dbHealth.ResourceID,
			Status:          health.HealthStatus(dbHealth.Status),
			RetryCount:      dbHealth.RetryCount,
			NextAvailableAt: dbHealth.NextAvailableAt,
			BackoffDuration: dbHealth.BackoffDuration,
			LastError:       dbHealth.LastError,
			LastErrorCode:   dbHealth.LastErrorCode,
			LastCheckAt:     dbHealth.LastCheckAt,
			LastSuccessAt:   dbHealth.LastSuccessAt,
			SuccessCount:    dbHealth.SuccessCount,
			ErrorCount:      dbHealth.ErrorCount,
			CreatedAt:       dbHealth.CreatedAt,
			UpdatedAt:       dbHealth.UpdatedAt,
		}
	}

	repoLogger.Debug("健康状态获取成功", "count", len(healths))
	return healths, nil
}

// BatchUpdateHealth 批量更新健康状态
func (r *Repository) BatchUpdateHealth(ctx context.Context, statuses []health.Health) error {
	repoLogger := r.logger.WithGroup("health_repository")
	repoLogger.Info("开始批量更新健康状态", "count", len(statuses))

	q := query.Q

	// 开启事务
	repoLogger.Debug("开启数据库事务")
	tx := q.Begin()
	defer func() {
		if r := recover(); r != nil {
			repoLogger.Error("事务执行过程中发生 panic，执行回滚", "recover", r)
			tx.Rollback()
		}
	}()

	for i, status := range statuses {
		repoLogger.Debug("更新健康状态",
			"index", i,
			"resource_id", status.ResourceID,
			"resource_type", status.ResourceType,
			"status", status.Status)

		// 转换为数据库类型并更新或创建
		dbHealth := &types.Health{
			ID:              status.ID,
			ResourceType:    types.ResourceType(status.ResourceType),
			ResourceID:      status.ResourceID,
			Status:          types.HealthStatus(status.Status),
			RetryCount:      status.RetryCount,
			NextAvailableAt: status.NextAvailableAt,
			BackoffDuration: status.BackoffDuration,
			LastError:       status.LastError,
			LastErrorCode:   status.LastErrorCode,
			LastCheckAt:     status.LastCheckAt,
			LastSuccessAt:   status.LastSuccessAt,
			SuccessCount:    status.SuccessCount,
			ErrorCount:      status.ErrorCount,
		}

		// 使用 Upsert 操作
		if err := tx.WithContext(ctx).Health.Save(dbHealth); err != nil {
			repoLogger.Error("更新健康状态失败，执行事务回滚",
				"error", err,
				"resource_id", status.ResourceID,
				"resource_type", status.ResourceType)
			tx.Rollback()
			return fmt.Errorf("批量更新健康状态失败：%w", err)
		}
	}

	// 检查提交事务是否有错误
	repoLogger.Debug("提交事务")
	if err := tx.Commit(); err != nil {
		repoLogger.Error("提交事务失败", "error", err)
		return fmt.Errorf("提交事务失败：%w", err)
	}

	repoLogger.Info("批量更新健康状态成功", "count", len(statuses))
	return nil
}

// CreateRequestLog 创建请求日志
//
// 保存请求日志到数据库
//
// 参数：
//   - ctx: 上下文
//   - log: 请求日志
//
// 返回值：
//   - error: 错误信息
func (r *Repository) CreateRequestLog(ctx context.Context, log *request.RequestLog) error {
	repoLogger := r.logger.WithGroup("log_repository")

	// 记录审计日志
	repoLogger.Info("创建请求日志",
		"request_id", log.ID,
		"request_type", log.RequestType,
		"model_name", log.ModelName,
		"original_model_name", log.OriginalModelName,
		"platform_id", log.PlatformID,
		"success", log.Success,
		"duration_ms", log.Duration.Milliseconds(),
		"total_tokens", log.TotalTokens)

	// 将 request.RequestLog 转换为数据库类型
	dbLog := &types.RequestLog{
		ID:                log.ID,
		Timestamp:         log.Timestamp,
		RequestType:       log.RequestType,
		ModelName:         log.ModelName,
		OriginalModelName: log.OriginalModelName,
		PlatformID:        log.PlatformID,
		APIKeyID:          log.APIKeyID,
		ModelID:           log.ModelID,
		Duration:          log.Duration.Microseconds(),
		Success:           log.Success,
		ErrorMsg:          log.ErrorMsg,
		PromptTokens:      log.PromptTokens,
		CompletionTokens:  log.CompletionTokens,
		TotalTokens:       log.TotalTokens,
	}
	if log.FirstByteTime != nil {
		firstByteTime := log.FirstByteTime.Microseconds()
		dbLog.FirstByteTime = &firstByteTime
	}

	// 保存到数据库
	repoLogger.Debug("保存请求日志到数据库")
	err := query.Q.WithContext(ctx).RequestLog.Create(dbLog)
	if err != nil {
		repoLogger.Error("保存请求日志失败",
			"error", err,
			"request_id", log.ID,
			"model_name", log.ModelName)
		return fmt.Errorf("保存请求日志失败：%w", err)
	}

	repoLogger.Debug("请求日志保存成功", "request_id", log.ID)
	return nil
}
