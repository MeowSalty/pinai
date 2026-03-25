package gateway

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/services/portal"
	portalLib "github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

// Service 定义数据面网关应用服务接口。
//
// 当前仅提供第一批最小落地链路：OpenAI compat Chat Completions。
type Service interface {
	// AnthropicCompatMessages 处理 Anthropic compat Messages 非流式请求。
	AnthropicCompatMessages(ctx context.Context, req *anthropicTypes.Request) (*anthropicTypes.Response, error)

	// AnthropicCompatMessagesStream 处理 Anthropic compat Messages 流式请求。
	AnthropicCompatMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent

	// GeminiCompatGenerateContent 处理 Gemini compat generateContent 非流式请求。
	GeminiCompatGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error)

	// GeminiCompatGenerateContentStream 处理 Gemini compat streamGenerateContent 流式请求。
	GeminiCompatGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent

	// OpenAICompatChatCompletion 处理 OpenAI compat Chat Completions 非流式请求。
	OpenAICompatChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error)

	// OpenAICompatChatCompletionStream 处理 OpenAI compat Chat Completions 流式请求。
	OpenAICompatChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent

	// OpenAICompatResponses 处理 OpenAI compat Responses 非流式请求。
	OpenAICompatResponses(ctx context.Context, req *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error)

	// OpenAICompatResponsesStream 处理 OpenAI compat Responses 流式请求。
	OpenAICompatResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request) <-chan *openaiResponsesTypes.StreamEvent

	// OpenAINativeChatCompletion 处理 OpenAI native Chat Completions 非流式请求。
	OpenAINativeChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error)
}

type service struct {
	portalService portal.Service
	logger        *slog.Logger
}

// New 创建网关应用服务。
func New(portalService portal.Service, logger *slog.Logger) Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &service{
		portalService: portalService,
		logger:        logger,
	}
}

// AnthropicCompatMessages 处理 Anthropic compat Messages 非流式请求。
func (s *service) AnthropicCompatMessages(ctx context.Context, req *anthropicTypes.Request) (*anthropicTypes.Response, error) {
	logger := s.logger.WithGroup("anthropic_compat_messages")
	logger.Info("开始执行 Anthropic compat Messages 非流式请求", "model", req.Model)

	resp, err := s.portalService.NativeAnthropicMessages(ctx, req, portalLib.WithCompatMode())
	if err != nil {
		logger.Error("Anthropic compat Messages 非流式请求失败", "error", err, "model", req.Model)
		return nil, fmt.Errorf("处理 Anthropic compat Messages 请求失败：%w", err)
	}

	logger.Info("Anthropic compat Messages 非流式请求成功", "model", req.Model)
	return resp, nil
}

// AnthropicCompatMessagesStream 处理 Anthropic compat Messages 流式请求。
func (s *service) AnthropicCompatMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent {
	logger := s.logger.WithGroup("anthropic_compat_messages_stream")
	logger.Info("开始执行 Anthropic compat Messages 流式请求", "model", req.Model)

	stream := s.portalService.NativeAnthropicMessagesStream(ctx, req, portalLib.WithCompatMode())
	logger.Info("Anthropic compat Messages 流式请求已启动", "model", req.Model)
	return stream
}

// GeminiCompatGenerateContent 处理 Gemini compat generateContent 非流式请求。
func (s *service) GeminiCompatGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error) {
	logger := s.logger.WithGroup("gemini_compat_generate_content")
	logger.Info("开始执行 Gemini compat generateContent 非流式请求", "model", req.Model)

	resp, err := s.portalService.NativeGeminiGenerateContent(ctx, req, portalLib.WithCompatMode())
	if err != nil {
		logger.Error("Gemini compat generateContent 非流式请求失败", "error", err, "model", req.Model)
		return nil, fmt.Errorf("处理 Gemini compat generateContent 请求失败：%w", err)
	}

	logger.Info("Gemini compat generateContent 非流式请求成功", "model", req.Model)
	return resp, nil
}

