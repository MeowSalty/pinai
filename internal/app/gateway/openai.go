package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	portalLib "github.com/MeowSalty/portal"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

type openAIChatCompletionInvoker func(context.Context, *openaiChatTypes.Request) (*openaiChatTypes.Response, error)
type openAIResponsesInvoker func(context.Context, *openaiResponsesTypes.Request) (*openaiResponsesTypes.Response, error)

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

func openAIStatusFromCodeOrMessage(code, message string) int {
	code = strings.ToLower(strings.TrimSpace(code))
	message = strings.ToLower(strings.TrimSpace(message))

	switch {
	case strings.Contains(code, "rate_limit"), strings.Contains(code, "quota"), strings.Contains(message, "rate limit"), strings.Contains(message, "too many requests"):
		return http.StatusTooManyRequests
	case strings.Contains(code, "invalid"), strings.Contains(code, "bad_request"), strings.Contains(message, "invalid request"), strings.Contains(message, "bad request"):
		return http.StatusBadRequest
	case strings.Contains(code, "authentication"), strings.Contains(code, "unauthorized"), strings.Contains(message, "unauthorized"), strings.Contains(message, "authentication"):
		return http.StatusUnauthorized
	case strings.Contains(code, "permission"), strings.Contains(code, "forbidden"), strings.Contains(message, "forbidden"):
		return http.StatusForbidden
	case strings.Contains(code, "not_found"), strings.Contains(message, "not found"):
		return http.StatusNotFound
	case strings.Contains(code, "timeout"), strings.Contains(message, "timeout"):
		return http.StatusGatewayTimeout
	default:
		return http.StatusBadGateway
	}
}

func openAIMapStringField(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}

	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}

	text := strings.TrimSpace(fmt.Sprintf("%v", raw))
	if text == "<nil>" {
		return ""
	}

	return text
}

func openAIResponseProtocolErrorFromFailed(event *openaiResponsesTypes.StreamEvent) (*DataPlaneError, bool) {
	if event == nil || event.Failed == nil {
		return nil, false
	}

	failed := event.Failed.Response
	if failed.Error == nil {
		mapped := DataPlaneError{
			StatusCode:             http.StatusBadGateway,
			Message:                "OpenAI Responses 流式返回 failed 终止事件",
			Provider:               "openai",
			ErrorType:              "response_failed",
			Raw:                    event.Failed,
			Retryable:              isRetryableByStatus(http.StatusBadGateway),
			ShouldProxyAsHTTPError: true,
		}
		if failed.Status != nil {
			mapped.ErrorCode = strings.TrimSpace(*failed.Status)
		}
		return &mapped, true
	}

	errCode := strings.TrimSpace(string(failed.Error.Code))
	errMessage := strings.TrimSpace(failed.Error.Message)
	if errMessage == "" {
		errMessage = "OpenAI Responses 流式返回 failed 错误"
	}
	statusCode := openAIStatusFromCodeOrMessage(errCode, errMessage)

	mapped := DataPlaneError{
		StatusCode:             statusCode,
		Message:                errMessage,
		Provider:               "openai",
		ErrorType:              "response_failed",
		ErrorCode:              errCode,
		Raw:                    failed.Error,
		Retryable:              isRetryableByStatus(statusCode),
		ShouldProxyAsHTTPError: true,
	}

	return &mapped, true
}

func openAIResponsesProtocolError(event *openaiResponsesTypes.StreamEvent) (*DataPlaneError, bool) {
	if event == nil {
		return nil, false
	}

	if event.Error != nil {
		errCode := ""
		if event.Error.Code != nil {
			errCode = strings.TrimSpace(*event.Error.Code)
		}
		errMessage := strings.TrimSpace(event.Error.Message)
		if errMessage == "" {
			errMessage = "OpenAI Responses 流式返回错误事件"
		}

		mapped := DataPlaneError{
			StatusCode:             openAIStatusFromCodeOrMessage(errCode, errMessage),
			Message:                errMessage,
			Provider:               "openai",
			ErrorType:              "response_error_event",
			ErrorCode:              errCode,
			Raw:                    event.Error,
			Retryable:              isRetryableByStatus(openAIStatusFromCodeOrMessage(errCode, errMessage)),
			ShouldProxyAsHTTPError: true,
		}
		if event.Error.Param != nil {
			mapped.Param = strings.TrimSpace(*event.Error.Param)
		}
		return &mapped, true
	}

	return openAIResponseProtocolErrorFromFailed(event)
}

