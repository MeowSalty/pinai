package portal

import (
	"context"

	portalTypes "github.com/MeowSalty/portal"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

// NativeOpenAIChatCompletion 处理 OpenAI 原生 Chat Completion 请求
func (s *service) NativeOpenAIChatCompletion(ctx context.Context, req *openaiChatTypes.Request, opts ...portalTypes.NativeOption) (*openaiChatTypes.Response, error) {
	originalModel := req.Model
	if mappedModel, exists := s.modelMappingRule[req.Model]; exists {
		req.Model = mappedModel
		s.logger.WithGroup("raw_openai_chat_completion").Debug("应用模型映射规则",
			"original_model", originalModel,
			"mapped_model", mappedModel)
	}

	return s.portal.NativeOpenAIChatCompletion(ctx, req, opts...)
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
	modelName := ""
	if req.Model != nil {
		modelName = *req.Model
	}

	originalModel := modelName
	if modelName != "" {
		if mappedModel, exists := s.modelMappingRule[modelName]; exists {
			s.logger.WithGroup("raw_openai_responses").Debug("应用模型映射规则",
				"original_model", originalModel,
				"mapped_model", mappedModel)
			modelName = mappedModel
			req.Model = &modelName
		}
	}

	return s.portal.NativeOpenAIResponses(ctx, req, opts...)
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
