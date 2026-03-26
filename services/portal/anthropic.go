package portal

import (
	"context"

	portalTypes "github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
)

// NativeAnthropicMessages 处理 Anthropic 原生 Messages 请求
func (s *service) NativeAnthropicMessages(ctx context.Context, req *anthropicTypes.Request, opts ...portalTypes.NativeOption) (*anthropicTypes.Response, error) {
	requestLogger := s.logger.WithGroup("raw_anthropic_messages")

	originalModel := req.Model
	if mappedModel, exists := s.mapModel(req.Model); exists {
		requestLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	return s.portal.NativeAnthropicMessages(ctx, req, opts...)
}

// NativeAnthropicMessagesStream 处理 Anthropic 原生流式 Messages 请求
func (s *service) NativeAnthropicMessagesStream(ctx context.Context, req *anthropicTypes.Request, opts ...portalTypes.NativeOption) <-chan *anthropicTypes.StreamEvent {
	streamLogger := s.logger.WithGroup("raw_anthropic_messages_stream")
	streamLogger.Info("开始处理 Anthropic 原生流式请求", "model", req.Model)

	originalModel := req.Model
	if mappedModel, exists := s.mapModel(req.Model); exists {
		streamLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	stream := s.portal.NativeAnthropicMessagesStream(ctx, req, opts...)
	streamLogger.Info("Anthropic 原生流启动成功", "model", req.Model, "original_model", originalModel)
	return stream
}
