package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	portalLib "github.com/MeowSalty/portal"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
)

type geminiGenerateContentInvoker func(context.Context, *geminiTypes.Request) (*geminiTypes.Response, error)

// GeminiNativeGenerateContent 处理 Gemini native generateContent 非流式请求。
func (s *service) GeminiNativeGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error) {
	return s.executeGeminiGenerateContent(ctx, req, "gemini_native_generate_content", "Gemini native generateContent", func(inCtx context.Context, inReq *geminiTypes.Request) (*geminiTypes.Response, error) {
		return s.portalService.NativeGeminiGenerateContent(inCtx, inReq)
	})
}

func (s *service) executeGeminiGenerateContent(ctx context.Context, req *geminiTypes.Request, loggerGroup, requestName string, invoker geminiGenerateContentInvoker) (*geminiTypes.Response, error) {
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

	for _, candidate := range event.Candidates {
		if candidate.FinishReason != "" && candidate.FinishReason != geminiTypes.FinishReasonUnspecified {
			return true
		}
	}

	if event.PromptFeedback != nil && event.PromptFeedback.BlockReason != "" && event.PromptFeedback.BlockReason != geminiTypes.BlockReasonUnspecified {
		return true
	}

	return false
}

func geminiFinishReasonIsProtocolError(reason geminiTypes.FinishReason) bool {
	switch reason {
	case geminiTypes.FinishReasonSafety,
		geminiTypes.FinishReasonRecitation,
		geminiTypes.FinishReasonBlocklist,
		geminiTypes.FinishReasonProhibitedContent,
		geminiTypes.FinishReasonSPII,
		geminiTypes.FinishReasonMalformedFunction,
		geminiTypes.FinishReasonImageSafety,
		geminiTypes.FinishReasonImageProhibited,
		geminiTypes.FinishReasonUnexpectedToolCall,
		geminiTypes.FinishReasonMissingThoughtSig:
		return true
	default:
		return false
	}
}

func geminiFinishReasonMessage(reason geminiTypes.FinishReason, finishMessage string) string {
	trimmed := strings.TrimSpace(finishMessage)
	if trimmed != "" {
		return trimmed
	}

	return fmt.Sprintf("Gemini 流式生成终止，原因：%s", reason)
}

func geminiProtocolErrorFromPromptFeedback(event *geminiTypes.StreamEvent) (*DataPlaneError, bool) {
	if event == nil {
		return nil, false
	}

	if event.PromptFeedback == nil {
		return nil, false
	}

	reason := event.PromptFeedback.BlockReason
	if reason == "" || reason == geminiTypes.BlockReasonUnspecified {
		return nil, false
	}

	message := fmt.Sprintf("提示内容被拦截：%s", reason)
	mapped := DataPlaneError{
		StatusCode:             http.StatusBadRequest,
		Message:                message,
		Provider:               "gemini",
		ErrorType:              "prompt_blocked",
		ErrorCode:              strings.ToLower(string(reason)),
		Raw:                    event.PromptFeedback,
		Retryable:              isRetryableByStatus(http.StatusBadRequest),
		ShouldProxyAsHTTPError: true,
	}

	return &mapped, true
}

func geminiProtocolErrorFromCandidates(event *geminiTypes.StreamEvent) (*DataPlaneError, bool) {
	if event == nil {
		return nil, false
	}

	for _, candidate := range event.Candidates {
		reason := candidate.FinishReason
		if !geminiFinishReasonIsProtocolError(reason) {
			continue
		}

		statusCode := http.StatusBadRequest
		if reason == geminiTypes.FinishReasonUnexpectedToolCall || reason == geminiTypes.FinishReasonMissingThoughtSig {
			statusCode = http.StatusBadGateway
		}

		mapped := DataPlaneError{
			StatusCode:             statusCode,
			Message:                geminiFinishReasonMessage(reason, candidate.FinishMessage),
			Provider:               "gemini",
			ErrorType:              "candidate_blocked",
			ErrorCode:              strings.ToLower(string(reason)),
			Raw:                    candidate,
			Retryable:              isRetryableByStatus(statusCode),
			ShouldProxyAsHTTPError: true,
		}

		return &mapped, true
	}

	return nil, false
}

