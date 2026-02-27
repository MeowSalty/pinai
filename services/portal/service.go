package portal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/services/health"
	"github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	adapterTypes "github.com/MeowSalty/portal/request/adapter/types"
)

// Service Portal 服务接口
//
// 封装所有与 Portal 相关的业务逻辑
type Service interface {
	// ChatCompletion 处理聊天完成请求
	//
	// Deprecated: 将在未来的版本中被移除，使用 Native* 方法替代
	ChatCompletion(ctx context.Context, req *adapterTypes.RequestContract) (*adapterTypes.ResponseContract, error)

	// Close 优雅关闭服务
	Close(timeout time.Duration) error

	// ChatCompletionStream 处理流式聊天完成请求
	//
	// Deprecated: 将在未来的版本中被移除，使用 Native* 方法替代
	ChatCompletionStream(ctx context.Context, req *adapterTypes.RequestContract) (<-chan *adapterTypes.StreamEventContract, error)

	// === Native ===

	// Anthropic
	NativeAnthropicMessages(ctx context.Context, req *anthropicTypes.Request, opts ...portal.NativeOption) (*anthropicTypes.Response, error)
	NativeAnthropicMessagesStream(ctx context.Context, req *anthropicTypes.Request, opts ...portal.NativeOption) <-chan *anthropicTypes.StreamEvent

	// Gemini
	NativeGeminiGenerateContent(ctx context.Context, req *geminiTypes.Request, opts ...portal.NativeOption) (*geminiTypes.Response, error)
	NativeGeminiStreamGenerateContent(ctx context.Context, req *geminiTypes.Request, opts ...portal.NativeOption) <-chan *geminiTypes.StreamEvent

	// OpenAI
	NativeOpenAIChatCompletion(ctx context.Context, req *openaiChatTypes.Request, opts ...portal.NativeOption) (*openaiChatTypes.Response, error)
	NativeOpenAIChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request, opts ...portal.NativeOption) <-chan *openaiChatTypes.StreamEvent
	NativeOpenAIResponses(ctx context.Context, req *openaiResponsesTypes.Request, opts ...portal.NativeOption) (*openaiResponsesTypes.Response, error)
	NativeOpenAIResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request, opts ...portal.NativeOption) <-chan *openaiResponsesTypes.StreamEvent
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
