package common

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/MeowSalty/pinai/internal/app/gateway"
)

const (
	// OpenAIErrorTypeInvalidRequest 表示请求参数或格式错误。
	OpenAIErrorTypeInvalidRequest = "invalid_request_error"
	// OpenAIErrorTypeAuthentication 表示鉴权失败。
	OpenAIErrorTypeAuthentication = "authentication_error"
	// OpenAIErrorTypeRateLimit 表示触发限流。
	OpenAIErrorTypeRateLimit = "rate_limit_error"
	// OpenAIErrorTypeInternal 表示内部错误。
	OpenAIErrorTypeInternal = "internal_error"
)

// OpenAIErrorDetail 表示 OpenAI 错误详情对象。
type OpenAIErrorDetail struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"`
	Code    *string `json:"code"`
}

// OpenAIHTTPErrorResponse 表示 OpenAI 非流式错误响应体。
type OpenAIHTTPErrorResponse struct {
	Error OpenAIErrorDetail `json:"error"`
}

// OpenAIResponsesSSEErrorResponse 表示 OpenAI Responses 流式错误事件。
type OpenAIResponsesSSEErrorResponse struct {
	Type  string            `json:"type"`
	Error OpenAIErrorDetail `json:"error"`
}

// OpenAIResponsesTypedEventError 表示 OpenAI Responses 更严格的 typed event 错误载荷。
type OpenAIResponsesTypedEventError struct {
	Type  string            `json:"type"`
	Error OpenAIErrorDetail `json:"error"`
}

// DetectOpenAIErrorType 根据状态码与错误内容推断 OpenAI 错误类型。
func DetectOpenAIErrorType(status int, err error) string {
	switch status {
	case 400:
		return OpenAIErrorTypeInvalidRequest
	case 401, 403:
		return OpenAIErrorTypeAuthentication
	case 429:
		return OpenAIErrorTypeRateLimit
	}

	if err == nil {
		return OpenAIErrorTypeInternal
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
		return OpenAIErrorTypeRateLimit
	}
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "authentication") {
		return OpenAIErrorTypeAuthentication
	}
	if strings.Contains(msg, "400") || strings.Contains(msg, "bad request") || strings.Contains(msg, "invalid request") {
		return OpenAIErrorTypeInvalidRequest
	}

	return OpenAIErrorTypeInternal
}

// NewOpenAIHTTPErrorResponse 构造 OpenAI 非流式错误响应体。
func NewOpenAIHTTPErrorResponse(message string, status int, err error, protocolErr ...*gateway.DataPlaneError) OpenAIHTTPErrorResponse {
	resolvedStatus := status
	resolvedMessage := strings.TrimSpace(message)
	resolvedType := ""
	var resolvedParam *string
	var resolvedCode *string

	if mapped := firstOpenAIDataPlaneError(protocolErr...); mapped != nil {
		if mapped.StatusCode >= 100 && mapped.StatusCode <= 599 {
			resolvedStatus = mapped.StatusCode
		}
		if text := strings.TrimSpace(mapped.Message); text != "" {
			resolvedMessage = text
		}
		if text := strings.TrimSpace(mapped.ErrorType); text != "" {
			resolvedType = text
		}
		if text := strings.TrimSpace(mapped.Param); text != "" {
			resolvedParam = stringPtr(text)
		}
		if text := strings.TrimSpace(mapped.ErrorCode); text != "" {
			resolvedCode = stringPtr(text)
		}
	}

	if resolvedMessage == "" && err != nil {
		resolvedMessage = strings.TrimSpace(err.Error())
	}
	if resolvedMessage == "" {
		resolvedMessage = "请求处理失败"
	}

	if resolvedType == "" {
		resolvedType = DetectOpenAIErrorType(resolvedStatus, err)
	}

	if resolvedCode == nil && resolvedType == OpenAIErrorTypeInternal {
		resolvedCode = stringPtr(OpenAIErrorTypeInternal)
	}

	return OpenAIHTTPErrorResponse{
		Error: OpenAIErrorDetail{
			Message: resolvedMessage,
			Type:    resolvedType,
			Param:   resolvedParam,
			Code:    resolvedCode,
		},
	}
}

// WriteOpenAIChatSSEError 写入 OpenAI Chat 流式错误事件。
func WriteOpenAIChatSSEError(w io.Writer, message string, status int, err error, protocolErr ...*gateway.DataPlaneError) error {
	errResp := NewOpenAIHTTPErrorResponse(message, status, err, protocolErr...)
	data, marshalErr := json.Marshal(errResp)
	if marshalErr != nil {
		return fmt.Errorf("序列化 OpenAI Chat 流式错误失败：%w", marshalErr)
	}
	if _, writeErr := fmt.Fprintf(w, "event: error\ndata: %s\n\n", data); writeErr != nil {
		return fmt.Errorf("写入 OpenAI Chat 流式错误失败：%w", writeErr)
	}
	return nil
}

// WriteOpenAIResponsesSSEError 写入 OpenAI Responses 流式错误事件。
func WriteOpenAIResponsesSSEError(w io.Writer, message string, status int, err error, protocolErr ...*gateway.DataPlaneError) error {
	httpErr := NewOpenAIHTTPErrorResponse(message, status, err, protocolErr...)
	errResp := OpenAIResponsesSSEErrorResponse{
		Type:  "error",
		Error: httpErr.Error,
	}
	data, marshalErr := json.Marshal(errResp)
	if marshalErr != nil {
		return fmt.Errorf("序列化 OpenAI Responses 流式错误失败：%w", marshalErr)
	}
	if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", data); writeErr != nil {
		return fmt.Errorf("写入 OpenAI Responses 流式错误失败：%w", writeErr)
	}
	return nil
}

// WriteOpenAIResponsesTypedEventError 写入 OpenAI Responses 更严格 typed event 错误事件。
func WriteOpenAIResponsesTypedEventError(w io.Writer, message string, status int, err error, protocolErr ...*gateway.DataPlaneError) error {
	httpErr := NewOpenAIHTTPErrorResponse(message, status, err, protocolErr...)
	errResp := OpenAIResponsesTypedEventError{
		Type:  "response.error",
		Error: httpErr.Error,
	}
	data, marshalErr := json.Marshal(errResp)
	if marshalErr != nil {
		return fmt.Errorf("序列化 OpenAI Responses typed event 错误失败：%w", marshalErr)
	}
	if _, writeErr := fmt.Fprintf(w, "event: response.error\ndata: %s\n\n", data); writeErr != nil {
		return fmt.Errorf("写入 OpenAI Responses typed event 错误失败：%w", writeErr)
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}

func firstOpenAIDataPlaneError(items ...*gateway.DataPlaneError) *gateway.DataPlaneError {
	for _, item := range items {
		if item != nil {
			return item
		}
	}

	return nil
}