func openAIChatProtocolError(event *openaiChatTypes.StreamEvent) (*DataPlaneError, bool) {
	if event == nil {
		return nil, false
	}

	for _, choice := range event.Choices {
		errorPayload, ok := choice.Delta.ExtraFields["error"]
		if !ok {
			continue
		}

		errMap, ok := errorPayload.(map[string]any)
		if !ok {
			mapped := DataPlaneError{
				StatusCode:             http.StatusBadGateway,
				Message:                "OpenAI Chat 流式返回错误负载",
				Provider:               "openai",
				ErrorType:              "chat_error_event",
				Raw:                    errorPayload,
				Retryable:              isRetryableByStatus(http.StatusBadGateway),
				ShouldProxyAsHTTPError: true,
			}
			return &mapped, true
		}

		errMessage := strings.TrimSpace(fmt.Sprintf("%v", errMap["message"]))
		if errMessage == "<nil>" {
			errMessage = ""
		}
		if errMessage == "" {
			errMessage = "OpenAI Chat 流式返回错误事件"
		}
		errType := openAIMapStringField(errMap, "type")
		if errType == "" {
			errType = "chat_error_event"
		}
		errCode := openAIMapStringField(errMap, "code")
		param := openAIMapStringField(errMap, "param")
		statusCode := http.StatusBadGateway
		if statusRaw, ok := errMap["status"]; ok {
			s := strings.TrimSpace(fmt.Sprintf("%v", statusRaw))
			if s == "<nil>" {
				s = ""
			}
			if v, err := strconv.Atoi(s); err == nil && validHTTPStatus(v) {
				statusCode = v
			}
		}
		if statusCode == http.StatusBadGateway {
			statusCode = openAIStatusFromCodeOrMessage(errCode, errMessage)
		}

		mapped := DataPlaneError{
			StatusCode:             statusCode,
			Message:                errMessage,
			Provider:               "openai",
			ErrorType:              errType,
			ErrorCode:              errCode,
			Param:                  param,
			Raw:                    errorPayload,
			Retryable:              isRetryableByStatus(statusCode),
			ShouldProxyAsHTTPError: true,
		}
		return &mapped, true
	}

	return nil, false
}

func normalizeOpenAIResponsesStream(streamCtx streamLogContext, source <-chan *openaiResponsesTypes.StreamEvent) <-chan OpenAIResponsesStreamResult {
	out := make(chan OpenAIResponsesStreamResult)
	go func() {
		defer close(out)

		streamCtx.logger.Debug("开始消费 OpenAI Responses 流式结果", streamCtx.attrs...)
		for event := range source {
			if event == nil {
				streamCtx.logger.Debug("忽略空 OpenAI Responses 流式事件", streamCtx.attrs...)
				continue
			}

			result := OpenAIResponsesStreamResult{
				Event: event,
				Done:  openAIResponsesStreamDone(event),
			}

			if protocolError, hasProtocolError := openAIResponsesProtocolError(event); hasProtocolError {
				result.ProtocolError = protocolError
				result.Terminal = true
				result.Done = true
			}

			if result.Done && !result.Terminal {
				result.Terminal = true
			}

			streamCtx.logger.Debug("OpenAI Responses 流式事件已收口",
				append(streamCtx.attrs,
					"terminal", result.Terminal,
					"done", result.Done,
					"has_protocol_error", result.ProtocolError != nil,
				)...,
			)

			out <- result
			if result.Done {
				logStreamComplete(streamCtx, "done",
					"terminal", result.Terminal,
					"has_protocol_error", result.ProtocolError != nil,
				)
				return
			}
		}

		logStreamComplete(streamCtx, "channel_closed")
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

		streamCtx.logger.Debug("开始消费 OpenAI Chat 流式结果", streamCtx.attrs...)
		for event := range source {
			if event == nil {
				streamCtx.logger.Debug("忽略空 OpenAI Chat 流式事件", streamCtx.attrs...)
				continue
			}

			result := OpenAIChatStreamResult{
				Event: event,
				Done:  openAIChatStreamDone(event),
			}

			if protocolError, hasProtocolError := openAIChatProtocolError(event); hasProtocolError {
				result.ProtocolError = protocolError
				result.Terminal = true
				result.Done = true
			}

			if result.Done && !result.Terminal {
				result.Terminal = true
			}

			streamCtx.logger.Debug("OpenAI Chat 流式事件已收口",
				append(streamCtx.attrs,
					"terminal", result.Terminal,
					"done", result.Done,
					"has_protocol_error", result.ProtocolError != nil,
				)...,
			)

			out <- result
			if result.Done {
				logStreamComplete(streamCtx, "done",
					"terminal", result.Terminal,
					"has_protocol_error", result.ProtocolError != nil,
				)
				return
			}
		}

		logStreamComplete(streamCtx, "channel_closed")
	}()

	return out
}

// OpenAICompatChatCompletion 处理 OpenAI compat Chat Completions 非流式请求。
func (s *service) OpenAICompatChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
	return s.executeOpenAIChatCompletion(ctx, req, "openai_compat_chat_completion", "OpenAI compat Chat Completions", func(inCtx context.Context, inReq *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
		return s.portalService.NativeOpenAIChatCompletion(inCtx, inReq, portalLib.WithCompatMode())
	})
}

