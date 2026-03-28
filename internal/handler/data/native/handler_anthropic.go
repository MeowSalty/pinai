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
// 解析请求体，处理 User-Agent 头部，根据 stream 参数决定返回 JSON 或 SSE。
// 当 stream=false 时，错误统一以 HTTP JSON 返回；当 stream=true 时，建流前协议错误以 HTTP JSON 返回，建流后错误以 SSE error 事件返回。
//
//	@Summary      发送 Anthropic 消息
//	@Description  处理原生 Anthropic API 的 messages 请求：非流式返回 JSON；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE error 事件返回
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
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "anthropic", "api_style", "native", "request_name", "messages", "protocol_mode", "auto")
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

	if h.collector != nil {
		h.collector.IncrementConnection()
		defer h.collector.DecrementConnection()
	}

	resp, err := h.gatewayService.AnthropicNativeMessages(c.Request.Context(), &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "请求失败")
		logger.Warn("Anthropic 原生 Messages 请求失败，返回 HTTP JSON 错误",
			"status_code", mappedErr.StatusCode,
			"error_type", mappedErr.ErrorType,
			"error_code", mappedErr.ErrorCode,
		)
		c.JSON(mappedErr.StatusCode, common.NewAnthropicErrorResponse(mappedErr.Message, mappedErr.StatusCode, err, &mappedErr))
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) streamAnthropic(c *gin.Context, req *anthropicTypes.Request) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	resultChan := h.gatewayService.AnthropicNativeMessagesStreamResult(ctx, req)

	connectionReleased := false
	releaseConnection := func() {
		if h.collector == nil || connectionReleased {
			return
		}
		h.collector.DecrementConnection()
		connectionReleased = true
	}
	if h.collector != nil {
		h.collector.IncrementConnection()
	}
	defer releaseConnection()

	flusher, _ := c.Writer.(http.Flusher)
	streamWriterBroken := false
	streamStarted := false

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "anthropic", "api_style", "native", "request_name", "messages", "protocol_mode", "sse", "flow", "stream")
	defer func() {
		if r := recover(); r != nil {
			cancel()
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines, "stream_phase", "panic")
			panicErr := fmt.Errorf("panic: %v", r)
			if !streamStarted {
				c.JSON(http.StatusInternalServerError, common.NewAnthropicErrorResponse("服务器内部错误", http.StatusInternalServerError, panicErr))
				return
			}
			if streamWriterBroken {
				logger.Error("panic 后连接已不可恢复，跳过补写 Anthropic 标准错误事件")
				return
			}

			if err := common.WriteAnthropicSSEError(c.Writer, "服务器内部错误", http.StatusInternalServerError, panicErr); err != nil {
				if common.IsAnthropicStreamWriteError(err) {
					streamWriterBroken = true
					logger.Error("panic 后发送 Anthropic 标准错误事件失败，连接已不可恢复", "error", err)
					return
				}
				logger.Error("panic 后发送 Anthropic 标准错误事件失败", "error", err)
				return
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
			"stream_phase", "pre_stream",
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
			cancel()
			logger.Warn("上游返回 Anthropic 原生流式协议错误事件",
				"event_type", result.EventType,
				"status_code", result.ProtocolError.StatusCode,
				"error_type", result.ProtocolError.ErrorType,
				"error_code", result.ProtocolError.ErrorCode,
				"stream_phase", "streaming",
			)

			if err := common.WriteAnthropicSSEError(c.Writer, result.ProtocolError.Message, result.ProtocolError.StatusCode, nil, result.ProtocolError); err != nil {
				if common.IsAnthropicStreamWriteError(err) {
					streamWriterBroken = true
					logger.Error("发送 Anthropic 标准错误事件失败，连接已不可恢复", "error", err)
				} else {
					logger.Error("发送 Anthropic 标准错误事件失败", "error", err)
				}
				return true
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
			logger.Error("序列化流事件失败", "error", err, "stream_phase", "streaming")

			if streamWriterBroken {
				logger.Error("连接已不可恢复，跳过补写 Anthropic 标准错误事件")
				return true
			}

			if writeErr := common.WriteAnthropicSSEError(c.Writer, "序列化流事件失败", http.StatusInternalServerError, err); writeErr != nil {
				if common.IsAnthropicStreamWriteError(writeErr) {
					streamWriterBroken = true
					logger.Error("序列化失败后补写 Anthropic 标准错误事件失败，连接已不可恢复", "error", writeErr)
				} else {
					logger.Error("序列化失败后补写 Anthropic 标准错误事件失败", "error", writeErr)
				}
				return true
			}

			if flusher != nil {
				flusher.Flush()
			}
			return true
		}

		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", result.EventType, data); err != nil {
			streamWriterBroken = true
			cancel()
			logger.Error("写入流事件失败，连接已不可恢复", "error", err, "stream_phase", "streaming")
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
