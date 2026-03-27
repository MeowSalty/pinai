package native

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	"github.com/MeowSalty/pinai/internal/handler/data/common"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	"github.com/gin-gonic/gin"
)

// AnthropicMessages 处理原生 Anthropic 消息请求，路径为 POST /multi/native/v1/messages。
// 解析请求体，处理 User-Agent 头部，根据 stream 参数决定返回流式或非流式响应。
// 成功时返回 200 和响应数据，失败时返回 400 或 500 错误。
//
//	@Summary      发送 Anthropic 消息
//	@Description  处理原生 Anthropic API 的 messages 请求，支持流式和非流式两种模式
//	@Tags         native-anthropic
//	@Accept       json
//	@Produce      json
//	@Param        request  body      anthropicTypes.Request  true  "请求体"
//	@Success      200      {object}  anthropicTypes.Response  "成功"
//	@Failure      400      {object}  anthropicTypes.ErrorResponse  "无效的请求体"
//	@Failure      500      {object}  anthropicTypes.ErrorResponse  "请求失败"
//	@Router       /multi/native/v1/messages [post]
//	@Security     ApiKeyAuth
func (h *Handler) AnthropicMessages(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "anthropic", "api_style", "native")
	var req anthropicTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("请求参数绑定失败", "error", err)
		c.JSON(http.StatusBadRequest, common.NewAnthropicErrorResponse(fmt.Sprintf("无效的请求体: %v", err), http.StatusBadRequest, err))
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		h.streamAnthropic(c, &req)
		return
	}

	resp, err := h.gatewayService.AnthropicNativeMessages(c.Request.Context(), &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "请求失败")
		c.JSON(mappedErr.StatusCode, common.NewAnthropicErrorResponse(mappedErr.Message, mappedErr.StatusCode, err, &mappedErr))
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) streamAnthropic(c *gin.Context, req *anthropicTypes.Request) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	resultChan := h.gatewayService.AnthropicNativeMessagesStreamResult(ctx, req)

	if h.collector != nil {
		defer h.collector.DecrementConnection()
	}

	flusher, _ := c.Writer.(http.Flusher)
	streamStarted := false

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "anthropic", "api_style", "native", "flow", "stream")
	defer func() {
		if r := recover(); r != nil {
			cancel()
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines)
			panicErr := fmt.Errorf("panic: %v", r)
			if !streamStarted {
				c.JSON(http.StatusInternalServerError, common.NewAnthropicErrorResponse("服务器内部错误", http.StatusInternalServerError, panicErr))
				return
			}

			if err := common.WriteAnthropicSSEError(c.Writer, "服务器内部错误", http.StatusInternalServerError, panicErr); err != nil {
				logger.Error("panic 后发送 Anthropic 标准错误事件失败", "error", err)
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}()

	firstResult, ok := <-resultChan
	if !ok {
		return
	}

	if firstResult.ProtocolError != nil && firstResult.ProtocolError.ShouldProxyAsHTTPError {
		logger.Warn("Anthropic 原生流式建流前收到可代理 HTTP 协议错误",
			"status_code", firstResult.ProtocolError.StatusCode,
			"error_type", firstResult.ProtocolError.ErrorType,
			"error_code", firstResult.ProtocolError.ErrorCode,
		)
		c.JSON(
			firstResult.ProtocolError.StatusCode,
			common.NewAnthropicErrorResponse(firstResult.ProtocolError.Message, firstResult.ProtocolError.StatusCode, nil, firstResult.ProtocolError),
		)
		return
	}

	common.SetBaseSSEHeaders(c)
	streamStarted = true

	writeResult := func(result gateway.AnthropicStreamResult) bool {
		if result.ProtocolError != nil {
			logger.Warn("上游返回 Anthropic 原生流式协议错误事件",
				"event_type", result.EventType,
				"status_code", result.ProtocolError.StatusCode,
				"error_type", result.ProtocolError.ErrorType,
				"error_code", result.ProtocolError.ErrorCode,
			)

			if err := common.WriteAnthropicSSEError(c.Writer, result.ProtocolError.Message, result.ProtocolError.StatusCode, nil, result.ProtocolError); err != nil {
				cancel()
				logger.Error("发送 Anthropic 标准错误事件失败", "error", err)
			}

			if flusher != nil {
				flusher.Flush()
			}

			return true
		}

		if result.Event == nil {
			return false
		}

		data, err := json.Marshal(result.Event)
		if err != nil {
			cancel()
			logger.Error("序列化流事件失败", "error", err)
			return true
		}

		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", result.EventType, data); err != nil {
			cancel()
			logger.Error("写入流事件失败", "error", err)
			return true
		}

		if flusher != nil {
			flusher.Flush()
		}

		return result.Done || result.Terminal
	}

	if writeResult(firstResult) {
		return
	}

	for result := range resultChan {
		if writeResult(result) {
			break
		}
	}
}
