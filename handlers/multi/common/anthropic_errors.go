package common

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
)

const (
	// AnthropicErrorTypeInvalidRequest 表示请求参数或格式错误。
	AnthropicErrorTypeInvalidRequest = "invalid_request_error"
	// AnthropicErrorTypeAuthentication 表示鉴权失败。
	AnthropicErrorTypeAuthentication = "authentication_error"
	// AnthropicErrorTypeNotFound 表示资源不存在。
	AnthropicErrorTypeNotFound = "not_found_error"
	// AnthropicErrorTypeRateLimit 表示触发限流。
	AnthropicErrorTypeRateLimit = "rate_limit_error"
	// AnthropicErrorTypeAPI 表示通用服务错误。
	AnthropicErrorTypeAPI = "api_error"
)

// DetectAnthropicErrorType 根据状态码与错误内容推断 Anthropic 错误类型。
func DetectAnthropicErrorType(status int, err error) string {
	switch status {
	case 400:
		return AnthropicErrorTypeInvalidRequest
	case 401, 403:
		return AnthropicErrorTypeAuthentication
	case 404:
		return AnthropicErrorTypeNotFound
	case 429:
		return AnthropicErrorTypeRateLimit
	}

	if err == nil {
		return AnthropicErrorTypeAPI
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
		return AnthropicErrorTypeRateLimit
	}
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "authentication") {
		return AnthropicErrorTypeAuthentication
	}
	if strings.Contains(msg, "404") || strings.Contains(msg, "not found") {
		return AnthropicErrorTypeNotFound
	}
	if strings.Contains(msg, "400") || strings.Contains(msg, "bad request") || strings.Contains(msg, "invalid request") {
		return AnthropicErrorTypeInvalidRequest
	}

	return AnthropicErrorTypeAPI
}

// NewAnthropicErrorResponse 构造 Anthropic 非流式错误响应体。
func NewAnthropicErrorResponse(message string, status int, err error) anthropicTypes.ErrorResponse {
	return anthropicTypes.ErrorResponse{
		Type: "error",
		Error: anthropicTypes.Error{
			Type:    DetectAnthropicErrorType(status, err),
			Message: message,
		},
	}
}

// WriteAnthropicSSEError 写入 Anthropic 流式错误事件。
func WriteAnthropicSSEError(w io.Writer, message string, status int, err error) error {
	errResp := NewAnthropicErrorResponse(message, status, err)
	data, marshalErr := json.Marshal(errResp)
	if marshalErr != nil {
		return fmt.Errorf("序列化 Anthropic 流式错误失败：%w", marshalErr)
	}

	if _, writeErr := fmt.Fprintf(w, "event: error\ndata: %s\n\n", data); writeErr != nil {
		return fmt.Errorf("写入 Anthropic 流式错误失败：%w", writeErr)
	}

	return nil
}