// OpenAICompatChatCompletionStream 处理 OpenAI compat Chat Completions 流式请求。
func (s *service) OpenAICompatChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent {
	streamCtx := newStreamLogContext(ctx, s.logger, "openai_compat_chat_completion_stream", "OpenAI compat Chat Completions", openAIChatModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *openaiChatTypes.StreamEvent {
		return s.portalService.NativeOpenAIChatCompletionStream(ctx, req, portalLib.WithCompatMode())
	})
}

// OpenAICompatChatCompletionStreamResult 处理 OpenAI compat Chat Completions 流式请求并返回最小收口结果。
func (s *service) OpenAICompatChatCompletionStreamResult(ctx context.Context, req *openaiChatTypes.Request) <-chan OpenAIChatStreamResult {
	streamCtx := newStreamLogContext(ctx, s.logger, "openai_compat_chat_completion_stream_result", "OpenAI compat Chat Completions", openAIChatModelFromRequest(req))
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
	streamCtx := newStreamLogContext(ctx, s.logger, "openai_compat_responses_stream", "OpenAI compat Responses", openAIResponsesModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *openaiResponsesTypes.StreamEvent {
		return s.portalService.NativeOpenAIResponsesStream(ctx, req, portalLib.WithCompatMode())
	})
}

// OpenAICompatResponsesStreamResult 处理 OpenAI compat Responses 流式请求并返回最小收口结果。
func (s *service) OpenAICompatResponsesStreamResult(ctx context.Context, req *openaiResponsesTypes.Request) <-chan OpenAIResponsesStreamResult {
	streamCtx := newStreamLogContext(ctx, s.logger, "openai_compat_responses_stream_result", "OpenAI compat Responses", openAIResponsesModelFromRequest(req))
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
	logger := enrichLoggerFromContext(ctx, s.logger.WithGroup(loggerGroup))
	modelName := ""
	if req != nil {
		modelName = req.Model
	}
	logger.Info("开始执行非流式请求", "request_name", requestName, "model", modelName)

	startTime := time.Now()
	resp, err := invoker(ctx, req)
	duration := time.Since(startTime)
	if err != nil {
		s.logNonStreamError(logger, requestName, err, duration, modelName)
		return nil, fmt.Errorf("处理 %s 请求失败：%w", requestName, err)
	}

	logger.Info("非流式请求成功", "request_name", requestName, "duration", duration, "model", modelName)
	return resp, nil
}

func (s *service) executeOpenAIResponses(ctx context.Context, req *openaiResponsesTypes.Request, loggerGroup, requestName string, invoker openAIResponsesInvoker) (*openaiResponsesTypes.Response, error) {
	logger := enrichLoggerFromContext(ctx, s.logger.WithGroup(loggerGroup))
	modelName := ""
	if req != nil && req.Model != nil {
		modelName = *req.Model
	}
	logger.Info("开始执行非流式请求", "request_name", requestName, "model", modelName)

	startTime := time.Now()
	resp, err := invoker(ctx, req)
	duration := time.Since(startTime)
	if err != nil {
		s.logNonStreamError(logger, requestName, err, duration, modelName)
		return nil, fmt.Errorf("处理 %s 请求失败：%w", requestName, err)
	}

	logger.Info("非流式请求成功", "request_name", requestName, "duration", duration, "model", modelName)
	return resp, nil
}

// OpenAINativeChatCompletionStream 处理 OpenAI native Chat Completions 流式请求。
func (s *service) OpenAINativeChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent {
	streamCtx := newStreamLogContext(ctx, s.logger, "openai_native_chat_completion_stream", "OpenAI native Chat Completions", openAIChatModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *openaiChatTypes.StreamEvent {
		return s.portalService.NativeOpenAIChatCompletionStream(ctx, req)
	})
}

// OpenAINativeChatCompletionStreamResult 处理 OpenAI native Chat Completions 流式请求并返回最小收口结果。
func (s *service) OpenAINativeChatCompletionStreamResult(ctx context.Context, req *openaiChatTypes.Request) <-chan OpenAIChatStreamResult {
	streamCtx := newStreamLogContext(ctx, s.logger, "openai_native_chat_completion_stream_result", "OpenAI native Chat Completions", openAIChatModelFromRequest(req))
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
	streamCtx := newStreamLogContext(ctx, s.logger, "openai_native_responses_stream", "OpenAI native Responses", openAIResponsesModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *openaiResponsesTypes.StreamEvent {
		return s.portalService.NativeOpenAIResponsesStream(ctx, req)
	})
}

// OpenAINativeResponsesStreamResult 处理 OpenAI native Responses 流式请求并返回最小收口结果。
func (s *service) OpenAINativeResponsesStreamResult(ctx context.Context, req *openaiResponsesTypes.Request) <-chan OpenAIResponsesStreamResult {
	streamCtx := newStreamLogContext(ctx, s.logger, "openai_native_responses_stream_result", "OpenAI native Responses", openAIResponsesModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *openaiResponsesTypes.StreamEvent {
		return s.portalService.NativeOpenAIResponsesStream(ctx, req)
	})

	return normalizeOpenAIResponsesStream(streamCtx, rawStream)
}
