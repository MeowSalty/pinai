package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/portal"
	"github.com/MeowSalty/portal/request"
	"github.com/MeowSalty/portal/routing"
	"github.com/MeowSalty/portal/routing/health"
	coreTypes "github.com/MeowSalty/portal/types"
)

// PortalService AI 网关服务接口
//
// 封装所有与 AI 网关相关的业务逻辑
type PortalService interface {
	// ChatCompletion 处理聊天完成请求
	ChatCompletion(ctx context.Context, req *coreTypes.Request) (*coreTypes.Response, error)

	// Shutdown 优雅关闭服务
	Close(timeout time.Duration) error

	// ChatCompletionStream 处理流式聊天完成请求
	ChatCompletionStream(ctx context.Context, req *coreTypes.Request) (<-chan *coreTypes.Response, error)
}

// portalService AI 网关服务实现
type portalService struct {
	portal *portal.Portal
}

// NewPortalService 创建新的 AI 网关服务实例
//
// 该函数初始化所有必要的组件，包括数据仓库和网关管理器，并正确配置日志记录器。
//
// 参数：
//   - ctx: 上下文，用于初始化网关管理器
//   - logger: 日志记录器实例，用于记录处理过程中的日志信息
//
// 返回值：
//   - PortalService: 初始化后的 AI 网关服务实例
//   - error: 初始化过程中可能出现的错误
func NewPortalService(ctx context.Context, logger *slog.Logger) (PortalService, error) {
	// 创建数据仓库实现
	repo := &DatabaseRepository{}

	// 创建网关管理器
	gatewayManager, err := portal.New(portal.Config{
		PlatformRepo: repo,
		ModelRepo:    repo,
		KeyRepo:      repo,
		HealthRepo:   repo,
		LogRepo:      repo,
	})
	if err != nil {
		return nil, fmt.Errorf("无法创建网关管理器：%w", err)
	}

	return &portalService{portal: gatewayManager}, nil
}

// ChatCompletion 处理聊天完成请求
//
// 提供统一的聊天完成处理入口，包含日志记录和错误处理
func (s *portalService) ChatCompletion(ctx context.Context, req *coreTypes.Request) (*coreTypes.Response, error) {
	// 调用 aigateway 进行处理
	resp, err := s.portal.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("聊天完成处理失败：%w", err)
	}

	return resp, nil
}

// ChatCompletionStream 处理流式聊天完成请求
func (s *portalService) ChatCompletionStream(ctx context.Context, req *coreTypes.Request) (<-chan *coreTypes.Response, error) {
	stream, err := s.portal.ChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("无法启动聊天完成流：%w", err)
	}

	return stream, nil
}

// Close 优雅关闭服务
//
// 停止健康管理器和取消所有相关的上下文
func (s *portalService) Close(timeout time.Duration) error {
	return s.portal.Close(timeout)
}

// DatabaseRepository 数据仓库实现
//
// 提供 portal 所需的数据访问接口
type DatabaseRepository struct{}

// GetModelByID 根据 ID 获取模型信息
func (r *DatabaseRepository) GetModelByID(ctx context.Context, id uint) (routing.Model, error) {
	q := query.Q

	dbModel, err := q.WithContext(ctx).Model.Where(q.Model.ID.Eq(id)).First()
	if err != nil {
		return routing.Model{}, fmt.Errorf("获取模型失败：%w", err)
	}

	// 转换为 core.Model 类型
	model := routing.Model{
		ID:         dbModel.ID,
		PlatformID: dbModel.PlatformID,
		Name:       dbModel.Name,
		Alias:      dbModel.Alias,
	}

	return model, nil
}

// FindModelsByNameOrAlias 根据名称或别名查找模型
func (r *DatabaseRepository) FindModelsByNameOrAlias(ctx context.Context, name string) ([]routing.Model, error) {
	q := query.Q

	// 使用 GORM 查询模型（按名称或别名查找）
	dbModels, err := q.WithContext(ctx).Model.Where(
		q.Model.Name.Eq(name),
	).Or(
		q.Model.Alias_.Eq(name),
	).Find()

	if err != nil {
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}

	// 转换为 core.Model 类型
	models := make([]routing.Model, len(dbModels))
	for i, dbModel := range dbModels {
		models[i] = routing.Model{
			ID:         dbModel.ID,
			PlatformID: dbModel.PlatformID,
			Name:       dbModel.Name,
			Alias:      dbModel.Alias,
		}
	}

	return models, nil
}

// GetPlatformByID 根据 ID 获取平台信息
func (r *DatabaseRepository) GetPlatformByID(ctx context.Context, id uint) (*routing.Platform, error) {
	q := query.Q

	dbPlatform, err := q.WithContext(ctx).Platform.Where(q.Platform.ID.Eq(id)).First()
	if err != nil {
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

	return platform, nil
}

// GetAllAPIKeysByPlatformID 根据平台 ID 获取所有 API 密钥
func (r *DatabaseRepository) GetAllAPIKeysByPlatformID(ctx context.Context, platformID uint) ([]*routing.APIKey, error) {
	q := query.Q

	dbKeys, err := q.WithContext(ctx).APIKey.Where(q.APIKey.PlatformID.Eq(platformID)).Find()
	if err != nil {
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

	return keys, nil
}

// GetAllHealth 获取所有健康状态
func (r *DatabaseRepository) GetAllHealth(ctx context.Context) ([]*health.Health, error) {
	q := query.Q

	dbHealths, err := q.WithContext(ctx).Health.Find()
	if err != nil {
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

	return healths, nil
}

// BatchUpdateHealth 批量更新健康状态
func (r *DatabaseRepository) BatchUpdateHealth(ctx context.Context, statuses []health.Health) error {
	q := query.Q

	// 开启事务
	tx := q.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, status := range statuses {
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
			tx.Rollback()
			return fmt.Errorf("批量更新健康状态失败：%w", err)
		}
	}

	// 检查提交事务是否有错误
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败：%w", err)
	}

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
func (r *DatabaseRepository) CreateRequestLog(ctx context.Context, log *request.RequestLog) error {
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
	err := query.Q.WithContext(ctx).RequestLog.Create(dbLog)
	if err != nil {
		return fmt.Errorf("保存请求日志失败：%w", err)
	}

	return nil
}
