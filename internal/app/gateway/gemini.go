package gateway

import (
	"context"
	"fmt"
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
