package portal

import (
	"context"
	"time"

	"github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

// Service Portal 服务接口
//
// 封装所有与 Portal 相关的业务逻辑
type Service interface {
	// Close 优雅关闭服务
	Close(timeout time.Duration) error

	// Anthropic
	NativeAnthropicMessages(ctx context.Context, req *anthropicTypes.Request, opts ...portal.NativeOption) (*anthropicTypes.Response, error)
	NativeAnthropicMessagesStream(ctx context.Context, req *anthropicTypes.Request, opts ...portal.NativeOption) <-chan *anthropicTypes.StreamEvent

	// Gemini
	NativeGeminiGenerateContent(ctx context.Context, req *geminiTypes.Request, opts ...portal.NativeOption) (*geminiTypes.Response, error)
	NativeGeminiStreamGenerateContent(ctx context.Context, req *geminiTypes.Request, opts ...portal.NativeOption) <-chan *geminiTypes.StreamEvent

	// OpenAI
	NativeOpenAIChatCompletion(ctx context.Context, req *openaiChatTypes.Request, opts ...portal.NativeOption) (*openaiChatTypes.Response, error)
	NativeOpenAIChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request, opts ...portal.NativeOption) <-chan *openaiChatTypes.StreamEvent
	NativeOpenAIResponses(ctx context.Context, req *openaiResponsesTypes.Request, opts ...portal.NativeOption) (*openaiResponsesTypes.Response, error)
	NativeOpenAIResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request, opts ...portal.NativeOption) <-chan *openaiResponsesTypes.StreamEvent
}