// GeminiCompatGenerateContentStream 处理 Gemini compat streamGenerateContent 流式请求。
func (s *service) GeminiCompatGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent {
	logger := s.logger.WithGroup("gemini_compat_generate_content_stream")
	logger.Info("开始执行 Gemini compat streamGenerateContent 流式请求", "model", req.Model)

	stream := s.portalService.NativeGeminiStreamGenerateContent(ctx, req, portalLib.WithCompatMode())
	logger.Info("Gemini compat streamGenerateContent 流式请求已启动", "model", req.Model)
	return stream
}

// OpenAICompatChatCompletion 处理 OpenAI compat Chat Completions 非流式请求。
func (s *service) OpenAICompatChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
	logger := s.logger.WithGroup("openai_compat_chat_completion")
	logger.Info("开始执行 OpenAI compat Chat Completions 非流式请求", "model", req.Model)

	resp, err := s.portalService.NativeOpenAIChatCompletion(ctx, req, portalLib.WithCompatMode())
	if err != nil {
		logger.Error("OpenAI compat Chat Completions 非流式请求失败", "error", err, "model", req.Model)
		return nil, fmt.Errorf("处理 OpenAI compat Chat Completions 请求失败：%w", err)
	}

	logger.Info("OpenAI compat Chat Completions 非流式请求成功", "model", req.Model)
	return resp, nil
}

// OpenAICompatChatCompletionStream 处理 OpenAI compat Chat Completions 流式请求。
func (s *service) OpenAICompatChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent {
	logger := s.logger.WithGroup("openai_compat_chat_completion_stream")
	logger.Info("开始执行 OpenAI compat Chat Completions 流式请求", "model", req.Model)

	stream := s.portalService.NativeOpenAIChatCompletionStream(ctx, req, portalLib.WithCompatMode())
	logger.Info("OpenAI compat Chat Completions 流式请求已启动", "model", req.Model)
	return stream
}

// OpenAICompatResponses 处理 OpenAI compat Responses 非流式请求。
func (s *service) OpenAICompatResponses(ctx context.Context, req *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error) {
	logger := s.logger.WithGroup("openai_compat_responses")
	logger.Info("开始执行 OpenAI compat Responses 非流式请求", "model", req.Model)

	resp, err := s.portalService.NativeOpenAIResponses(ctx, req, portalLib.WithCompatMode())
	if err != nil {
		logger.Error("OpenAI compat Responses 非流式请求失败", "error", err, "model", req.Model)
		return nil, fmt.Errorf("处理 OpenAI compat Responses 请求失败：%w", err)
	}

	logger.Info("OpenAI compat Responses 非流式请求成功", "model", req.Model)
	return resp, nil
}

// OpenAICompatResponsesStream 处理 OpenAI compat Responses 流式请求。
func (s *service) OpenAICompatResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request) <-chan *openaiResponsesTypes.StreamEvent {
	logger := s.logger.WithGroup("openai_compat_responses_stream")
	logger.Info("开始执行 OpenAI compat Responses 流式请求", "model", req.Model)

	stream := s.portalService.NativeOpenAIResponsesStream(ctx, req, portalLib.WithCompatMode())
	logger.Info("OpenAI compat Responses 流式请求已启动", "model", req.Model)
	return stream
}

// OpenAINativeChatCompletion 处理 OpenAI native Chat Completions 非流式请求。
func (s *service) OpenAINativeChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
	logger := s.logger.WithGroup("openai_native_chat_completion")
	logger.Info("开始执行 OpenAI native Chat Completions 非流式请求", "model", req.Model)

	resp, err := s.portalService.NativeOpenAIChatCompletion(ctx, req)
	if err != nil {
		logger.Error("OpenAI native Chat Completions 非流式请求失败", "error", err, "model", req.Model)
		return nil, fmt.Errorf("处理 OpenAI native Chat Completions 请求失败：%w", err)
	}

	logger.Info("OpenAI native Chat Completions 非流式请求成功", "model", req.Model)
	return resp, nil
}
