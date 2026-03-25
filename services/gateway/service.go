package gateway

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/services/portal"
	portalLib "github.com/MeowSalty/portal"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
)

// Service 定义数据面网关应用服务接口。
//
// 当前仅提供第一批最小落地链路：OpenAI compat Chat Completions。
type Service interface {
	// OpenAICompatChatCompletion 处理 OpenAI compat Chat Completions 非流式请求。
	OpenAICompatChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error)

	// OpenAICompatChatCompletionStream 处理 OpenAI compat Chat Completions 流式请求。
	OpenAICompatChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent
}

type service struct {
	portalService portal.Service
	logger        *slog.Logger
}

// New 创建网关应用服务。
func New(portalService portal.Service, logger *slog.Logger) Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &service{
		portalService: portalService,
		logger:        logger,
	}
}

// OpenAICompatChatCompletion 处理 OpenAI compat Chat Completions 非流式请求。
func (s *service) OpenAICompatChatCompletion(ctx context.Context, req *openaiChatTypes.Request) (*openaiChatTypes.Response, error) {
	logger := s.logger.WithGroup("openai_compat_chat_completion")
	logger.Info("开始执行 OpenAI compat Chat Completions 非流式请求", "model", req.Model)

	resp, err := s.portalService.NativeOpenAIChatCompletion(ctx, req, portalLib.WithCompatMode())
	if err != nil {
		logger.Error("OpenAI compat Chat Completions 非流式请求失败", "error", err, "model", req.Model)
		return nil, fmt.Errorf("处理 OpenAI compat Chat Completions 请求失败：%w", err)
	}

	logger.Info("OpenAI compat Chat Completions 非流式请求成功", "model", req.Model)
	return resp, nil
}

// OpenAICompatChatCompletionStream 处理 OpenAI compat Chat Completions 流式请求。
func (s *service) OpenAICompatChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request) <-chan *openaiChatTypes.StreamEvent {
	logger := s.logger.WithGroup("openai_compat_chat_completion_stream")
	logger.Info("开始执行 OpenAI compat Chat Completions 流式请求", "model", req.Model)

	stream := s.portalService.NativeOpenAIChatCompletionStream(ctx, req, portalLib.WithCompatMode())
	logger.Info("OpenAI compat Chat Completions 流式请求已启动", "model", req.Model)
	return stream
}
