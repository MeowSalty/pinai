package portal

import (
	"context"
	"fmt"
	"time"

	portalTypes "github.com/MeowSalty/portal"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

// NativeOpenAIChatCompletion 处理 OpenAI 原生 Chat Completion 请求
func (s *service) NativeOpenAIChatCompletion(ctx context.Context, req *openaiChatTypes.Request, opts ...portalTypes.NativeOption) (*openaiChatTypes.Response, error) {
	requestLogger := s.logger.WithGroup("raw_openai_chat_completion")
	requestLogger.Info("开始处理 OpenAI Chat 原生请求", "model", req.Model)

	originalModel := req.Model
	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		requestLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	startTime := time.Now()
	resp, err := s.portal.NativeOpenAIChatCompletion(ctx, req, opts...)
	duration := time.Since(startTime)

	if err != nil {
		requestLogger.Error("OpenAI Chat 原生请求处理失败",
			"error", err,
			"duration", duration,
			"model", req.Model,
			"original_model", originalModel)
		return nil, fmt.Errorf("OpenAI Chat 原生请求处理失败：%w", err)
	}

	requestLogger.Info("OpenAI Chat 原生请求处理成功",
		"duration", duration,
		"model", req.Model,
		"original_model", originalModel)

	return resp, nil
}

// NativeOpenAIChatCompletionStream 处理 OpenAI 原生 Chat Completion 流式请求
func (s *service) NativeOpenAIChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request, opts ...portalTypes.NativeOption) <-chan *openaiChatTypes.StreamEvent {
	streamLogger := s.logger.WithGroup("raw_openai_chat_completion_stream")
	streamLogger.Info("开始处理 OpenAI Chat 原生流式请求", "model", req.Model)

	originalModel := req.Model
	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		streamLogger.Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
		req.Model = mappedModel
	}

	stream := s.portal.NativeOpenAIChatCompletionStream(ctx, req, opts...)
	streamLogger.Info("OpenAI Chat 原生流启动成功", "model", req.Model, "original_model", originalModel)
	return stream
}

// NativeOpenAIResponses 处理 OpenAI 原生 Responses 请求
func (s *service) NativeOpenAIResponses(ctx context.Context, req *openaiResponsesTypes.Request, opts ...portalTypes.NativeOption) (*openaiResponsesTypes.Response, error) {
	requestLogger := s.logger.WithGroup("raw_openai_responses")

	modelName := ""
	if req.Model != nil {
		modelName = *req.Model
	}
	requestLogger.Info("开始处理 OpenAI Responses 原生请求", "model", modelName)

	originalModel := modelName
	if modelName != "" {
		if mappedModel, exists := s.modelMappingRule[modelName]; exists {
			requestLogger.Debug("应用模型映射规则",
				"original_model", originalModel,
				"mapped_model", mappedModel)
			modelName = mappedModel
			req.Model = &modelName
		}
	}

	startTime := time.Now()
	resp, err := s.portal.NativeOpenAIResponses(ctx, req, opts...)
	duration := time.Since(startTime)

	if err != nil {
		requestLogger.Error("OpenAI Responses 原生请求处理失败",
			"error", err,
			"duration", duration,
			"model", modelName,
			"original_model", originalModel)
		return nil, fmt.Errorf("OpenAI Responses 原生请求处理失败：%w", err)
	}

	requestLogger.Info("OpenAI Responses 原生请求处理成功",
		"duration", duration,
		"model", modelName,
		"original_model", originalModel)

	return resp, nil
}

// NativeOpenAIResponsesStream 处理 OpenAI 原生 Responses 流式请求
func (s *service) NativeOpenAIResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request, opts ...portalTypes.NativeOption) <-chan *openaiResponsesTypes.StreamEvent {
	streamLogger := s.logger.WithGroup("raw_openai_responses_stream")

	modelName := ""
	if req.Model != nil {
		modelName = *req.Model
	}
	streamLogger.Info("开始处理 OpenAI Responses 原生流式请求", "model", modelName)

	originalModel := modelName
	if modelName != "" {
		if mappedModel, exists := s.modelMappingRule[modelName]; exists {
			streamLogger.Debug("应用模型映射规则",
				"original_model", originalModel,
				"mapped_model", mappedModel)
			modelName = mappedModel
			req.Model = &modelName
		}
	}

	stream := s.portal.NativeOpenAIResponsesStream(ctx, req, opts...)
	streamLogger.Info("OpenAI Responses 原生流启动成功", "model", modelName, "original_model", originalModel)
	return stream
}
