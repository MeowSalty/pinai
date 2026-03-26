package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

// AnthropicStreamResult 定义 Anthropic 流式事件的最小收口结果。
//
// 网关负责将上游事件标准化为统一字段，handler 仅需负责 SSE/HTTP 协议写回。
type AnthropicStreamResult struct {
	Event        *anthropicTypes.StreamEvent
	EventType    anthropicTypes.StreamEventType
	ErrorMessage string
	Done         bool
}

// OpenAIChatStreamResult 定义 OpenAI Chat 流式事件的最小收口结果。
//
// 网关负责将上游事件标准化为统一字段，handler 仅需负责 SSE/HTTP 协议写回。
type OpenAIChatStreamResult struct {
	Event        *openaiChatTypes.StreamEvent
	ErrorMessage string
	Done         bool
}

// OpenAIResponsesStreamResult 定义 OpenAI Responses 流式事件的最小收口结果。
//
// 网关负责将上游事件标准化为统一字段，handler 仅需负责 SSE/HTTP 协议写回。
type OpenAIResponsesStreamResult struct {
	Event        *openaiResponsesTypes.StreamEvent
	ErrorMessage string
	Done         bool
}

// GeminiStreamResult 定义 Gemini 流式事件的最小收口结果。
//
// 网关负责将上游事件标准化为统一字段，handler 仅需负责 SSE/HTTP 协议写回。
type GeminiStreamResult struct {
	Event        *geminiTypes.StreamEvent
	ErrorMessage string
	Done         bool
}

// DataPlaneError 定义数据面错误映射的最小收口结果。
//
// 第一轮仅收口状态码与对外错误信息，避免在 handler 侧重复分支。
type DataPlaneError struct {
	StatusCode int
	Message    string
}