func normalizeGeminiStream(streamCtx streamLogContext, source <-chan *geminiTypes.StreamEvent) <-chan GeminiStreamResult {
	out := make(chan GeminiStreamResult)
	go func() {
		defer close(out)

		streamCtx.logger.Debug("开始消费 Gemini 流式结果", streamCtx.attrs...)
		for event := range source {
			if event == nil {
				streamCtx.logger.Debug("忽略空 Gemini 流式事件", streamCtx.attrs...)
				continue
			}

			result := GeminiStreamResult{
				Event: event,
				Done:  geminiStreamDone(event),
			}

			if protocolError, hasProtocolError := geminiProtocolErrorFromPromptFeedback(event); hasProtocolError {
				result.ProtocolError = protocolError
				result.Terminal = true
				result.Done = true
			}

			if result.ProtocolError == nil {
				if protocolError, hasProtocolError := geminiProtocolErrorFromCandidates(event); hasProtocolError {
					result.ProtocolError = protocolError
					result.Terminal = true
					result.Done = true
				}
			}

			if result.Done && !result.Terminal {
				result.Terminal = true
			}

			if result.ProtocolError != nil {
				result.Done = true
			}

			streamCtx.logger.Debug("Gemini 流式事件已收口",
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

// GeminiNativeGenerateContentStream 处理 Gemini native streamGenerateContent 流式请求。
func (s *service) GeminiNativeGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent {
	streamCtx := newStreamLogContext(ctx, s.logger, "gemini_native_generate_content_stream", "Gemini native streamGenerateContent", geminiModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *geminiTypes.StreamEvent {
		return s.portalService.NativeGeminiStreamGenerateContent(ctx, req)
	})
}

// GeminiNativeGenerateContentStreamResult 处理 Gemini native streamGenerateContent 流式请求并返回最小收口结果。
func (s *service) GeminiNativeGenerateContentStreamResult(ctx context.Context, req *geminiTypes.Request) <-chan GeminiStreamResult {
	streamCtx := newStreamLogContext(ctx, s.logger, "gemini_native_generate_content_stream_result", "Gemini native streamGenerateContent", geminiModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *geminiTypes.StreamEvent {
		return s.portalService.NativeGeminiStreamGenerateContent(ctx, req)
	})

	return normalizeGeminiStream(streamCtx, rawStream)
}

// GeminiCompatGenerateContent 处理 Gemini compat generateContent 非流式请求。
func (s *service) GeminiCompatGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error) {
	return s.executeGeminiGenerateContent(ctx, req, "gemini_compat_generate_content", "Gemini compat generateContent", func(inCtx context.Context, inReq *geminiTypes.Request) (*geminiTypes.Response, error) {
		return s.portalService.NativeGeminiGenerateContent(inCtx, inReq, portalLib.WithCompatMode())
	})
}

// GeminiCompatGenerateContentStream 处理 Gemini compat streamGenerateContent 流式请求。
func (s *service) GeminiCompatGenerateContentStream(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent {
	streamCtx := newStreamLogContext(ctx, s.logger, "gemini_compat_generate_content_stream", "Gemini compat streamGenerateContent", geminiModelFromRequest(req))
	return startStream(streamCtx, func() <-chan *geminiTypes.StreamEvent {
		return s.portalService.NativeGeminiStreamGenerateContent(ctx, req, portalLib.WithCompatMode())
	})
}

// GeminiCompatGenerateContentStreamResult 处理 Gemini compat streamGenerateContent 流式请求并返回最小收口结果。
func (s *service) GeminiCompatGenerateContentStreamResult(ctx context.Context, req *geminiTypes.Request) <-chan GeminiStreamResult {
	streamCtx := newStreamLogContext(ctx, s.logger, "gemini_compat_generate_content_stream_result", "Gemini compat streamGenerateContent", geminiModelFromRequest(req))
	rawStream := startStream(streamCtx, func() <-chan *geminiTypes.StreamEvent {
		return s.portalService.NativeGeminiStreamGenerateContent(ctx, req, portalLib.WithCompatMode())
	})

	return normalizeGeminiStream(streamCtx, rawStream)
}
