package portal

import (
	"context"
	"fmt"
	"time"

	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
)

// NativeAnthropicMessages 处理 Anthropic 原生 Messages 请求
func (s *service) NativeAnthropicMessages(ctx context.Context, req *anthropicTypes.Request) (*anthropicTypes.Response, error) {
	requestLogger := s.logger.WithGroup("raw_anthropic_messages")
	requestLogger.Info("开始处理 Anthropic 原生请求", "model", req.Model)

	originalModel := req.Model
	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		requestLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	startTime := time.Now()
	resp, err := s.portal.NativeAnthropicMessages(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		requestLogger.Error("Anthropic 原生请求处理失败",
			"error", err,
			"duration", duration,
			"model", req.Model,
			"original_model", originalModel)
		return nil, fmt.Errorf("Anthropic 原生请求处理失败：%w", err)
	}

	requestLogger.Info("Anthropic 原生请求处理成功",
		"duration", duration,
		"model", req.Model,
		"original_model", originalModel)

	return resp, nil
}

// NativeAnthropicMessagesStream 处理 Anthropic 原生流式 Messages 请求
func (s *service) NativeAnthropicMessagesStream(ctx context.Context, req *anthropicTypes.Request) <-chan *anthropicTypes.StreamEvent {
	streamLogger := s.logger.WithGroup("raw_anthropic_messages_stream")
	streamLogger.Info("开始处理 Anthropic 原生流式请求", "model", req.Model)

	originalModel := req.Model
	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		streamLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	stream := s.portal.NativeAnthropicMessagesStream(ctx, req)
	streamLogger.Info("Anthropic 原生流启动成功", "model", req.Model, "original_model", originalModel)
	return stream
}
