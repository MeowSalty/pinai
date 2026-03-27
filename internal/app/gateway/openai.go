package gateway

import (
	"context"
	"fmt"
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
