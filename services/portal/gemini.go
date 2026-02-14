package portal

import (
	"context"
	"fmt"
	"time"

	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
)

// NativeGeminiGenerateContent 处理 Gemini 原生 GenerateContent 请求
func (s *service) NativeGeminiGenerateContent(ctx context.Context, req *geminiTypes.Request) (*geminiTypes.Response, error) {
	requestLogger := s.logger.WithGroup("raw_gemini_generate_content")
	requestLogger.Info("开始处理 Gemini 原生请求", "model", req.Model)

	originalModel := req.Model
	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		requestLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	startTime := time.Now()
	resp, err := s.portal.NativeGeminiGenerateContent(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		requestLogger.Error("Gemini 原生请求处理失败",
			"error", err,
			"duration", duration,
			"model", req.Model,
			"original_model", originalModel)
		return nil, fmt.Errorf("Gemini 原生请求处理失败：%w", err)
	}

	requestLogger.Info("Gemini 原生请求处理成功",
		"duration", duration,
		"model", req.Model,
		"original_model", originalModel)

	return resp, nil
}

// NativeGeminiStreamGenerateContent 处理 Gemini 原生流式 StreamGenerateContent 请求
func (s *service) NativeGeminiStreamGenerateContent(ctx context.Context, req *geminiTypes.Request) <-chan *geminiTypes.StreamEvent {
	streamLogger := s.logger.WithGroup("raw_gemini_stream_generate_content")
	streamLogger.Info("开始处理 Gemini 原生流式请求", "model", req.Model)

	originalModel := req.Model
	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		streamLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	stream := s.portal.NativeGeminiStreamGenerateContent(ctx, req)
	streamLogger.Info("Gemini 原生流启动成功", "model", req.Model, "original_model", originalModel)
	return stream
}
