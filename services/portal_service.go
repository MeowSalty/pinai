package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database"
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/portal"
	"github.com/MeowSalty/portal/health"
	"github.com/MeowSalty/portal/selector"
	coreTypes "github.com/MeowSalty/portal/types"
)

// AIGatewayService AI 网关服务接口
// 封装所有与 AI 网关相关的业务逻辑
type AIGatewayService interface {
	// ProcessChatCompletion 处理聊天完成请求
	ProcessChatCompletion(ctx context.Context, req *coreTypes.Request) (*coreTypes.Response, error)

	// ProcessCompletion 处理文本补全请求
	// ProcessCompletion(ctx context.Context, req *coreTypes.Request) (*coreTypes.Response, error)

	// CompletionStream 处理流式文本补全请求
	// CompletionStream(ctx context.Context, req *coreTypes.Request) (<-chan *coreTypes.Response, error)

	// Shutdown 优雅关闭服务
	Shutdown(ctx context.Context) error

	// ChatCompletionStream 处理流式聊天完成请求
	ChatCompletionStream(ctx context.Context, req *coreTypes.Request) (<-chan *coreTypes.Response, error)
}

// aiGatewayService AI 网关服务实现
type aiGatewayService struct {
	// gatewayManager aigateway 管理器实例
	gatewayManager *portal.GatewayManager

	// healthManager 健康状态管理器
	healthManager *health.Manager

	// logger 日志实例
	logger *slog.Logger

	// ctx 服务上下文
	ctx context.Context

	// cancel 取消函数
	cancel context.CancelFunc
}

// NewAIGatewayService 创建新的 AI 网关服务实例
// 初始化所有必要的组件，包括健康管理器、选择器和适配器
func NewAIGatewayService(ctx context.Context, logger *slog.Logger) (AIGatewayService, error) {
	serviceCtx, cancel := context.WithCancel(ctx)
	serviceLogger := logger.With("service", "aigateway")

	serviceLogger.Info("正在初始化 AI 网关服务")

	// 创建数据仓库实现
	repo := &DatabaseRepository{}

	// 创建健康状态管理器
	healthManager, err := health.NewManager(serviceCtx, repo, serviceLogger.WithGroup("health"), time.Minute)
	if err != nil {
		cancel()
		serviceLogger.Error("创建健康管理器失败", "error", err)
		return nil, fmt.Errorf("创建健康管理器失败：%w", err)
	}

	// 创建随机选择器
	channelSelector := selector.NewRandomSelector(healthManager)

	// 配置网关管理器
	config := &portal.Config{
		Repo:          repo,
		HealthManager: healthManager,
		Selector:      channelSelector,
		Logger:        serviceLogger.WithGroup("gateway_manager"),
	}

	// 创建网关管理器
	gatewayManager := portal.NewGatewayManager(config)

	service := &aiGatewayService{
		gatewayManager: gatewayManager,
		healthManager:  healthManager,
		logger:         serviceLogger,
		ctx:            serviceCtx,
		cancel:         cancel,
	}

	serviceLogger.Info("AI 网关服务初始化完成")
	return service, nil
}

// ProcessChatCompletion 处理聊天完成请求
// 提供统一的聊天完成处理入口，包含日志记录和错误处理
func (s *aiGatewayService) ProcessChatCompletion(ctx context.Context, req *coreTypes.Request) (*coreTypes.Response, error) {
	s.logger.InfoContext(ctx, "开始处理聊天完成请求",
		slog.String("model", req.Model),
		slog.Int("message_count", len(req.Messages)),
		slog.Bool("stream", *req.Stream),
	)

	// 调用 aigateway 进行处理
	resp, err := s.gatewayManager.ChatCompletion(ctx, req)
	if err != nil {
		s.logger.ErrorContext(ctx, "聊天完成处理失败", slog.Any("error", err))
		return nil, fmt.Errorf("聊天完成处理失败：%w", err)
	}

	// 记录成功处理的信息
	s.logger.InfoContext(ctx, "聊天完成处理成功",
		"prompt_tokens", resp.Usage.PromptTokens,
		"completion_tokens", resp.Usage.CompletionTokens,
		"total_tokens", resp.Usage.TotalTokens,
	)

	return resp, nil
}

// ChatCompletionStream 处理流式聊天完成请求
func (s *aiGatewayService) ChatCompletionStream(ctx context.Context, req *coreTypes.Request) (<-chan *coreTypes.Response, error) {
	s.logger.InfoContext(ctx, "开始处理流式聊天完成请求",
		slog.String("model", req.Model),
		slog.Int("message_count", len(req.Messages)),
	)

	stream, err := s.gatewayManager.ChatCompletionStream(ctx, req)
	if err != nil {
		s.logger.ErrorContext(ctx, "无法启动聊天完成流", slog.Any("error", err))
		return nil, fmt.Errorf("无法启动聊天完成流：%w", err)
	}

	return stream, nil
}

// Shutdown 优雅关闭服务
// 停止健康管理器和取消所有相关的上下文
func (s *aiGatewayService) Shutdown(ctx context.Context) error {
	s.logger.Info("开始关闭 AI 网关服务")

	// 关闭健康管理器
	if s.healthManager != nil {
		s.logger.Info("正在关闭健康管理器")
		s.healthManager.Shutdown()
		s.logger.Info("健康管理器已关闭")
	}

	// 取消服务上下文
	if s.cancel != nil {
		s.cancel()
		s.logger.Info("服务上下文已取消")
	}

	s.logger.Info("AI 网关服务关闭完成")
	return nil
}

// DatabaseRepository 数据仓库实现
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
			return nil, fmt.Errorf("查询模型失败：%w", err)
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

	return tx.Commit()
}

// CountRequestStats
func (r *DatabaseRepository) CountRequestStats(ctx context.Context, params *coreTypes.StatsQueryParams) (*coreTypes.StatsSummary, error) {
	panic("unimplemented")
}

// QueryRequestStats i
func (r *DatabaseRepository) QueryRequestStats(ctx context.Context, params *coreTypes.StatsQueryParams) ([]*coreTypes.RequestStat, error) {
	panic("unimplemented")
}

// SaveRequestStat
func (r *DatabaseRepository) SaveRequestStat(ctx context.Context, stat *coreTypes.RequestStat) error {
	panic("unimplemented")
}