// Service 定义数据面网关应用服务接口。
//
// 当前仅提供第一批最小落地链路：OpenAI compat Chat Completions。
type Service interface {
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

func anthropicStreamEventType(event *anthropicTypes.StreamEvent) (anthropicTypes.StreamEventType, bool) {
	if event == nil {
		return "", false
	}

	switch {
	case event.MessageStart != nil:
		return event.MessageStart.Type, true
	case event.MessageDelta != nil:
		return event.MessageDelta.Type, true
	case event.MessageStop != nil:
		return event.MessageStop.Type, true
	case event.ContentBlockStart != nil:
		return event.ContentBlockStart.Type, true
	case event.ContentBlockDelta != nil:
		return event.ContentBlockDelta.Type, true
	case event.ContentBlockStop != nil:
		return event.ContentBlockStop.Type, true
	case event.Ping != nil:
		return event.Ping.Type, true
	case event.Error != nil:
		return event.Error.Type, true
	default:
		return "", false
	}
}

func anthropicStreamErrorMessage(event *anthropicTypes.StreamEvent) (string, bool) {
	if event == nil || event.Error == nil {
		return "", false
	}

	return event.Error.Error.Error.Message, true
}

func normalizeAnthropicStream(streamCtx streamLogContext, source <-chan *anthropicTypes.StreamEvent) <-chan AnthropicStreamResult {
	out := make(chan AnthropicStreamResult)
	go func() {
		defer close(out)

		streamCtx.logger.Info("开始消费 Anthropic 流式结果", streamCtx.attrs...)
		for event := range source {
			eventType, ok := anthropicStreamEventType(event)
			if !ok {
				streamCtx.logger.Debug("忽略无法识别的 Anthropic 流式事件", streamCtx.attrs...)
				continue
			}

			result := AnthropicStreamResult{
				Event:     event,
				EventType: eventType,
			}

			if message, hasError := anthropicStreamErrorMessage(event); hasError {
				result.ErrorMessage = message
				result.Done = true
			}

			if event.MessageStop != nil {
				result.Done = true
			}

			streamCtx.logger.Debug("Anthropic 流式事件已收口",
				append(streamCtx.attrs,
					"event_type", result.EventType,
					"done", result.Done,
					"has_error", result.ErrorMessage != "",
				)...,
			)

			out <- result
			if result.Done {
				streamCtx.logger.Info("Anthropic 流式结束条件满足",
					append(streamCtx.attrs,
						"event_type", result.EventType,
						"has_error", result.ErrorMessage != "",
					)...,
				)
				return
			}
		}

		streamCtx.logger.Info("Anthropic 流式上游通道关闭", streamCtx.attrs...)
	}()

	return out
}

func geminiModelFromRequest(req *geminiTypes.Request) string {
	if req == nil {
		return ""
	}

	return req.Model
}

func geminiStreamDone(event *geminiTypes.StreamEvent) bool {
	if event == nil {
		return false
	}

	if event.PromptFeedback != nil && event.PromptFeedback.BlockReason != "" {
		return true
	}

	for _, candidate := range event.Candidates {
		if candidate.FinishReason != "" && candidate.FinishReason != geminiTypes.FinishReasonUnspecified {
			return true
		}
	}

	return false
}

func geminiStreamErrorMessage(event *geminiTypes.StreamEvent) (string, bool) {
	if event == nil {
		return "", false
	}

	if event.PromptFeedback != nil && event.PromptFeedback.BlockReason != "" {
		return fmt.Sprintf("提示内容被拦截：%s", event.PromptFeedback.BlockReason), true
	}

	for _, candidate := range event.Candidates {
		if candidate.FinishMessage != "" {
			return candidate.FinishMessage, true
		}
	}

	return "", false
}

func normalizeGeminiStream(streamCtx streamLogContext, source <-chan *geminiTypes.StreamEvent) <-chan GeminiStreamResult {
	out := make(chan GeminiStreamResult)
	go func() {
		defer close(out)

		streamCtx.logger.Info("开始消费 Gemini 流式结果", streamCtx.attrs...)
		for event := range source {
			if event == nil {
				streamCtx.logger.Debug("忽略空 Gemini 流式事件", streamCtx.attrs...)
				continue
			}

			result := GeminiStreamResult{
				Event: event,
				Done:  geminiStreamDone(event),
			}

			if message, hasError := geminiStreamErrorMessage(event); hasError {
				result.ErrorMessage = message
				result.Done = true
			}

			streamCtx.logger.Debug("Gemini 流式事件已收口",
				append(streamCtx.attrs,
					"done", result.Done,
					"has_error", result.ErrorMessage != "",
				)...,
			)

			out <- result
			if result.Done {
				streamCtx.logger.Info("Gemini 流式结束条件满足",
					append(streamCtx.attrs,
						"has_error", result.ErrorMessage != "",
					)...,
				)
				return
			}
		}

		streamCtx.logger.Info("Gemini 流式上游通道关闭", streamCtx.attrs...)
	}()

	return out
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

func openAIResponsesStreamDone(event *openaiResponsesTypes.StreamEvent) bool {
	if event == nil {
		return false
	}

	return event.Completed != nil || event.Failed != nil || event.Incomplete != nil
}

func openAIResponsesStreamErrorMessage(event *openaiResponsesTypes.StreamEvent) (string, bool) {
	if event == nil || event.Error == nil {
		return "", false
	}

	return event.Error.Message, true
}

func normalizeOpenAIResponsesStream(streamCtx streamLogContext, source <-chan *openaiResponsesTypes.StreamEvent) <-chan OpenAIResponsesStreamResult {
	out := make(chan OpenAIResponsesStreamResult)
	go func() {
		defer close(out)

		streamCtx.logger.Info("开始消费 OpenAI Responses 流式结果", streamCtx.attrs...)
		for event := range source {
			if event == nil {
				streamCtx.logger.Debug("忽略空 OpenAI Responses 流式事件", streamCtx.attrs...)
				continue
			}

			result := OpenAIResponsesStreamResult{
				Event: event,
				Done:  openAIResponsesStreamDone(event),
			}

			if message, hasError := openAIResponsesStreamErrorMessage(event); hasError {
				result.ErrorMessage = message
				result.Done = true
			}

			streamCtx.logger.Debug("OpenAI Responses 流式事件已收口",
				append(streamCtx.attrs,
					"done", result.Done,
					"has_error", result.ErrorMessage != "",
				)...,
			)

			out <- result
			if result.Done {
				streamCtx.logger.Info("OpenAI Responses 流式结束条件满足",
					append(streamCtx.attrs,
						"has_error", result.ErrorMessage != "",
					)...,
				)
				return
			}
		}

		streamCtx.logger.Info("OpenAI Responses 流式上游通道关闭", streamCtx.attrs...)
	}()

	return out
}

func openAIChatStreamDone(event *openaiChatTypes.StreamEvent) bool {
	if event == nil {
		return false
	}

	for _, choice := range event.Choices {
		if choice.FinishReason != nil {
			return true
		}
	}

	return false
}

func normalizeOpenAIChatStream(streamCtx streamLogContext, source <-chan *openaiChatTypes.StreamEvent) <-chan OpenAIChatStreamResult {
	out := make(chan OpenAIChatStreamResult)
	go func() {
		defer close(out)

		streamCtx.logger.Info("开始消费 OpenAI Chat 流式结果", streamCtx.attrs...)
		for event := range source {
			if event == nil {
				streamCtx.logger.Debug("忽略空 OpenAI Chat 流式事件", streamCtx.attrs...)
				continue
			}

			result := OpenAIChatStreamResult{
				Event: event,
				Done:  openAIChatStreamDone(event),
			}

			streamCtx.logger.Debug("OpenAI Chat 流式事件已收口",
				append(streamCtx.attrs,
					"done", result.Done,
				)...,
			)

			out <- result
			if result.Done {
				streamCtx.logger.Info("OpenAI Chat 流式结束条件满足", streamCtx.attrs...)
				return
			}
		}

		streamCtx.logger.Info("OpenAI Chat 流式上游通道关闭", streamCtx.attrs...)
	}()

	return out
}

// MapDataPlaneError 对数据面错误进行第一轮统一映射。
func (s *service) MapDataPlaneError(err error, fallbackAction string) DataPlaneError {
	if err == nil {
		return DataPlaneError{StatusCode: http.StatusInternalServerError, Message: fallbackAction}
	}

	lowerMsg := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(lowerMsg, "timeout"), strings.Contains(lowerMsg, "deadline"):
		return DataPlaneError{StatusCode: http.StatusGatewayTimeout, Message: "上游请求超时"}
	case errors.Is(err, context.Canceled):
		return DataPlaneError{StatusCode: http.StatusRequestTimeout, Message: "请求已取消"}
	case strings.Contains(lowerMsg, "429"), strings.Contains(lowerMsg, "rate limit"), strings.Contains(lowerMsg, "too many requests"), strings.Contains(lowerMsg, "quota"):
		return DataPlaneError{StatusCode: http.StatusTooManyRequests, Message: "请求过于频繁，请稍后重试"}
	case strings.Contains(lowerMsg, "401"), strings.Contains(lowerMsg, "403"), strings.Contains(lowerMsg, "unauthorized"), strings.Contains(lowerMsg, "forbidden"), strings.Contains(lowerMsg, "authentication"):
		return DataPlaneError{StatusCode: http.StatusUnauthorized, Message: "鉴权失败"}
	case strings.Contains(lowerMsg, "404"), strings.Contains(lowerMsg, "not found"):
		return DataPlaneError{StatusCode: http.StatusNotFound, Message: "请求资源不存在"}
	default:
		return DataPlaneError{StatusCode: http.StatusInternalServerError, Message: fmt.Sprintf("%s：%v", fallbackAction, err)}
	}
}

// AnthropicNativeMessagesStream 处理 Anthropic native Messages 流式请求。
func (s *service) AnthropicNativeMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent {
	streamCtx := newStreamLogContext(s.logger, "anthropic_native_messages_stream", "Anthropic native Messages", anthropicModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *anthropicTypes.StreamEvent {
		return s.portalService.NativeAnthropicMessagesStream(ctx, req)
	})
}

// AnthropicNativeMessagesStreamResult 处理 Anthropic native Messages 流式请求并返回最小收口结果。
func (s *service) AnthropicNativeMessagesStreamResult(ctx context.Context, req *anthropicTypes.Request) <-chan AnthropicStreamResult {
	streamCtx := newStreamLogContext(s.logger, "anthropic_native_messages_stream_result", "Anthropic native Messages", anthropicModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *anthropicTypes.StreamEvent {
		return s.portalService.NativeAnthropicMessagesStream(ctx, req)
	})

	return normalizeAnthropicStream(streamCtx, rawStream)
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

// GeminiNativeGenerateContentStreamResult 处理 Gemini native streamGenerateContent 流式请求并返回最小收口结果。
func (s *service) GeminiNativeGenerateContentStreamResult(ctx context.Context, req *geminiTypes.Request) <-chan GeminiStreamResult {
	streamCtx := newStreamLogContext(s.logger, "gemini_native_generate_content_stream_result", "Gemini native streamGenerateContent", geminiModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *geminiTypes.StreamEvent {
		return s.portalService.NativeGeminiStreamGenerateContent(ctx, req)
	})

	return normalizeGeminiStream(streamCtx, rawStream)
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

// AnthropicCompatMessagesStreamResult 处理 Anthropic compat Messages 流式请求并返回最小收口结果。
func (s *service) AnthropicCompatMessagesStreamResult(ctx context.Context, req *anthropicTypes.Request) <-chan AnthropicStreamResult {
	streamCtx := newStreamLogContext(s.logger, "anthropic_compat_messages_stream_result", "Anthropic compat Messages", anthropicModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *anthropicTypes.StreamEvent {
		return s.portalService.NativeAnthropicMessagesStream(ctx, req, portalLib.WithCompatMode())
	})

	return normalizeAnthropicStream(streamCtx, rawStream)
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

// GeminiCompatGenerateContentStreamResult 处理 Gemini compat streamGenerateContent 流式请求并返回最小收口结果。
func (s *service) GeminiCompatGenerateContentStreamResult(ctx context.Context, req *geminiTypes.Request) <-chan GeminiStreamResult {
	streamCtx := newStreamLogContext(s.logger, "gemini_compat_generate_content_stream_result", "Gemini compat streamGenerateContent", geminiModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *geminiTypes.StreamEvent {
		return s.portalService.NativeGeminiStreamGenerateContent(ctx, req, portalLib.WithCompatMode())
	})

	return normalizeGeminiStream(streamCtx, rawStream)
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

// OpenAICompatChatCompletionStreamResult 处理 OpenAI compat Chat Completions 流式请求并返回最小收口结果。
func (s *service) OpenAICompatChatCompletionStreamResult(ctx context.Context, req *openaiChatTypes.Request) <-chan OpenAIChatStreamResult {
	streamCtx := newStreamLogContext(s.logger, "openai_compat_chat_completion_stream_result", "OpenAI compat Chat Completions", openAIChatModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *openaiChatTypes.StreamEvent {
		return s.portalService.NativeOpenAIChatCompletionStream(ctx, req, portalLib.WithCompatMode())
	})

	return normalizeOpenAIChatStream(streamCtx, rawStream)
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

// OpenAICompatResponsesStreamResult 处理 OpenAI compat Responses 流式请求并返回最小收口结果。
func (s *service) OpenAICompatResponsesStreamResult(ctx context.Context, req *openaiResponsesTypes.Request) <-chan OpenAIResponsesStreamResult {
	streamCtx := newStreamLogContext(s.logger, "openai_compat_responses_stream_result", "OpenAI compat Responses", openAIResponsesModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *openaiResponsesTypes.StreamEvent {
		return s.portalService.NativeOpenAIResponsesStream(ctx, req, portalLib.WithCompatMode())
	})

	return normalizeOpenAIResponsesStream(streamCtx, rawStream)
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

// OpenAINativeChatCompletionStreamResult 处理 OpenAI native Chat Completions 流式请求并返回最小收口结果。
func (s *service) OpenAINativeChatCompletionStreamResult(ctx context.Context, req *openaiChatTypes.Request) <-chan OpenAIChatStreamResult {
	streamCtx := newStreamLogContext(s.logger, "openai_native_chat_completion_stream_result", "OpenAI native Chat Completions", openAIChatModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *openaiChatTypes.StreamEvent {
		return s.portalService.NativeOpenAIChatCompletionStream(ctx, req)
	})

	return normalizeOpenAIChatStream(streamCtx, rawStream)
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

// OpenAINativeResponsesStreamResult 处理 OpenAI native Responses 流式请求并返回最小收口结果。
func (s *service) OpenAINativeResponsesStreamResult(ctx context.Context, req *openaiResponsesTypes.Request) <-chan OpenAIResponsesStreamResult {
	streamCtx := newStreamLogContext(s.logger, "openai_native_responses_stream_result", "OpenAI native Responses", openAIResponsesModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *openaiResponsesTypes.StreamEvent {
		return s.portalService.NativeOpenAIResponsesStream(ctx, req)
	})

	return normalizeOpenAIResponsesStream(streamCtx, rawStream)
}
