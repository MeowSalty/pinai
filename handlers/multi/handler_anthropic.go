package multi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/handlers/multi/common"
	"github.com/MeowSalty/pinai/services/stats"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	"github.com/gin-gonic/gin"
)

// Messages 处理 Anthropic 消息完成请求，路径为 POST /multi/v1/messages。
// 解析请求体并转换为统一格式，根据 stream 参数决定返回流式或非流式响应。
// 成功时返回 200 状态码及消息响应，失败时返回 400/401/500 状态码。
//
// @Summary      消息完成
// @Description  创建消息完成响应，支持流式和非流式两种模式
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
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)

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
		c.JSON(http.StatusInternalServerError, common.NewAnthropicErrorResponse(fmt.Sprintf("处理请求时出错：%v", err), http.StatusInternalServerError, err))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleAnthropicStreamResponse 处理 Anthropic 流式响应。
// 设置 SSE 头部，通过 ChatCompletionStream 获取事件通道，将流式事件转换为 Anthropic 格式并写入响应流。
// 包含 panic 恢复机制，发生错误时发送错误事件并记录日志。
func (h *Handler) handleAnthropicStreamResponse(c *gin.Context, req *anthropicTypes.Request) {
	common.SetBaseSSEHeaders(c)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 获取流式响应通道
	resultChan := h.gatewayService.AnthropicCompatMessagesStreamResult(ctx, req)

	// 使用流式跟踪，确保在流结束时减少连接数
	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)
	// 添加 defer recover 来捕获流式处理中的 panic
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			// 将堆栈信息按行分割，以数组形式记录，提高 JSON 日志可读性
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("流式响应处理发生 panic",
				"panic", r,
				"stack", stackLines,
			)
			if err := common.WriteAnthropicSSEError(c.Writer, fmt.Sprintf("服务器内部错误: %v", r), http.StatusInternalServerError, fmt.Errorf("panic: %v", r)); err != nil {
				logger.Error("panic 后发送 Anthropic 错误事件失败", "error", err)
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}()

	for result := range resultChan {
		if result.Event == nil {
			continue
		}

		if result.ErrorMessage != "" {
			logger.Warn("上游返回 Anthropic 流式错误事件",
				"event_type", result.EventType,
				"error_message", result.ErrorMessage,
			)

			if writeErr := common.WriteAnthropicSSEError(c.Writer, result.ErrorMessage, http.StatusInternalServerError, fmt.Errorf("%s", result.ErrorMessage)); writeErr != nil {
				cancel()
				logger.Error("无法发送标准 Anthropic 错误事件", "error", writeErr)
				break
			}

			if flusher != nil {
				flusher.Flush()
			}

			if result.Done {
				break
			}
			continue
		}

		// 发送事件
		data, marshalErr := json.Marshal(result.Event)
		if marshalErr != nil {
			cancel()
			logger.Error("无法序列化流式事件", "error", marshalErr)
			break
		}
		_, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", result.EventType, data)

		if err != nil {
			cancel()
			logger.Error("写入流式响应失败", "error", err)
			break
		}

		// 刷新缓冲区
		if flusher != nil {
			flusher.Flush()
		}

		if result.Done {
			break
		}
	}
}
