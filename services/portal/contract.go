package portal

import (
	"context"
	"fmt"
	"time"

	adapterTypes "github.com/MeowSalty/portal/request/adapter/types"
)

// ChatCompletion 处理聊天完成请求
//
// 提供统一的聊天完成处理入口，包含日志记录和错误处理
func (s *service) ChatCompletion(ctx context.Context, req *adapterTypes.RequestContract) (*adapterTypes.ResponseContract, error) {
	requestLogger := s.logger.WithGroup("chat_completion")
	requestLogger.Info("开始处理聊天完成请求", "model", req.Model)

	originalModel := req.Model

	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		requestLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	startTime := time.Now()

	resp, err := s.portal.ChatCompletion(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		requestLogger.Error("聊天完成处理失败",
			"error", err,
			"duration", duration,
			"model", req.Model,
			"original_model", originalModel)
		return nil, fmt.Errorf("聊天完成处理失败：%w", err)
	}

	requestLogger.Info("聊天完成请求处理成功",
		"duration", duration,
		"model", req.Model,
		"original_model", originalModel,
		"response_id", resp.ID,
		"usage", resp.Usage)

	return resp, nil
}

// ChatCompletionStream 处理流式聊天完成请求
func (s *service) ChatCompletionStream(ctx context.Context, req *adapterTypes.RequestContract) (<-chan *adapterTypes.StreamEventContract, error) {
	streamLogger := s.logger.WithGroup("chat_completion_stream")
	streamLogger.Info("开始处理流式聊天完成请求", "model", req.Model)

	originalModel := req.Model

	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		streamLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	streamLogger.Debug("正在启动流式处理")
	stream := s.portal.ChatCompletionStream(ctx, req)

	streamLogger.Info("聊天完成流启动成功", "model", req.Model)
	return stream, nil
}
