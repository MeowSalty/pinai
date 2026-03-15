package native

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/handlers/multi/common"
	"github.com/MeowSalty/pinai/services/stats"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	openaiSharedTypes "github.com/MeowSalty/portal/request/adapter/openai/types/shared"
	"github.com/gin-gonic/gin"
)

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}

// OpenAIChatCompletions 处理原生 OpenAI 聊天补全请求，路径为 POST /multi/native/v1/chat/completions。
// 解析请求体，处理 User-Agent 头部，根据 stream 参数决定返回流式或非流式响应。
// 成功时返回 200 和响应数据，失败时返回 400 或 500 错误。
//
//	@Summary      OpenAI 聊天补全
//	@Description  处理原生 OpenAI API 的 chat completions 请求，支持流式和非流式两种模式
//	@Tags         native-openai
//	@Accept       json
//	@Produce      json
//	@Param        request  body      openaiChatTypes.Request  true  "请求体"
//	@Success      200      {object}  openaiChatTypes.Response  "成功"
//	@Failure      400      {object}  openaiSharedTypes.HTTPError    "无效的请求体"
//	@Failure      500      {object}  openaiSharedTypes.HTTPError    "请求失败"
//	@Router       /multi/native/v1/chat/completions [post]
//	@Security     ApiKeyAuth
func (h *Handler) OpenAIChatCompletions(c *gin.Context) {
	var req openaiChatTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, openaiSharedTypes.HTTPError{
			Error: openaiSharedTypes.ErrorDetail{
				Message: fmt.Sprintf("无效的请求体: %v", err),
				Type:    "invalid_request_error",
			},
		})
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		h.streamOpenAIChat(c, &req, true)
		return
	}

	resp, err := h.portalService.NativeOpenAIChatCompletion(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, openaiSharedTypes.HTTPError{
			Error: openaiSharedTypes.ErrorDetail{
				Message: fmt.Sprintf("请求失败: %v", err),
				Type:    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// OpenAIResponses 处理原生 OpenAI 响应请求，路径为 POST /multi/native/v1/responses。
// 解析请求体，处理 User-Agent 头部，根据 stream 参数决定返回流式或非流式响应。
// 成功时返回 200 和响应数据，失败时返回 400 或 500 错误。
//
//	@Summary      OpenAI 响应
//	@Description  处理原生 OpenAI API 的 responses 请求，支持流式和非流式两种模式
//	@Tags         native-openai
//	@Accept       json
//	@Produce      json
//	@Param        request  body      openaiResponsesTypes.Request  true  "请求体"
//	@Success      200      {object}  openaiResponsesTypes.Response  "成功"
//	@Failure      400      {object}  openaiSharedTypes.HTTPError         "无效的请求体"
//	@Failure      500      {object}  openaiSharedTypes.HTTPError         "请求失败"
//	@Router       /multi/native/v1/responses [post]
//	@Security     ApiKeyAuth
func (h *Handler) OpenAIResponses(c *gin.Context) {
	var req openaiResponsesTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, openaiSharedTypes.HTTPError{
			Error: openaiSharedTypes.ErrorDetail{
				Message: fmt.Sprintf("无效的请求体: %v", err),
				Type:    "invalid_request_error",
			},
		})
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		h.streamOpenAIResponses(c, &req, true)
		return
	}

	resp, err := h.portalService.NativeOpenAIResponses(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, openaiSharedTypes.HTTPError{
			Error: openaiSharedTypes.ErrorDetail{
				Message: fmt.Sprintf("请求失败: %v", err),
				Type:    "internal_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) streamOpenAIChat(c *gin.Context, req *openaiChatTypes.Request, sendDone bool) {
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	eventChan := h.portalService.NativeOpenAIChatCompletionStream(ctx, req)

	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines)
			h.sendOpenAIStreamError(c.Writer, "internal_error", fmt.Sprintf("异常: %v", r), "internal_error")
		}
	}()

	for event := range eventChan {
		data, err := json.Marshal(event)
		if err != nil {
			cancel()
			logger.Error("序列化流事件失败", "error", err)
			h.sendOpenAIStreamError(c.Writer, "internal_error", fmt.Sprintf("序列化流事件失败: %v", err), "internal_error")
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			cancel()
			logger.Error("写入流事件失败", "error", err)
			h.sendOpenAIStreamError(c.Writer, "internal_error", fmt.Sprintf("写入流事件失败: %v", err), "internal_error")
			break
		}

		flusher.Flush()
	}

	if sendDone {
		if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
			cancel()
			logger.Error("写入流结束标识失败", "error", err)
		}
		flusher.Flush()
	}
}

// sendOpenAIResponsesStreamError 发送 OpenAI Responses 流式错误事件。
func (h *Handler) sendOpenAIResponsesStreamError(w io.Writer, code, message, param string) {
	codePtr := &code
	paramPtr := &param

	errEvent := openaiResponsesTypes.ResponseErrorEvent{
		Type:           openaiResponsesTypes.StreamEventError,
		Code:           codePtr,
		Message:        message,
		Param:          paramPtr,
		SequenceNumber: 0,
	}

	data, _ := json.Marshal(errEvent)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// sendOpenAIStreamError 发送流式错误响应
func (h *Handler) sendOpenAIStreamError(w io.Writer, errorType, message, code string) {
	errResp := openaiSharedTypes.HTTPError{
		Error: openaiSharedTypes.ErrorDetail{
			Message: message,
			Type:    errorType,
			Param:   nil,
			Code:    stringPtr(code),
		},
	}
	data, _ := json.Marshal(errResp)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func (h *Handler) streamOpenAIResponses(c *gin.Context, req *openaiResponsesTypes.Request, sendDone bool) {
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	eventChan := h.portalService.NativeOpenAIResponsesStream(ctx, req)

	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines)
			h.sendOpenAIResponsesStreamError(c.Writer, "internal_error", fmt.Sprintf("异常: %v", r), "internal_error")
		}
	}()

	for event := range eventChan {
		data, err := json.Marshal(event)
		if err != nil {
			cancel()
			logger.Error("序列化流事件失败", "error", err)
			h.sendOpenAIResponsesStreamError(c.Writer, "internal_error", fmt.Sprintf("序列化流事件失败: %v", err), "internal_error")
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			cancel()
			logger.Error("写入流事件失败", "error", err)
			h.sendOpenAIResponsesStreamError(c.Writer, "internal_error", fmt.Sprintf("写入流事件失败: %v", err), "internal_error")
			break
		}

		logger.Debug("写入流事件成功")
		flusher.Flush()
	}

	if sendDone {
		if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
			cancel()
			logger.Error("写入流结束标识失败", "error", err)
		}
		flusher.Flush()
	}
}
