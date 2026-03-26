package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/services/portal"
	portalLib "github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

type openAIChatCompletionInvoker func(context.Context, *openaiChatTypes.Request) (*openaiChatTypes.Response, error)
type openAIResponsesInvoker func(context.Context, *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error)
type anthropicMessagesInvoker func(context.Context, *anthropicTypes.Request) (*anthropicTypes.Response, error)
type geminiGenerateContentInvoker func(context.Context, *geminiTypes.Request) (*geminiTypes.Response, error)

type streamLogContext struct {
	logger *slog.Logger
	attrs  []any
}

// Service 定义数据面网关应用服务接口。
//
// 当前仅提供第一批最小落地链路：OpenAI compat Chat Completions。
type Service interface {
	// AnthropicNativeMessages 处理 Anthropic native Messages 非流式请求。
	AnthropicNativeMessages(ctx context.Context, req *anthropicTypes.Request) (*anthropicTypes.Response, error)

	// AnthropicNativeMessagesStream 处理 Anthropic native Messages 流式请求。
	AnthropicNativeMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent

	// GeminiNativeGenerateContent 处理 Gemini native generateContent 非流式请求。
	GeminiNativeGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error)

	// GeminiNativeGenerateContentStream 处理 Gemini native streamGenerateContent 流式请求。
	GeminiNativeGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent

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

	// OpenAINativeChatCompletionStream 处理 OpenAI native Chat Completions 流式请求。
	OpenAINativeChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent

	// OpenAINativeResponses 处理 OpenAI native Responses 非流式请求。
	OpenAINativeResponses(ctx context.Context, req *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error)

	// OpenAINativeResponsesStream 处理 OpenAI native Responses 流式请求。
	OpenAINativeResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request) <-chan *openaiResponsesTypes.StreamEvent
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

// AnthropicNativeMessages 处理 Anthropic native Messages 非流式请求。
func (s *service) AnthropicNativeMessages(ctx context.Context, req *anthropicTypes.Request) (*anthropicTypes.Response, error) {
	return s.executeAnthropicMessages(ctx, req, "anthropic_native_messages", "Anthropic native Messages", func(inCtx context.Context, inReq *anthropicTypes.Request) (*anthropicTypes.Response, error) {
		return s.portalService.NativeAnthropicMessages(inCtx, inReq)
	})
}

func (s *service) executeAnthropicMessages(ctx context.Context, req *anthropicTypes.Request, loggerGroup, requestName string, invoker anthropicMessagesInvoker) (*anthropicTypes.Response, error) {
	logger := s.logger.WithGroup(loggerGroup)
	modelName := ""
	if req != nil {
		modelName = req.Model
	}

	logger.Info("开始执行非流式请求", "request_name", requestName, "model", modelName)

	startTime := time.Now()
	resp, err := invoker(ctx, req)
	duration := time.Since(startTime)
	if err != nil {
		logger.Error("非流式请求失败", "request_name", requestName, "error", err, "duration", duration, "model", modelName)
		return nil, fmt.Errorf("处理 %s 请求失败：%w", requestName, err)
	}

	logger.Info("非流式请求成功", "request_name", requestName, "duration", duration, "model", modelName)
	return resp, nil
}

func newStreamLogContext(baseLogger *slog.Logger, loggerGroup, requestName, modelName string) streamLogContext {
	return streamLogContext{
		logger: baseLogger.WithGroup(loggerGroup),
		attrs:  []any{"request_name", requestName, "model", modelName},
	}
}

func startStream[T any](streamCtx streamLogContext, invoker func() <-chan T) <-chan T {
	streamCtx.logger.Info("开始执行流式请求", streamCtx.attrs...)
	stream := invoker()
	streamCtx.logger.Info("流式请求已启动", streamCtx.attrs...)
	return stream
}

func anthropicModelFromRequest(req *anthropicTypes.Request) string {
	if req == nil {
		return ""
	}

	return req.Model
}

func geminiModelFromRequest(req *geminiTypes.Request) string {
	if req == nil {
		return ""
	}

	return req.Model
}

func openAIChatModelFromRequest(req *openaiChatTypes.Request) string {
	if req == nil {
		return ""
	}

	return req.Model
}

func openAIResponsesModelFromRequest(req *openaiResponsesTypes.Request) string {
	if req == nil || req.Model == nil {
		return ""
	}

	return *req.Model
}

