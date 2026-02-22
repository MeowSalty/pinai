package portal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/portal/request"
	"github.com/MeowSalty/portal/routing"
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

// FindModelsWithDefaultEndpoint 通过模型名称查找，返回带有平台和默认端点的完整信息
//
// 如果平台没有默认端点，返回错误。
//
// 参数：
//   - ctx: 上下文
//   - name: 模型名称或别名
//
// 返回值：
//   - []routing.ModelWithEndpoint: 匹配的模型列表（包含平台和默认端点信息）
//   - error: 错误信息
func (r *Repository) FindModelsWithDefaultEndpoint(ctx context.Context, name string) ([]routing.ModelWithEndpoint, error) {
	repoLogger := r.logger.WithGroup("model_repository")
	repoLogger.Debug("根据名称或别名查找模型（含默认端点）", "name", name)

	q := query.Q
	db := q.WithContext(ctx).Model

	// 使用 JOIN 查询模型、平台和默认端点
	dbModels, err := db.
		Join(q.Endpoint, q.Model.PlatformID.EqCol(q.Endpoint.PlatformID)).
		Where(q.Endpoint.IsDefault.Is(true)).
		Where(
			db.Where(q.Model.Name.Eq(name)).Or(q.Model.Alias_.Eq(name)),
		).
		Preload(
			q.Model.APIKeys,
			q.Model.Platform,
			q.Model.Platform.Endpoints.On(
				q.Endpoint.IsDefault.Is(true),
			),
		).
		Find()

	if err != nil {
		repoLogger.Error("查询模型失败", "error", err, "name", name)
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}

	if len(dbModels) == 0 {
		repoLogger.Debug("未找到匹配的模型", "name", name)
		return nil, nil
	}

	// 转换为 routing.ModelWithEndpoint 类型
	modelsWithEndpoint := make([]routing.ModelWithEndpoint, 0, len(dbModels))
	for _, model := range dbModels {
		// 转换 APIKeys
		apiKeys := make([]routing.APIKey, len(model.APIKeys))
		for j, dbKey := range model.APIKeys {
			apiKeys[j] = routing.APIKey{
				ID:    dbKey.ID,
				Value: dbKey.Value,
			}
		}

		// 转换 CustomHeaders
		endpointCustomHeaders := copyStringMap(model.Platform.Endpoints[0].CustomHeaders)

		platformCustomHeaders := make(map[string]string)
		// 注意：当前数据库模型 Platform 没有 CustomHeaders 字段，使用空 map

		modelsWithEndpoint = append(modelsWithEndpoint, routing.ModelWithEndpoint{
			Model: routing.Model{
				ID:         model.ID,
				PlatformID: model.PlatformID,
				Name:       model.Name,
				Alias:      model.Alias,
				APIKeys:    apiKeys,
			},
			Platform: routing.Platform{
				ID:            model.Platform.ID,
				Name:          model.Platform.Name,
				BaseURL:       model.Platform.BaseURL,
				RateLimit:     routing.RateLimitConfig{RPM: model.Platform.RateLimit.RPM, TPM: model.Platform.RateLimit.TPM},
				CustomHeaders: platformCustomHeaders,
			},
			Endpoint: routing.Endpoint{
				ID:              model.Platform.Endpoints[0].ID,
				EndpointType:    model.Platform.Endpoints[0].EndpointType,
				EndpointVariant: model.Platform.Endpoints[0].EndpointVariant,
				Path:            model.Platform.Endpoints[0].Path,
				CustomHeaders:   endpointCustomHeaders,
			},
		})
	}

	repoLogger.Debug("模型查询成功", "name", name, "found_count", len(modelsWithEndpoint))
	return modelsWithEndpoint, nil
}

