package multi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/services/stats"
	"github.com/MeowSalty/portal"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	"github.com/gofiber/fiber/v2"
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
// @Failure      400      {object}  fiber.Map
// @Failure      401      {object}  fiber.Map
// @Failure      500      {object}  fiber.Map
// @Router       /multi/v1/messages [post]
// @Security     ApiKeyAuth
func (h *Handler) Messages(c *fiber.Ctx) error {
	// 解析请求
	var req anthropicTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	applyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		// 流式响应
		return h.handleAnthropicStreamResponse(c, &req)
	}

	// 非流式响应
	resp, err := h.portalService.NativeAnthropicMessages(c.Context(), &req, portal.WithCompatMode())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	return c.JSON(resp)
}

// handleAnthropicStreamResponse 处理 Anthropic 流式响应。
// 设置 SSE 头部，通过 ChatCompletionStream 获取事件通道，将流式事件转换为 Anthropic 格式并写入响应流。
// 包含 panic 恢复机制，发生错误时发送错误事件并记录日志。
func (h *Handler) handleAnthropicStreamResponse(c *fiber.Ctx, req *anthropicTypes.Request) error {
	setSSEHeaders(c)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(c.Context())

	// 获取流式响应通道
	eventChan := h.portalService.NativeAnthropicMessagesStream(ctx, req, portal.WithCompatMode())

	// 使用流式跟踪包装器，确保在流结束时减少连接数
	collector := stats.GetCollector()
	path := c.Path()
	method := c.Method()
	body := append([]byte(nil), c.Body()...)
	c.Context().SetBodyStreamWriter(collector.WithStreamTracking(func(w *bufio.Writer) error {
		logger := h.logger.With("path", path, "method", method, "body", string(body))
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
					fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
					w.Flush()
				}
			}
		}()

		isErr := false
		for event := range eventChan {
			// 检查是否有错误字段
			if event.Error != nil {
				isErr = true

				// 序列化错误事件
				jsonBytes, marshalErr := json.Marshal(event.Error)
				if marshalErr != nil {
					cancel()
					logger.Error("无法序列化错误事件", "error", marshalErr)
					break
				}

				// 发送错误事件
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes)); err != nil {
					cancel()
					logger.Error("无法发送错误事件，写入流失败", "error", err)
					break
				}
				w.Flush()
				break
			}

			eventType, ok := anthropicEventType(event)
			if !ok {
				continue
			}

			// 发送事件
			data, _ := json.Marshal(event)
			_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)

			if err != nil {
				cancel()
				logger.Error("写入流式响应失败", "error", err)
				break
			}

			// 刷新缓冲区
			w.Flush()
		}
		if isErr {
			return nil
		}

		// 发送流结束标记
		_, err := fmt.Fprintf(w, "data: [DONE]\n\n")
		if err != nil {
			cancel()
			logger.Error("写入流结束标记失败", "error", err)
		}

		w.Flush()
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
