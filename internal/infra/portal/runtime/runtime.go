package runtime

import (
	"context"
	"time"

	portalTypes "github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	adapterTypes "github.com/MeowSalty/portal/request/adapter/types"
)

// Runtime 定义 portal 适配执行所需的最小运行时能力边界。
type Runtime interface {
	ChatCompletion(ctx context.Context, req *adapterTypes.RequestContract) (*adapterTypes.ResponseContract, error)
	ChatCompletionStream(ctx context.Context, req *adapterTypes.RequestContract) <-chan *adapterTypes.StreamEventContract

	NativeAnthropicMessages(ctx context.Context, req *anthropicTypes.Request, opts ...portalTypes.NativeOption) (*anthropicTypes.Response, error)
	NativeAnthropicMessagesStream(ctx context.Context, req *anthropicTypes.Request, opts ...portalTypes.NativeOption) <-chan *anthropicTypes.StreamEvent

	NativeGeminiGenerateContent(ctx context.Context, req *geminiTypes.Request, opts ...portalTypes.NativeOption) (*geminiTypes.Response, error)
	NativeGeminiStreamGenerateContent(ctx context.Context, req *geminiTypes.Request, opts ...portalTypes.NativeOption) <-chan *geminiTypes.StreamEvent

	NativeOpenAIChatCompletion(ctx context.Context, req *openaiChatTypes.Request, opts ...portalTypes.NativeOption) (*openaiChatTypes.Response, error)
	NativeOpenAIChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request, opts ...portalTypes.NativeOption) <-chan *openaiChatTypes.StreamEvent
	NativeOpenAIResponses(ctx context.Context, req *openaiResponsesTypes.Request, opts ...portalTypes.NativeOption) (*openaiResponsesTypes.Response, error)
	NativeOpenAIResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request, opts ...portalTypes.NativeOption) <-chan *openaiResponsesTypes.StreamEvent

	Close(timeout time.Duration) error
}
