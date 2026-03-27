package gateway

import (
	"context"
	"fmt"
	"time"

	portalLib "github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
)

type anthropicMessagesInvoker func(context.Context, *anthropicTypes.Request) (*anthropicTypes.Response, error)

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
