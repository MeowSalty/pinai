package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database"
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/portal"
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
	portal *portal.GatewayManager
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
	gatewayManager, err := portal.New(
		ctx,
		portal.WithRepository(repo),
		portal.WithLogger(logger),
	)
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
	return s.portal.Shutdown(timeout)
}

// DatabaseRepository 数据仓库实现
//
// 提供 aigateway 所需的数据访问接口
type DatabaseRepository struct{}

// FindModelsByName 根据名称查找模型
func (r *DatabaseRepository) FindModelsByName(ctx context.Context, name string) ([]*coreTypes.Model, error) {
	q := database.Q

	// 使用 GORM 查询模型（先按名称查找，再按别名查找）
	dbModels, err := q.WithContext(ctx).Model.Where(
		q.Model.Name.Eq(name),
	).Find()

	if err != nil {
		return nil, fmt.Errorf("查询模型失败：%w", err)
	}

	// 如果按名称没找到，再按别名查找
	if len(dbModels) == 0 {
		dbModels, err = q.WithContext(ctx).Model.Where(
			q.Model.Alias_.Eq(name),
		).Find()
		if err != nil {
			return nil, fmt.Errorf("按别名查询模型失败：%w", err)
		}
	}

	// 转换为 core.Model 类型
	models := make([]*coreTypes.Model, len(dbModels))
	for i, dbModel := range dbModels {
		models[i] = &coreTypes.Model{
			ID:         dbModel.ID,
			PlatformID: dbModel.PlatformID,
			Name:       dbModel.Name,
			Alias:      dbModel.Alias,
		}
	}

	return models, nil
}

// GetPlatformByID 根据 ID 获取平台信息
func (r *DatabaseRepository) GetPlatformByID(ctx context.Context, id uint) (*coreTypes.Platform, error) {
	q := database.Q

	dbPlatform, err := q.WithContext(ctx).Platform.Where(q.Platform.ID.Eq(id)).First()
	if err != nil {
		return nil, fmt.Errorf("获取平台失败：%w", err)
	}

	// 转换为 core.Platform 类型
	platform := &coreTypes.Platform{
		ID:      dbPlatform.ID,
		Name:    dbPlatform.Name,
		Format:  dbPlatform.Format,
		BaseURL: dbPlatform.BaseURL,
		RateLimit: coreTypes.RateLimitConfig{
			RPM: dbPlatform.RateLimit.RPM,
			TPM: dbPlatform.RateLimit.TPM,
		},
	}

	return platform, nil
}

// GetAllAPIKeys 获取平台的所有 API 密钥
func (r *DatabaseRepository) GetAllAPIKeys(ctx context.Context, platformID uint) ([]*coreTypes.APIKey, error) {
	q := database.Q

	dbKeys, err := q.WithContext(ctx).APIKey.Where(q.APIKey.PlatformID.Eq(platformID)).Find()
	if err != nil {
		return nil, fmt.Errorf("获取 API 密钥失败：%w", err)
	}

	// 转换为 core.APIKey 类型
	keys := make([]*coreTypes.APIKey, len(dbKeys))
	for i, dbKey := range dbKeys {
		keys[i] = &coreTypes.APIKey{
			ID:    dbKey.ID,
			Value: dbKey.Value,
		}
	}

	return keys, nil
}

// GetAllHealthStatus 获取所有健康状态
func (r *DatabaseRepository) GetAllHealthStatus(ctx context.Context) ([]*coreTypes.Health, error) {
	q := database.Q

	dbHealths, err := q.WithContext(ctx).Health.Find()
	if err != nil {
		return nil, fmt.Errorf("获取健康状态失败：%w", err)
	}

	// 转换为 core.Health 类型
	healths := make([]*coreTypes.Health, len(dbHealths))
	for i, dbHealth := range dbHealths {
		healths[i] = &coreTypes.Health{
			ID:                dbHealth.ID,
			ResourceType:      coreTypes.ResourceType(dbHealth.ResourceType),
			ResourceID:        dbHealth.ResourceID,
			RelatedPlatformID: dbHealth.RelatedPlatformID,
			RelatedAPIKeyID:   dbHealth.RelatedAPIKeyID,
			Status:            coreTypes.HealthStatus(dbHealth.Status),
			RetryCount:        dbHealth.RetryCount,
			NextAvailableAt:   dbHealth.NextAvailableAt,
			BackoffDuration:   dbHealth.BackoffDuration,
			LastError:         dbHealth.LastError,
			LastErrorCode:     dbHealth.LastErrorCode,
			LastCheckAt:       dbHealth.LastCheckAt,
			LastSuccessAt:     dbHealth.LastSuccessAt,
			SuccessCount:      dbHealth.SuccessCount,
			ErrorCount:        dbHealth.ErrorCount,
			CreatedAt:         dbHealth.CreatedAt,
			UpdatedAt:         dbHealth.UpdatedAt,
		}
	}

	return healths, nil
}

