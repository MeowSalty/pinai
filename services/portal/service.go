package portal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/health"
	"github.com/MeowSalty/portal"
	adapterTypes "github.com/MeowSalty/portal/request/adapter/types"
	coreHealth "github.com/MeowSalty/portal/routing/health"
)

// Service Portal 服务接口
//
// 封装所有与 Portal 相关的业务逻辑
type Service interface {
	// ChatCompletion 处理聊天完成请求
	ChatCompletion(ctx context.Context, req *adapterTypes.RequestContract) (*adapterTypes.ResponseContract, error)

	// Close 优雅关闭服务
	Close(timeout time.Duration) error

	// ChatCompletionStream 处理流式聊天完成请求
	ChatCompletionStream(ctx context.Context, req *adapterTypes.RequestContract) (<-chan *adapterTypes.StreamEventContract, error)
}

// healthStorageAdapter 适配器，将内部 health.Storage 转换为 portal 需要的 health.Storage 接口
type healthStorageAdapter struct {
	storage *health.Storage
}

// Get 实现 portal health.Storage 接口的 Get 方法
func (a *healthStorageAdapter) Get(resourceType coreHealth.ResourceType, resourceID uint) (*coreHealth.Health, error) {
	// 将 portal 库的 ResourceType 转换为内部 health 包的 ResourceType
	internalResourceType := convertResourceTypeToInternal(resourceType)
	internalResult, err := a.storage.Get(internalResourceType, resourceID)
	if err != nil || internalResult == nil {
		return nil, err
	}

	// 将内部 health.Health 转换为 portal 库的 health.Health
	return &coreHealth.Health{
		ResourceType:    resourceType,
		ResourceID:      internalResult.ResourceID,
		Status:          coreHealth.HealthStatus(internalResult.Status),
		RetryCount:      internalResult.RetryCount,
		NextAvailableAt: internalResult.NextAvailableAt,
		BackoffDuration: internalResult.BackoffDuration,
		LastError:       internalResult.LastError,
		LastErrorCode:   internalResult.LastErrorCode,
		LastCheckAt:     internalResult.LastCheckAt,
		LastSuccessAt:   internalResult.LastSuccessAt,
		SuccessCount:    internalResult.SuccessCount,
		ErrorCount:      internalResult.ErrorCount,
		CreatedAt:       internalResult.CreatedAt,
		UpdatedAt:       internalResult.UpdatedAt,
	}, nil
}

// Set 实现 portal health.Storage 接口的 Set 方法
func (a *healthStorageAdapter) Set(status *coreHealth.Health) error {
	// 将 portal 库的 health.Health 转换为内部 health 包的 health.Health 类型
	internalStatus := &types.Health{
		ResourceType:    convertResourceTypeToInternal(status.ResourceType),
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
		CreatedAt:       status.CreatedAt,
		UpdatedAt:       status.UpdatedAt,
	}

	return a.storage.Set(internalStatus)
}

// Delete 实现 portal health.Storage 接口的 Delete 方法
func (a *healthStorageAdapter) Delete(resourceType coreHealth.ResourceType, resourceID uint) error {
	// 将 portal 库的 ResourceType 转换为内部 health 包的 ResourceType
	internalResourceType := convertResourceTypeToInternal(resourceType)
	return a.storage.Delete(internalResourceType, resourceID)
}

// convertResourceTypeToInternal 将 portal 库的 ResourceType 转换为内部 health 包的 ResourceType
func convertResourceTypeToInternal(portalType coreHealth.ResourceType) types.ResourceType {
	// 直接类型转换，因为它们应该有相同的值定义
	return types.ResourceType(portalType)
}

// service Portal 服务实现
type service struct {
	portal           *portal.Portal
	modelMappingRule map[string]string
	logger           *slog.Logger
}

