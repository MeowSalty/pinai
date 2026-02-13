package native

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/services/stats"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	"github.com/gofiber/fiber/v2"
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
//	@Failure      400      {object}  map[string]string        "无效的请求体"
//	@Failure      500      {object}  map[string]string        "请求失败"
//	@Router       /multi/native/v1/messages [post]
//	@Security     ApiKeyAuth
func (h *Handler) AnthropicMessages(c *fiber.Ctx) error {
	var req anthropicTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求体: %v", err),
		})
	}

	// 处理 User-Agent 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	applyUserAgent(req.Headers, h.userAgent, c)

	if req.Stream != nil && *req.Stream {
		return h.streamAnthropic(c, &req)
	}

	resp, err := h.portalService.NativeAnthropicMessages(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("请求失败: %v", err),
		})
	}

	return c.JSON(resp)
}

func (h *Handler) streamAnthropic(c *fiber.Ctx, req *anthropicTypes.Request) error {
	setSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Context())
	eventChan := h.portalService.NativeAnthropicMessagesStream(ctx, req)

	collector := stats.GetCollector()
	path := c.Path()
	method := c.Method()
	body := append([]byte(nil), c.Body()...)
	c.Context().SetBodyStreamWriter(collector.WithStreamTracking(func(w *bufio.Writer) error {
		logger := h.logger.With("path", path, "method", method, "body", string(body))
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
				logger.Error("原生流处理异常", "panic", r, "stack", stackLines)
			}
		}()

		for event := range eventChan {
			eventType, ok := anthropicEventType(event)
			if !ok {
				continue
			}

			data, err := json.Marshal(event)
			if err != nil {
				cancel()
				logger.Error("序列化流事件失败", "error", err)
				break
			}

			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data); err != nil {
				cancel()
				logger.Error("写入流事件失败", "error", err)
				break
			}

			w.Flush()

			if event.Error != nil {
				break
			}
		}

		return nil
	}))

	return nil
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
