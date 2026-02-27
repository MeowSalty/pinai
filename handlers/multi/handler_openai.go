package multi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/services/stats"
	portalLib "github.com/MeowSalty/portal"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	"github.com/gofiber/fiber/v2"
)

// ChatCompletions 处理 OpenAI 聊天完成请求，路径为 POST /multi/v1/chat/completions。
// 解析请求体并转换为统一格式，根据 stream 参数决定返回流式或非流式响应。
// 成功时返回 200 状态码及聊天完成响应，失败时返回 400/401/500 状态码。
//
// @Summary      聊天完成
// @Description  创建聊天完成响应，支持流式和非流式两种模式
// @Tags         OpenAI
// @Accept       json
// @Produce      json
// @Param        request  body      openaiChatTypes.Request  true  "聊天完成请求"
// @Success      200      {object}  openaiChatTypes.Response
// @Failure      400      {object}  fiber.Map
// @Failure      401      {object}  fiber.Map
// @Failure      500      {object}  fiber.Map
// @Router       /multi/v1/chat/completions [post]
// @Security     ApiKeyAuth
func (h *Handler) ChatCompletions(c *fiber.Ctx) error {
	// 解析请求
	var req openaiChatTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
	}

	// 处理 User-Agent 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	applyUserAgent(req.Headers, h.userAgent, c)

	if req.Stream != nil && *req.Stream {
		// 流式响应
		return h.streamOpenAIChat(c, &req, true)
	}

	// 非流式响应
	resp, err := h.portalService.NativeOpenAIChatCompletion(c.Context(), &req, portalLib.WithCompatMode())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	return c.JSON(resp)
}

// Responses 处理 OpenAI Responses API 请求，路径为 POST /multi/v1/responses。
// 解析请求体并转换为统一格式，根据 stream 参数决定返回流式或非流式响应。
// 成功时返回 200 状态码及 Responses 响应，失败时返回 400/401/500 状态码。
//
// @Summary      Responses
// @Description  创建 Responses API 响应，支持流式和非流式两种模式
// @Tags         OpenAI
// @Accept       json
// @Produce      json
// @Param        request  body      openaiResponsesTypes.Request  true  "Responses 请求"
// @Success      200      {object}  openaiResponsesTypes.Response
// @Failure      400      {object}  fiber.Map
// @Failure      401      {object}  fiber.Map
// @Failure      500      {object}  fiber.Map
// @Router       /multi/v1/responses [post]
// @Security     ApiKeyAuth
func (h *Handler) Responses(c *fiber.Ctx) error {
	var req openaiResponsesTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	applyUserAgent(req.Headers, h.userAgent, c)

	if req.Stream != nil && *req.Stream {
		return h.streamOpenAIResponses(c, &req, true)
	}

	resp, err := h.portalService.NativeOpenAIResponses(c.Context(), &req, portalLib.WithCompatMode())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	return c.JSON(resp)
}

func (h *Handler) streamOpenAIChat(c *fiber.Ctx, req *openaiChatTypes.Request, sendDone bool) error {
	setSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Context())
	eventChan := h.portalService.NativeOpenAIChatCompletionStream(ctx, req, portalLib.WithCompatMode())

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
				logger.Error("流式响应处理发生 panic",
					"panic", r,
					"path", path,
					"method", method,
					"body", string(body),
					"stack", stackLines,
				)
				sendStreamError(w, "internal_error", fmt.Sprintf("服务器内部错误: %v", r), "internal_error")
			}
		}()

		for event := range eventChan {
			data, err := json.Marshal(event)
			if err != nil {
				cancel()
				logger.Error("无法序列化事件", "error", err)
				sendStreamError(w, "internal_error", fmt.Sprintf("无法序列化事件: %v", err), "internal_error")
				break
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				cancel()
				logger.Error("写入流式响应失败", "error", err)
				sendStreamError(w, "internal_error", fmt.Sprintf("写入流式响应失败: %v", err), "internal_error")
				break
			}

			w.Flush()
		}

		if sendDone {
			if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
				cancel()
				logger.Error("写入流结束标记失败", "error", err)
			}
			w.Flush()
		}

		return nil
	}))

	return nil
}

func (h *Handler) streamOpenAIResponses(c *fiber.Ctx, req *openaiResponsesTypes.Request, sendDone bool) error {
	setSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Context())
	eventChan := h.portalService.NativeOpenAIResponsesStream(ctx, req, portalLib.WithCompatMode())

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
				logger.Error("流式响应处理发生 panic",
					"panic", r,
					"path", path,
					"method", method,
					"body", string(body),
					"stack", stackLines,
				)
				sendStreamError(w, "internal_error", fmt.Sprintf("服务器内部错误: %v", r), "internal_error")
			}
		}()

		for event := range eventChan {
			data, err := json.Marshal(event)
			if err != nil {
				cancel()
				logger.Error("无法序列化事件", "error", err)
				sendStreamError(w, "internal_error", fmt.Sprintf("无法序列化事件: %v", err), "internal_error")
				break
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				cancel()
				logger.Error("写入流式响应失败", "error", err)
				sendStreamError(w, "internal_error", fmt.Sprintf("写入流式响应失败: %v", err), "internal_error")
				break
			}

			w.Flush()
		}

		if sendDone {
			if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
				cancel()
				logger.Error("写入流结束标记失败", "error", err)
			}
			w.Flush()
		}

		return nil
	}))

	return nil
}