// BatchUpdateHealthStatus 批量更新健康状态
func (r *DatabaseRepository) BatchUpdateHealthStatus(ctx context.Context, statuses []*coreTypes.Health) error {
	q := database.Q

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
			ID:                status.ID,
			ResourceType:      types.ResourceType(status.ResourceType),
			ResourceID:        status.ResourceID,
			RelatedPlatformID: status.RelatedPlatformID,
			RelatedAPIKeyID:   status.RelatedAPIKeyID,
			Status:            types.HealthStatus(status.Status),
			RetryCount:        status.RetryCount,
			NextAvailableAt:   status.NextAvailableAt,
			BackoffDuration:   status.BackoffDuration,
			LastError:         status.LastError,
			LastErrorCode:     status.LastErrorCode,
			LastCheckAt:       status.LastCheckAt,
			LastSuccessAt:     status.LastSuccessAt,
			SuccessCount:      status.SuccessCount,
			ErrorCount:        status.ErrorCount,
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

// CountRequestStats 统计请求数据
//
// 根据给定的查询参数统计请求数据
//
// 参数：
//   - ctx: 上下文
//   - params: 统计查询参数
//
// 返回值：
//   - *coreTypes.StatsSummary: 统计摘要数据
//   - error: 错误信息
func (r *DatabaseRepository) CountRequestStats(ctx context.Context, params *coreTypes.StatsQueryParams) (*coreTypes.StatsSummary, error) {
	q := database.Q.WithContext(ctx).RequestStat

	// 根据查询参数构建查询条件
	if params.ModelName != nil && *params.ModelName != "" {
		q = q.Where(database.Q.RequestStat.ModelName.Eq(*params.ModelName))
	}

	if params.StartTime != nil && !params.StartTime.IsZero() {
		q = q.Where(database.Q.RequestStat.Timestamp.Gte(*params.StartTime))
	}

	if params.EndTime != nil && !params.EndTime.IsZero() {
		q = q.Where(database.Q.RequestStat.Timestamp.Lte(*params.EndTime))
	}

	if params.Success != nil {
		q = q.Where(database.Q.RequestStat.Success.Is(*params.Success))
	}

	// 执行统计查询
	summary := &coreTypes.StatsSummary{}

	// 查询总请求数
	totalCount, err := q.Count()
	if err != nil {
		return nil, fmt.Errorf("统计总请求数失败：%w", err)
	}
	summary.TotalRequests = totalCount

	// 查询成功请求数
	successCondition := true
	successCount, err := q.Where(database.Q.RequestStat.Success.Is(successCondition)).Count()
	if err != nil {
		return nil, fmt.Errorf("统计成功请求数失败：%w", err)
	}
	summary.SuccessRequests = successCount

	return summary, nil
}

// QueryRequestStats 查询请求统计数据
//
// 根据给定的查询参数查询请求统计数据
//
// 参数：
//   - ctx: 上下文
//   - params: 统计查询参数
//
// 返回值：
//   - []*coreTypes.RequestStat: 请求统计列表
//   - error: 错误信息
func (r *DatabaseRepository) QueryRequestStats(ctx context.Context, params *coreTypes.StatsQueryParams) ([]*coreTypes.RequestStat, error) {
	q := database.Q.WithContext(ctx).RequestStat

	// 根据查询参数构建查询条件
	if params.ModelName != nil && *params.ModelName != "" {
		q = q.Where(database.Q.RequestStat.ModelName.Eq(*params.ModelName))
	}

	if params.StartTime != nil && !params.StartTime.IsZero() {
		q = q.Where(database.Q.RequestStat.Timestamp.Gte(*params.StartTime))
	}

	if params.EndTime != nil && !params.EndTime.IsZero() {
		q = q.Where(database.Q.RequestStat.Timestamp.Lte(*params.EndTime))
	}

	if params.Success != nil {
		q = q.Where(database.Q.RequestStat.Success.Is(*params.Success))
	}

	// 执行查询
	results, err := q.Find()
	if err != nil {
		return nil, fmt.Errorf("查询请求统计数据失败：%w", err)
	}

	// 转换为 coreTypes.RequestStat 类型
	requestStats := make([]*coreTypes.RequestStat, len(results))
	for i, result := range results {
		requestStats[i] = &coreTypes.RequestStat{
			ID:          result.ID,
			Timestamp:   result.Timestamp,
			RequestType: result.RequestType,
			ModelName:   result.ModelName,
			ChannelInfo: coreTypes.ChannelInfo{
				PlatformID: result.ChannelInfo.PlatformID,
				APIKeyID:   result.ChannelInfo.APIKeyID,
				ModelID:    result.ChannelInfo.ModelID,
			},
			Duration:      result.Duration,
			FirstByteTime: result.FirstByteTime,
			Success:       result.Success,
			ErrorMsg:      result.ErrorMsg,
		}
	}

	return requestStats, nil
}

// SaveRequestStat 保存请求统计
//
// 保存请求统计信息到数据库
//
// 参数：
//   - ctx: 上下文
//   - stat: 请求统计信息
//
// 返回值：
//   - error: 错误信息
func (r *DatabaseRepository) SaveRequestStat(ctx context.Context, stat *coreTypes.RequestStat) error {
	// 将 coreTypes.RequestStat 转换为数据库类型
	dbStat := &types.RequestStat{
		ID:          stat.ID,
		Timestamp:   stat.Timestamp,
		RequestType: stat.RequestType,
		ModelName:   stat.ModelName,
		ChannelInfo: types.ChannelInfo{
			PlatformID: stat.ChannelInfo.PlatformID,
			APIKeyID:   stat.ChannelInfo.APIKeyID,
			ModelID:    stat.ChannelInfo.ModelID,
		},
		Duration:      stat.Duration,
		FirstByteTime: stat.FirstByteTime,
		Success:       stat.Success,
		ErrorMsg:      stat.ErrorMsg,
	}

	// 保存到数据库
	err := database.Q.WithContext(ctx).RequestStat.Create(dbStat)
	if err != nil {
		return fmt.Errorf("保存请求统计信息失败：%w", err)
	}

	return nil
}
