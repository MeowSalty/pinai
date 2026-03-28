package multi

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

// Messages 处理 Anthropic 消息完成请求，路径为 POST /multi/v1/messages。
// 解析请求体并转换为统一格式，根据 stream 参数决定返回 JSON 或 SSE。
// 当 stream=false 时，错误统一以 HTTP JSON 返回；当 stream=true 时，建流前协议错误以 HTTP JSON 返回，建流后错误以 SSE error 事件返回。
//
// @Summary      消息完成
// @Description  创建消息完成响应：非流式返回 JSON；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE error 事件返回
// @Tags         Anthropic
// @Accept       json
// @Produce      json
// @Param        request  body      anthropicTypes.Request  true  "消息请求"
// @Success      200      {object}  anthropicTypes.MessageResponse
// @Failure      400      {object}  anthropicTypes.ErrorResponse
// @Failure      401      {object}  anthropicTypes.ErrorResponse
// @Failure      500      {object}  anthropicTypes.ErrorResponse
// @Router       /multi/v1/messages [post]
// @Security     ApiKeyAuth
func (h *Handler) Messages(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "anthropic", "api_style", "compat", "request_name", "messages", "protocol_mode", "json")

	// 解析请求
	var req anthropicTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Anthropic Messages 请求参数校验失败", "error", err)
		c.JSON(http.StatusBadRequest, common.NewAnthropicErrorResponse(fmt.Sprintf("无效的请求格式： %v", err), http.StatusBadRequest, err))
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		// 流式响应
		h.handleAnthropicStreamResponse(c, &req)
		return
	}

	// 非流式响应
	resp, err := h.gatewayService.AnthropicCompatMessages(c.Request.Context(), &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "处理请求时出错")
		logger.Warn("Anthropic Messages 请求失败，返回 HTTP JSON 错误",
			"status_code", mappedErr.StatusCode,
			"error_type", mappedErr.ErrorType,
			"error_code", mappedErr.ErrorCode,
		)
		c.JSON(mappedErr.StatusCode, common.NewAnthropicErrorResponse(mappedErr.Message, mappedErr.StatusCode, err, &mappedErr))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleAnthropicStreamResponse 处理 Anthropic 流式响应。
// 设置 SSE 头部，通过 ChatCompletionStream 获取事件通道，将流式事件转换为 Anthropic 格式并写入响应流。
// 包含 panic 恢复机制，发生错误时发送错误事件并记录日志。
func (h *Handler) handleAnthropicStreamResponse(c *gin.Context, req *anthropicTypes.Request) {
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 获取流式响应通道
	resultChan := h.gatewayService.AnthropicCompatMessagesStreamResult(ctx, req)

	if h.collector != nil {
		// 使用流式跟踪，确保在流结束时减少连接数
		defer h.collector.DecrementConnection()
	}

	flusher, _ := c.Writer.(http.Flusher)
	streamWriterBroken := false
	streamStarted := false

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "anthropic", "api_style", "compat", "request_name", "messages", "protocol_mode", "sse", "flow", "stream")
	// 添加 defer recover 来捕获流式处理中的 panic
	defer func() {
		if r := recover(); r != nil {
			cancel()
			stack := debug.Stack()
			// 将堆栈信息按行分割，以数组形式记录，提高 JSON 日志可读性
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("流式响应处理发生 panic",
				"panic", r,
				"stack", stackLines,
				"stream_phase", "panic",
			)
			panicErr := fmt.Errorf("panic: %v", r)
			if !streamStarted {
				c.JSON(http.StatusInternalServerError, common.NewAnthropicErrorResponse("服务器内部错误", http.StatusInternalServerError, panicErr))
				return
			}

			if streamWriterBroken {
				logger.Error("panic 后连接已不可恢复，跳过补写 Anthropic 流式错误事件")
				return
			}

			if sendErr := common.WriteAnthropicSSEError(c.Writer, "服务器内部错误", http.StatusInternalServerError, panicErr); sendErr != nil {
				if common.IsAnthropicStreamWriteError(sendErr) {
					streamWriterBroken = true
					logger.Error("panic 后发送 Anthropic 流式错误事件失败，连接已不可恢复", "error", sendErr)
					return
				}
				logger.Error("panic 后发送 Anthropic 流式错误事件失败", "error", sendErr)
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
		logger.Warn("Anthropic 流式建流前收到可代理 HTTP 协议错误",
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
			logger.Warn("上游返回 Anthropic 流式协议错误事件",
				"event_type", result.EventType,
				"status_code", result.ProtocolError.StatusCode,
				"error_type", result.ProtocolError.ErrorType,
				"error_code", result.ProtocolError.ErrorCode,
				"stream_phase", "streaming",
			)

			if writeErr := common.WriteAnthropicSSEError(c.Writer, result.ProtocolError.Message, result.ProtocolError.StatusCode, nil, result.ProtocolError); writeErr != nil {
				if common.IsAnthropicStreamWriteError(writeErr) {
					streamWriterBroken = true
					logger.Error("无法发送 Anthropic 流式错误事件，连接已不可恢复", "error", writeErr)
				} else {
					logger.Error("无法发送 Anthropic 流式错误事件", "error", writeErr)
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

		data, marshalErr := json.Marshal(result.Event)
		if marshalErr != nil {
			cancel()
			logger.Error("无法序列化流式事件", "error", marshalErr, "stream_phase", "streaming")

			if streamWriterBroken {
				logger.Error("连接已不可恢复，跳过补写 Anthropic 流式错误事件")
				return true
			}

			if writeErr := common.WriteAnthropicSSEError(c.Writer, "无法序列化流式事件", http.StatusInternalServerError, marshalErr); writeErr != nil {
				if common.IsAnthropicStreamWriteError(writeErr) {
					streamWriterBroken = true
					logger.Error("序列化失败后补写 Anthropic 流式错误事件失败，连接已不可恢复", "error", writeErr)
				} else {
					logger.Error("序列化失败后补写 Anthropic 流式错误事件失败", "error", writeErr)
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
			logger.Error("写入流式响应失败，连接已不可恢复", "error", err, "stream_phase", "streaming")
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
