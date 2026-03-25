package native

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
	var req anthropicTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("请求参数绑定失败",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"error", err)
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
		c.JSON(http.StatusInternalServerError, common.NewAnthropicErrorResponse(fmt.Sprintf("请求失败: %v", err), http.StatusInternalServerError, err))
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) streamAnthropic(c *gin.Context, req *anthropicTypes.Request) {
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	eventChan := h.portalService.NativeAnthropicMessagesStream(ctx, req)

	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines)
			if err := common.WriteAnthropicSSEError(c.Writer, fmt.Sprintf("服务器内部错误: %v", r), http.StatusInternalServerError, fmt.Errorf("panic: %v", r)); err != nil {
				logger.Error("panic 后发送 Anthropic 标准错误事件失败", "error", err)
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}()

	for event := range eventChan {
		eventType, ok := anthropicEventType(event)
		if !ok {
			continue
		}

		data, err := json.Marshal(event)
		if err != nil {
			logger.Error("序列化流事件失败", "error", err)
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventType, data); err != nil {
			logger.Error("写入流事件失败", "error", err)
			break
		}

		flusher.Flush()

		if event.Error != nil {
			break
		}
	}
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
