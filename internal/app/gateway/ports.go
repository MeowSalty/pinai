package gateway

import (
	"context"
	"time"

	portalLib "github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

// NativeOption 定义 gateway 应用层透传给底层调用的可选参数。
//
// 当前仅用于兼容模式等最小能力收口，避免应用层直接依赖 infra 包类型。
type NativeOption = portalLib.NativeOption

// GatewayLifecyclePort 定义网关依赖资源生命周期能力。
type GatewayLifecyclePort interface {
	// Close 优雅关闭底层资源。
	Close(timeout time.Duration) error
}

// AnthropicMessagesPort 定义 Anthropic Messages 最小调用能力。
type AnthropicMessagesPort interface {
	NativeAnthropicMessages(ctx context.Context, req *anthropicTypes.Request, opts ...NativeOption) (*anthropicTypes.Response, error)
	NativeAnthropicMessagesStream(ctx context.Context, req *anthropicTypes.Request, opts ...NativeOption) <-chan *anthropicTypes.StreamEvent
}

// GeminiGenerateContentPort 定义 Gemini generateContent 最小调用能力。
type GeminiGenerateContentPort interface {
	NativeGeminiGenerateContent(ctx context.Context, req *geminiTypes.Request, opts ...NativeOption) (*geminiTypes.Response, error)
	NativeGeminiStreamGenerateContent(ctx context.Context, req *geminiTypes.Request, opts ...NativeOption) <-chan *geminiTypes.StreamEvent
}

// OpenAIChatPort 定义 OpenAI Chat Completions 最小调用能力。
type OpenAIChatPort interface {
	NativeOpenAIChatCompletion(ctx context.Context, req *openaiChatTypes.Request, opts ...NativeOption) (*openaiChatTypes.Response, error)
	NativeOpenAIChatCompletionStream(ctx context.Context, req *openaiChatTypes.Request, opts ...NativeOption) <-chan *openaiChatTypes.StreamEvent
}

// OpenAIResponsesPort 定义 OpenAI Responses 最小调用能力。
type OpenAIResponsesPort interface {
	NativeOpenAIResponses(ctx context.Context, req *openaiResponsesTypes.Request, opts ...NativeOption) (*openaiResponsesTypes.Response, error)
	NativeOpenAIResponsesStream(ctx context.Context, req *openaiResponsesTypes.Request, opts ...NativeOption) <-chan *openaiResponsesTypes.StreamEvent
}

// GatewayPort 聚合 gateway 应用层当前依赖的最小 ports。
type GatewayPort interface {
	GatewayLifecyclePort
	AnthropicMessagesPort
	GeminiGenerateContentPort
	OpenAIChatPort
	OpenAIResponsesPort
}