// AnthropicNativeMessagesStream 处理 Anthropic native Messages 流式请求。
func (s *service) AnthropicNativeMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "anthropic_native_messages_stream", "Anthropic native Messages", anthropicModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *anthropicTypes.StreamEvent {
		return s.portalService.NativeAnthropicMessagesStream(ctx, req)
	})
}

// GeminiNativeGenerateContent 处理 Gemini native generateContent 非流式请求。
func (s *service) GeminiNativeGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error) {
	return s.executeGeminiGenerateContent(ctx, req, "gemini_native_generate_content", "Gemini native generateContent", func(inCtx context.Context, inReq *geminiTypes.Request) (*geminiTypes.Response, error) {
		return s.portalService.NativeGeminiGenerateContent(inCtx, inReq)
	})
}

func (s *service) executeGeminiGenerateContent(ctx context.Context, req *geminiTypes.Request, loggerGroup, requestName string, invoker geminiGenerateContentInvoker) (*geminiTypes.Response, error) {
	logger := s.logger.WithGroup(loggerGroup)
	modelName := ""
	if req != nil {
		modelName = req.Model
	}

	logger.Info("开始执行非流式请求", "request_name", requestName, "model", modelName)

	startTime := time.Now()
	resp, err := invoker(ctx, req)
	duration := time.Since(startTime)
	if err != nil {
		logger.Error("非流式请求失败", "request_name", requestName, "error", err, "duration", duration, "model", modelName)
		return nil, fmt.Errorf("处理 %s 请求失败：%w", requestName, err)
	}

	logger.Info("非流式请求成功", "request_name", requestName, "duration", duration, "model", modelName)
	return resp, nil
}

// GeminiNativeGenerateContentStream 处理 Gemini native streamGenerateContent 流式请求。
func (s *service) GeminiNativeGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "gemini_native_generate_content_stream", "Gemini native streamGenerateContent", geminiModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *geminiTypes.StreamEvent {
		return s.portalService.NativeGeminiStreamGenerateContent(ctx, req)
	})
}

// AnthropicCompatMessages 处理 Anthropic compat Messages 非流式请求。
func (s *service) AnthropicCompatMessages(ctx context.Context, req *anthropicTypes.Request) (*anthropicTypes.Response, error) {
	return s.executeAnthropicMessages(ctx, req, "anthropic_compat_messages", "Anthropic compat Messages", func(inCtx context.Context, inReq *anthropicTypes.Request) (*anthropicTypes.Response, error) {
		return s.portalService.NativeAnthropicMessages(inCtx, inReq, portalLib.WithCompatMode())
	})
}

// AnthropicCompatMessagesStream 处理 Anthropic compat Messages 流式请求。
func (s *service) AnthropicCompatMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "anthropic_compat_messages_stream", "Anthropic compat Messages", anthropicModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *anthropicTypes.StreamEvent {
		return s.portalService.NativeAnthropicMessagesStream(ctx, req, portalLib.WithCompatMode())
	})
}

// GeminiCompatGenerateContent 处理 Gemini compat generateContent 非流式请求。
func (s *service) GeminiCompatGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error) {
	return s.executeGeminiGenerateContent(ctx, req, "gemini_compat_generate_content", "Gemini compat generateContent", func(inCtx context.Context, inReq *geminiTypes.Request) (*geminiTypes.Response, error) {
		return s.portalService.NativeGeminiGenerateContent(inCtx, inReq, portalLib.WithCompatMode())
	})
}

// GeminiCompatGenerateContentStream 处理 Gemini compat streamGenerateContent 流式请求。
func (s *service) GeminiCompatGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "gemini_compat_generate_content_stream", "Gemini compat streamGenerateContent", geminiModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *geminiTypes.StreamEvent {
		return s.portalService.NativeGeminiStreamGenerateContent(ctx, req, portalLib.WithCompatMode())
	})
}

// OpenAICompatChatCompletion 处理 OpenAI compat Chat Completions 非流式请求。
func (s *service) OpenAICompatChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
	return s.executeOpenAIChatCompletion(ctx, req, "openai_compat_chat_completion", "OpenAI compat Chat Completions", func(inCtx context.Context, inReq *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
		return s.portalService.NativeOpenAIChatCompletion(inCtx, inReq, portalLib.WithCompatMode())
	})
}

