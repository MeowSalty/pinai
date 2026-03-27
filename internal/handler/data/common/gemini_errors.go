package common

import (
	"net/http"
	"strings"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	"github.com/gin-gonic/gin"
)

const (
	// GeminiErrorStatusInvalidArgument 表示请求参数或格式错误。
	GeminiErrorStatusInvalidArgument = "INVALID_ARGUMENT"
	// GeminiErrorStatusPermissionDenied 表示鉴权失败或权限不足。
	GeminiErrorStatusPermissionDenied = "PERMISSION_DENIED"
	// GeminiErrorStatusNotFound 表示资源不存在。
	GeminiErrorStatusNotFound = "NOT_FOUND"
	// GeminiErrorStatusResourceExhausted 表示触发限流或配额耗尽。
	GeminiErrorStatusResourceExhausted = "RESOURCE_EXHAUSTED"
	// GeminiErrorStatusInternal 表示内部错误。
	GeminiErrorStatusInternal = "INTERNAL"
	// GeminiErrorStatusUnavailable 表示服务暂不可用。
	GeminiErrorStatusUnavailable = "UNAVAILABLE"
	// GeminiErrorStatusDeadlineExceeded 表示超时。
	GeminiErrorStatusDeadlineExceeded = "DEADLINE_EXCEEDED"
)

// DetectGeminiErrorStatus 根据 HTTP 状态码与错误内容推断 Gemini error.status。
func DetectGeminiErrorStatus(status int, err error) string {
	switch status {
	case http.StatusBadRequest:
		return GeminiErrorStatusInvalidArgument
	case http.StatusUnauthorized, http.StatusForbidden:
		return GeminiErrorStatusPermissionDenied
	case http.StatusNotFound:
		return GeminiErrorStatusNotFound
	case http.StatusTooManyRequests:
		return GeminiErrorStatusResourceExhausted
	case http.StatusServiceUnavailable:
		return GeminiErrorStatusUnavailable
	case http.StatusGatewayTimeout:
		return GeminiErrorStatusDeadlineExceeded
	case http.StatusInternalServerError:
		return GeminiErrorStatusInternal
	}

	if err == nil {
		return GeminiErrorStatusInternal
	}

	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") || strings.Contains(msg, "quota") {
		return GeminiErrorStatusResourceExhausted
	}
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "permission") || strings.Contains(msg, "authentication") {
		return GeminiErrorStatusPermissionDenied
	}
	if strings.Contains(msg, "404") || strings.Contains(msg, "not found") {
		return GeminiErrorStatusNotFound
	}
	if strings.Contains(msg, "400") || strings.Contains(msg, "bad request") || strings.Contains(msg, "invalid") {
		return GeminiErrorStatusInvalidArgument
	}
	if strings.Contains(msg, "503") || strings.Contains(msg, "unavailable") {
		return GeminiErrorStatusUnavailable
	}
	if strings.Contains(msg, "504") || strings.Contains(msg, "deadline") || strings.Contains(msg, "timeout") {
		return GeminiErrorStatusDeadlineExceeded
	}

	return GeminiErrorStatusInternal
}

// NewGeminiErrorResponse 构造 Gemini 标准错误响应体。
func NewGeminiErrorResponse(message string, status int, err error, protocolErr ...*gateway.DataPlaneError) geminiTypes.ErrorResponse {
	resolvedStatus := status
	resolvedPublicMessage := strings.TrimSpace(message)
	resolvedInternalDetail := ""
	resolvedErrorStatus := ""

	if mapped := firstGeminiDataPlaneError(protocolErr...); mapped != nil {
		if mapped.StatusCode >= 100 && mapped.StatusCode <= 599 {
			resolvedStatus = mapped.StatusCode
		}
		if text := strings.TrimSpace(mapped.Message); text != "" {
			if resolvedPublicMessage == "" {
				resolvedPublicMessage = text
			} else {
				resolvedInternalDetail = text
			}
		}
		if text := strings.TrimSpace(mapped.ErrorCode); text != "" {
			resolvedErrorStatus = text
		} else if text := strings.TrimSpace(mapped.ErrorType); text != "" {
			resolvedErrorStatus = text
		}
	}

	resolvedMessage := composePublicErrorMessage(resolvedPublicMessage, err, resolvedInternalDetail)

	if resolvedErrorStatus == "" {
		resolvedErrorStatus = DetectGeminiErrorStatus(resolvedStatus, err)
	}

	return geminiTypes.ErrorResponse{
		Error: geminiTypes.ErrorDetail{
			Code:    resolvedStatus,
			Message: resolvedMessage,
			Status:  resolvedErrorStatus,
		},
	}
}

// WriteGeminiJSONError 输出 Gemini 标准错误响应。
func WriteGeminiJSONError(c *gin.Context, status int, message string, err error, protocolErr ...*gateway.DataPlaneError) {
	resp := NewGeminiErrorResponse(message, status, err, protocolErr...)
	c.JSON(resp.Error.Code, resp)
}

func firstGeminiDataPlaneError(items ...*gateway.DataPlaneError) *gateway.DataPlaneError {
	return firstDataPlaneError(items...)
}