// New 创建新的 Portal 服务实例
//
// 该函数初始化所有必要的组件，包括数据仓库和网关管理器，并正确配置日志记录器。
//
// 参数：
//   - ctx: 上下文，用于初始化网关管理器
//   - logger: 日志记录器实例，用于记录处理过程中的日志信息
//   - modelMappingStr: 模型映射规则字符串，格式为 "key1:value1,key2:value2"
//   - healthStorage: 健康状态存储实例，由 health 包初始化后传入
//
// 返回值：
//   - Service: 初始化后的 Portal 服务实例
//   - error: 初始化过程中可能出现的错误
func New(ctx context.Context, logger *slog.Logger, modelMappingStr string, healthStorage *health.Storage) (Service, error) {
	logger.Info("开始初始化 Portal 服务", "model_mapping", modelMappingStr)

	repoLogger := logger.WithGroup("database_repository")
	repo := &Repository{logger: repoLogger}

	// 创建适配器，将内部 health.Storage 转换为 portal 库需要的接口
	adapter := &healthStorageAdapter{
		storage: healthStorage,
	}

	logger.Debug("正在创建网关管理器")
	gatewayManager, err := portal.New(portal.Config{
		PlatformRepo:  repo,
		ModelRepo:     repo,
		KeyRepo:       repo,
		HealthStorage: adapter,
		LogRepo:       repo,
		Logger:        NewSlogAdapter(logger),
	})
	if err != nil {
		logger.Error("创建网关管理器失败", "error", err)
		return nil, fmt.Errorf("无法创建网关管理器：%w", err)
	}
	logger.Info("网关管理器创建成功")

	logger.Debug("正在解析模型映射规则")
	modelMappingRule, err := parseModelMapping(modelMappingStr)
	if err != nil {
		logger.Error("解析模型映射规则失败", "error", err, "mapping_str", modelMappingStr)
		return nil, fmt.Errorf("解析模型映射规则失败：%w", err)
	}

	if len(modelMappingRule) == 0 {
		logger.Debug("未启用模型映射规则")
	} else {
		logger.Info("使用自定义模型映射规则", "mapping", modelMappingRule, "count", len(modelMappingRule))
	}

	logger.Info("Portal 服务初始化完成")
	return &service{
		portal:           gatewayManager,
		modelMappingRule: modelMappingRule,
		logger:           logger,
	}, nil
}

// ChatCompletion 处理聊天完成请求
//
// 提供统一的聊天完成处理入口，包含日志记录和错误处理
func (s *service) ChatCompletion(ctx context.Context, req *adapterTypes.RequestContract) (*adapterTypes.ResponseContract, error) {
	requestLogger := s.logger.WithGroup("chat_completion")
	requestLogger.Info("开始处理聊天完成请求", "model", req.Model)

	originalModel := req.Model

	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		requestLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	startTime := time.Now()

	resp, err := s.portal.ChatCompletion(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		requestLogger.Error("聊天完成处理失败",
			"error", err,
			"duration", duration,
			"model", req.Model,
			"original_model", originalModel)
		return nil, fmt.Errorf("聊天完成处理失败：%w", err)
	}

	requestLogger.Info("聊天完成请求处理成功",
		"duration", duration,
		"model", req.Model,
		"original_model", originalModel,
		"response_id", resp.ID,
		"usage", resp.Usage)

	return resp, nil
}

// ChatCompletionStream 处理流式聊天完成请求
func (s *service) ChatCompletionStream(ctx context.Context, req *adapterTypes.RequestContract) (<-chan *adapterTypes.StreamEventContract, error) {
	streamLogger := s.logger.WithGroup("chat_completion_stream")
	streamLogger.Info("开始处理流式聊天完成请求", "model", req.Model)

	originalModel := req.Model

	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		streamLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	streamLogger.Debug("正在启动流式处理")
	stream := s.portal.ChatCompletionStream(ctx, req)

	streamLogger.Info("聊天完成流启动成功", "model", req.Model)
	return stream, nil
}

// Close 优雅关闭服务
//
// 停止健康管理器和取消所有相关的上下文
func (s *service) Close(timeout time.Duration) error {
	s.logger.Info("开始优雅关闭服务", "timeout", timeout)

	err := s.portal.Close(timeout)
	if err != nil {
		s.logger.Error("服务关闭失败", "error", err, "timeout", timeout)
		return err
	}

	s.logger.Info("服务关闭成功")
	return nil
}