// OpenAICompatChatCompletionStream 处理 OpenAI compat Chat Completions 流式请求。
func (s *service) OpenAICompatChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "openai_compat_chat_completion_stream", "OpenAI compat Chat Completions", openAIChatModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *openaiChatTypes.StreamEvent {
		return s.portalService.NativeOpenAIChatCompletionStream(ctx, req, portalLib.WithCompatMode())
	})
}

// OpenAICompatResponses 处理 OpenAI compat Responses 非流式请求。
func (s *service) OpenAICompatResponses(ctx context.Context, req *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error) {
	return s.executeOpenAIResponses(ctx, req, "openai_compat_responses", "OpenAI compat Responses", func(inCtx context.Context, inReq *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error) {
		return s.portalService.NativeOpenAIResponses(inCtx, inReq, portalLib.WithCompatMode())
	})
}

// OpenAICompatResponsesStream 处理 OpenAI compat Responses 流式请求。
func (s *service) OpenAICompatResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request) <-chan *openaiResponsesTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "openai_compat_responses_stream", "OpenAI compat Responses", openAIResponsesModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *openaiResponsesTypes.StreamEvent {
		return s.portalService.NativeOpenAIResponsesStream(ctx, req, portalLib.WithCompatMode())
	})
}

// OpenAINativeChatCompletion 处理 OpenAI native Chat Completions 非流式请求。
func (s *service) OpenAINativeChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
	return s.executeOpenAIChatCompletion(ctx, req, "openai_native_chat_completion", "OpenAI native Chat Completions", func(inCtx context.Context, inReq *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
		return s.portalService.NativeOpenAIChatCompletion(inCtx, inReq)
	})
}

func (s *service) executeOpenAIChatCompletion(ctx context.Context, req *openaiChatTypes.Request, loggerGroup, requestName string, invoker openAIChatCompletionInvoker) (*openaiChatTypes.Response, error) {
	logger := s.logger.WithGroup(loggerGroup)
	logger.Info("开始执行非流式请求", "request_name", requestName, "model", req.Model)

	startTime := time.Now()
	resp, err := invoker(ctx, req)
	duration := time.Since(startTime)
	if err != nil {
		logger.Error("非流式请求失败", "request_name", requestName, "error", err, "duration", duration, "model", req.Model)
		return nil, fmt.Errorf("处理 %s 请求失败：%w", requestName, err)
	}

	logger.Info("非流式请求成功", "request_name", requestName, "duration", duration, "model", req.Model)
	return resp, nil
}

func (s *service) executeOpenAIResponses(ctx context.Context, req *openaiResponsesTypes.Request, loggerGroup, requestName string, invoker openAIResponsesInvoker) (*openaiResponsesTypes.Response, error) {
	logger := s.logger.WithGroup(loggerGroup)
	modelName := ""
	if req != nil && req.Model != nil {
		modelName = *req.Model
	}
	logger.Info("开始执行非流式请求", "request_name", requestName, "model", modelName)

	startTime := time.Now()
	resp, err := invoker(ctx, req)
	duration := time.Since(startTime)
	if err != nil {
		logger.Error("非流式请求失败", "request_name", requestName, "error", err, "duration", duration, "model", modelName)
		return nil, fmt.Errorf("处理 %s 请求失败：%w", requestName, err)
	}

	logger.Info("非流式请求成功", "request_name", requestName, "duration", duration, "model", modelName)
	return resp, nil
}

// OpenAINativeChatCompletionStream 处理 OpenAI native Chat Completions 流式请求。
func (s *service) OpenAINativeChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "openai_native_chat_completion_stream", "OpenAI native Chat Completions", openAIChatModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *openaiChatTypes.StreamEvent {
		return s.portalService.NativeOpenAIChatCompletionStream(ctx, req)
	})
}

// OpenAINativeResponses 处理 OpenAI native Responses 非流式请求。
func (s *service) OpenAINativeResponses(ctx context.Context, req *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error) {
	return s.executeOpenAIResponses(ctx, req, "openai_native_responses", "OpenAI native Responses", func(inCtx context.Context, inReq *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error) {
		return s.portalService.NativeOpenAIResponses(inCtx, inReq)
	})
}

// OpenAINativeResponsesStream 处理 OpenAI native Responses 流式请求。
func (s *service) OpenAINativeResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request) <-chan *openaiResponsesTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "openai_native_responses_stream", "OpenAI native Responses", openAIResponsesModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *openaiResponsesTypes.StreamEvent {
		return s.portalService.NativeOpenAIResponsesStream(ctx, req)
	})
}
