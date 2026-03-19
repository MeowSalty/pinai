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
	"github.com/MeowSalty/portal"
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
// @Failure      400      {object}  gin.H
// @Failure      401      {object}  gin.H
// @Failure      500      {object}  gin.H
// @Router       /multi/v1/messages [post]
// @Security     ApiKeyAuth
func (h *Handler) Messages(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)

	// 解析请求
	var req anthropicTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Anthropic Messages 请求参数校验失败", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
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
	resp, err := h.portalService.NativeAnthropicMessages(c.Request.Context(), &req, portal.WithCompatMode())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
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
	eventChan := h.portalService.NativeAnthropicMessagesStream(ctx, req, portal.WithCompatMode())

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
			// 尝试发送错误信息给客户端
			errorEvent := map[string]any{
				"error": map[string]any{
					"type":    "internal_error",
					"message": fmt.Sprintf("服务器内部错误: %v", r),
				},
			}
			if jsonBytes, err := json.Marshal(errorEvent); err == nil {
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonBytes))
				flusher.Flush()
			}
		}
	}()

	isErr := false
	for event := range eventChan {
		if event == nil {
			continue
		}

		// 检查是否有错误字段
		if event.Error != nil {
			isErr = true
			logger.Warn("上游返回 Anthropic 流式错误事件",
				"event_type", event.Error.Type,
				"error_type", event.Error.Error.Error.Type,
				"error_message", event.Error.Error.Error.Message,
			)

			// 序列化错误事件
			jsonBytes, marshalErr := json.Marshal(event.Error)
			if marshalErr != nil {
				cancel()
				logger.Error("无法序列化错误事件", "error", marshalErr)
				break
			}

			// 发送错误事件
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonBytes)); err != nil {
				cancel()
				logger.Error("无法发送错误事件，写入流失败", "error", err)
				break
			}
			flusher.Flush()
			break
		}

		eventType, ok := anthropicEventType(event)
		if !ok {
			continue
		}

		// 发送事件
		data, marshalErr := json.Marshal(event)
		if marshalErr != nil {
			cancel()
			logger.Error("无法序列化流式事件", "error", marshalErr)
			break
		}
		_, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventType, data)

		if err != nil {
			cancel()
			logger.Error("写入流式响应失败", "error", err)
			break
		}

		// 刷新缓冲区
		flusher.Flush()
	}
	if isErr {
		return
	}

	// 发送流结束标记
	_, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	if err != nil {
		cancel()
		logger.Error("写入流结束标记失败", "error", err)
	}

	flusher.Flush()
}

func anthropicEventType(event *anthropicTypes.StreamEvent) (anthropicTypes.StreamEventType, bool) {
	if event == nil {
		return "", false
	}
	switch {
	case event.MessageStart != nil:
		return event.MessageStart.Type, true
	case event.MessageDelta != nil:
		return event.MessageDelta.Type, true
	case event.MessageStop != nil:
		return event.MessageStop.Type, true
	case event.ContentBlockStart != nil:
		return event.ContentBlockStart.Type, true
	case event.ContentBlockDelta != nil:
		return event.ContentBlockDelta.Type, true
	case event.ContentBlockStop != nil:
		return event.ContentBlockStop.Type, true
	case event.Ping != nil:
		return event.Ping.Type, true
	case event.Error != nil:
		return event.Error.Type, true
	default:
		return "", false
	}
}
