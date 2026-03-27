package gateway

import (
	"net/http"

	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
)

// AnthropicStreamResult 定义 Anthropic 流式事件的最小收口结果。
//
// 网关负责将上游事件标准化为统一字段，handler 仅需负责 SSE/HTTP 协议写回。
type AnthropicStreamResult struct {
	Event        *anthropicTypes.StreamEvent
	EventType    anthropicTypes.StreamEventType
	ErrorMessage string
	Done         bool
}

// OpenAIChatStreamResult 定义 OpenAI Chat 流式事件的最小收口结果。
//
// 网关负责将上游事件标准化为统一字段，handler 仅需负责 SSE/HTTP 协议写回。
type OpenAIChatStreamResult struct {
	Event        *openaiChatTypes.StreamEvent
	ErrorMessage string
	Done         bool
}

// OpenAIResponsesStreamResult 定义 OpenAI Responses 流式事件的最小收口结果。
//
// 网关负责将上游事件标准化为统一字段，handler 仅需负责 SSE/HTTP 协议写回。
type OpenAIResponsesStreamResult struct {
	Event        *openaiResponsesTypes.StreamEvent
	ErrorMessage string
	Done         bool
}

// GeminiStreamResult 定义 Gemini 流式事件的最小收口结果。
//
// 网关负责将上游事件标准化为统一字段，handler 仅需负责 SSE/HTTP 协议写回。
type GeminiStreamResult struct {
	Event        *geminiTypes.StreamEvent
	ErrorMessage string
	Done         bool
}

// DataPlaneError 定义数据面错误映射的最小收口结果。
//
// 第一轮仅收口状态码与对外错误信息，避免在 handler 侧重复分支。
type DataPlaneError struct {
	StatusCode int
	Message    string
}

func defaultDataPlaneError(action string) DataPlaneError {
	return DataPlaneError{StatusCode: http.StatusInternalServerError, Message: action}
}
