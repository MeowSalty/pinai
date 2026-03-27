package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

// Service 定义数据面网关应用服务接口。
//
// 当前仅提供第一批最小落地链路：OpenAI compat Chat Completions。
type Service interface {
	// Close 优雅关闭网关依赖资源。
	Close(timeout time.Duration) error

	// AnthropicNativeMessages 处理 Anthropic native Messages 非流式请求。
	AnthropicNativeMessages(ctx context.Context, req *anthropicTypes.Request) (*anthropicTypes.Response, error)

	// AnthropicNativeMessagesStream 处理 Anthropic native Messages 流式请求。
	AnthropicNativeMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent

	// AnthropicNativeMessagesStreamResult 处理 Anthropic native Messages 流式请求并返回最小收口结果。
	AnthropicNativeMessagesStreamResult(ctx context.Context, req *anthropicTypes.Request) <-chan AnthropicStreamResult

	// GeminiNativeGenerateContent 处理 Gemini native generateContent 非流式请求。
	GeminiNativeGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error)

	// GeminiNativeGenerateContentStream 处理 Gemini native streamGenerateContent 流式请求。
	GeminiNativeGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent

	// GeminiNativeGenerateContentStreamResult 处理 Gemini native streamGenerateContent 流式请求并返回最小收口结果。
	GeminiNativeGenerateContentStreamResult(ctx context.Context, req *geminiTypes.Request) <-chan GeminiStreamResult

	// AnthropicCompatMessages 处理 Anthropic compat Messages 非流式请求。
	AnthropicCompatMessages(ctx context.Context, req *anthropicTypes.Request) (*anthropicTypes.Response, error)

	// AnthropicCompatMessagesStream 处理 Anthropic compat Messages 流式请求。
	AnthropicCompatMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent

	// AnthropicCompatMessagesStreamResult 处理 Anthropic compat Messages 流式请求并返回最小收口结果。
	AnthropicCompatMessagesStreamResult(ctx context.Context, req *anthropicTypes.Request) <-chan AnthropicStreamResult

	// GeminiCompatGenerateContent 处理 Gemini compat generateContent 非流式请求。
	GeminiCompatGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error)

	// GeminiCompatGenerateContentStream 处理 Gemini compat streamGenerateContent 流式请求。
	GeminiCompatGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent

	// GeminiCompatGenerateContentStreamResult 处理 Gemini compat streamGenerateContent 流式请求并返回最小收口结果。
	GeminiCompatGenerateContentStreamResult(ctx context.Context, req *geminiTypes.Request) <-chan GeminiStreamResult

	// OpenAICompatChatCompletion 处理 OpenAI compat Chat Completions 非流式请求。
	OpenAICompatChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error)

	// OpenAICompatChatCompletionStream 处理 OpenAI compat Chat Completions 流式请求。
	OpenAICompatChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent

	// OpenAICompatChatCompletionStreamResult 处理 OpenAI compat Chat Completions 流式请求并返回最小收口结果。
	OpenAICompatChatCompletionStreamResult(ctx context.Context, req *openaiChatTypes.Request) <-chan OpenAIChatStreamResult

	// OpenAICompatResponses 处理 OpenAI compat Responses 非流式请求。
	OpenAICompatResponses(ctx context.Context, req *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error)

	// OpenAICompatResponsesStream 处理 OpenAI compat Responses 流式请求。
	OpenAICompatResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request) <-chan *openaiResponsesTypes.StreamEvent

	// OpenAICompatResponsesStreamResult 处理 OpenAI compat Responses 流式请求并返回最小收口结果。
	OpenAICompatResponsesStreamResult(ctx context.Context, req *openaiResponsesTypes.Request) <-chan OpenAIResponsesStreamResult

	// OpenAINativeChatCompletion 处理 OpenAI native Chat Completions 非流式请求。
	OpenAINativeChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error)

	// OpenAINativeChatCompletionStream 处理 OpenAI native Chat Completions 流式请求。
	OpenAINativeChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent

	// OpenAINativeChatCompletionStreamResult 处理 OpenAI native Chat Completions 流式请求并返回最小收口结果。
	OpenAINativeChatCompletionStreamResult(ctx context.Context, req *openaiChatTypes.Request) <-chan OpenAIChatStreamResult

	// OpenAINativeResponses 处理 OpenAI native Responses 非流式请求。
	OpenAINativeResponses(ctx context.Context, req *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error)

	// OpenAINativeResponsesStream 处理 OpenAI native Responses 流式请求。
	OpenAINativeResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request) <-chan *openaiResponsesTypes.StreamEvent

	// OpenAINativeResponsesStreamResult 处理 OpenAI native Responses 流式请求并返回最小收口结果。
	OpenAINativeResponsesStreamResult(ctx context.Context, req *openaiResponsesTypes.Request) <-chan OpenAIResponsesStreamResult

	// MapDataPlaneError 对数据面错误进行第一轮统一映射。
	MapDataPlaneError(err error, fallbackAction string) DataPlaneError
}

type service struct {
	portalService GatewayPort
	logger        *slog.Logger
}

// New 创建网关应用服务。
func New(portalService GatewayPort, logger *slog.Logger) Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &service{
		portalService: portalService,
		logger:        logger,
	}
}

// Close 优雅关闭网关依赖资源。
func (s *service) Close(timeout time.Duration) error {
	if s.portalService == nil {
		return nil
	}

	if err := s.portalService.Close(timeout); err != nil {
		return fmt.Errorf("关闭网关依赖的 Portal 服务失败：%w", err)
	}

	return nil
}