// FindModelsWithEndpoint 通过模型名称 + 端点类型 + 变体查找
//
// 返回包含模型、平台和端点的完整信息。
//
// 参数：
//   - ctx: 上下文
//   - name: 模型名称或别名
//   - endpointType: 端点类型（如 "openai", "anthropic"）
//   - endpointVariant: 端点变体（如 "chat", "responses"）
//
// 返回值：
//   - []routing.ModelWithEndpoint: 匹配的模型列表（包含平台和端点信息）
//   - error: 错误信息
func (r *Repository) FindModelsWithEndpoint(ctx context.Context, name, endpointType, endpointVariant string) ([]routing.ModelWithEndpoint, error) {
	repoLogger := r.logger.WithGroup("model_repository")
	repoLogger.Debug("根据名称或别名以及端点类型和变体查找模型", "name", name, "endpoint_type", endpointType, "endpoint_variant", endpointVariant)

	q := query.Q
	db := q.WithContext(ctx).Model

	// 使用 JOIN 查询模型、平台和端点
	dbModels, err := db.
		Join(q.Endpoint, q.Model.PlatformID.EqCol(q.Endpoint.PlatformID)).
		Where(q.Endpoint.EndpointType.Eq(endpointType)).
		Where(q.Endpoint.EndpointVariant.Eq(endpointVariant)).
		Where(
			db.Where(q.Model.Name.Eq(name)).Or(q.Model.Alias_.Eq(name)),
		).
		Preload(
			q.Model.APIKeys,
			q.Model.Platform,
			q.Model.Platform.Endpoints.On(
				q.Endpoint.EndpointType.Eq(endpointType),
				q.Endpoint.EndpointVariant.Eq(endpointVariant),
			)).
		Find()

	if err != nil {
		repoLogger.Error("查询模型失败", "error", err, "name", name, "endpoint_type", endpointType, "endpoint_variant", endpointVariant)
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}

	if len(dbModels) == 0 {
		repoLogger.Debug("未找到匹配的模型", "name", name, "endpoint_type", endpointType, "endpoint_variant", endpointVariant)
		return nil, nil
	}

	// 转换为 routing.ModelWithEndpoint 类型
	modelsWithEndpoint := make([]routing.ModelWithEndpoint, 0, len(dbModels))
	for _, model := range dbModels {
		// 转换 APIKeys
		apiKeys := make([]routing.APIKey, len(model.APIKeys))
		for j, dbKey := range model.APIKeys {
			apiKeys[j] = routing.APIKey{
				ID:    dbKey.ID,
				Value: dbKey.Value,
			}
		}

		// 转换 CustomHeaders
		endpointCustomHeaders := copyStringMap(model.Platform.Endpoints[0].CustomHeaders)

		platformCustomHeaders := make(map[string]string)
		// 注意：当前数据库模型 Platform 没有 CustomHeaders 字段，使用空 map

		modelsWithEndpoint = append(modelsWithEndpoint, routing.ModelWithEndpoint{
			Model: routing.Model{
				ID:         model.ID,
				PlatformID: model.PlatformID,
				Name:       model.Name,
				Alias:      model.Alias,
				APIKeys:    apiKeys,
			},
			Platform: routing.Platform{
				ID:            model.Platform.ID,
				Name:          model.Platform.Name,
				BaseURL:       model.Platform.BaseURL,
				RateLimit:     routing.RateLimitConfig{RPM: model.Platform.RateLimit.RPM, TPM: model.Platform.RateLimit.TPM},
				CustomHeaders: platformCustomHeaders,
			},
			Endpoint: routing.Endpoint{
				ID:              model.Platform.Endpoints[0].ID,
				EndpointType:    model.Platform.Endpoints[0].EndpointType,
				EndpointVariant: model.Platform.Endpoints[0].EndpointVariant,
				Path:            model.Platform.Endpoints[0].Path,
				CustomHeaders:   endpointCustomHeaders,
			},
		})
	}

	repoLogger.Debug("模型查询成功", "name", name, "endpoint_type", endpointType, "endpoint_variant", endpointVariant, "found_count", len(modelsWithEndpoint))
	return modelsWithEndpoint, nil
}

// GetPlatformByID 根据 ID 获取平台信息
//
// 参数：
//   - ctx: 上下文
//   - id: 平台 ID
//
// 返回值：
//   - *routing.Platform: 平台信息
//   - error: 错误信息
func (r *Repository) GetPlatformByID(ctx context.Context, id uint) (*routing.Platform, error) {
	repoLogger := r.logger.WithGroup("platform_repository")
	repoLogger.Debug("根据 ID 获取平台信息", "platform_id", id)

	q := query.Q

	// 预加载 Endpoints 关联数据
	dbPlatform, err := q.WithContext(ctx).Platform.Preload(q.Platform.Endpoints).Where(q.Platform.ID.Eq(id)).First()
	if err != nil {
		repoLogger.Error("获取平台失败", "error", err, "platform_id", id)
		return nil, fmt.Errorf("获取平台失败：%w", err)
	}

	// 查找默认端点
	var defaultEndpoint *types.Endpoint
	for i := range dbPlatform.Endpoints {
		if dbPlatform.Endpoints[i].IsDefault {
			defaultEndpoint = &dbPlatform.Endpoints[i]
			break
		}
	}

	// 如果没有默认端点，使用第一个端点
	if defaultEndpoint == nil && len(dbPlatform.Endpoints) > 0 {
		defaultEndpoint = &dbPlatform.Endpoints[0]
		repoLogger.Warn("未找到默认端点，使用第一个端点", "platform_id", id)
	}

	// 转换为 routing.Platform 类型
	platform := &routing.Platform{
		ID:      dbPlatform.ID,
		Name:    dbPlatform.Name,
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

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
