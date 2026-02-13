package native

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/services/stats"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	openaiSharedTypes "github.com/MeowSalty/portal/request/adapter/openai/types/shared"
	"github.com/gofiber/fiber/v2"
)

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
//	@Failure      400      {object}  openaiSharedTypes.Error    "无效的请求体"
//	@Failure      500      {object}  openaiSharedTypes.Error    "请求失败"
//	@Router       /multi/native/v1/chat/completions [post]
//	@Security     ApiKeyAuth
func (h *Handler) OpenAIChatCompletions(c *fiber.Ctx) error {
	var req openaiChatTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(openaiSharedTypes.Error{
			Error: openaiSharedTypes.ErrorDetail{
				Message: fmt.Sprintf("无效的请求体: %v", err),
				Type:    "invalid_request_error",
				Code:    "invalid_request",
			},
		})
	}

	// 处理 User-Agent 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	applyUserAgent(req.Headers, h.userAgent, c)

	if req.Stream != nil && *req.Stream {
		return h.streamOpenAIChat(c, &req, true)
	}

	resp, err := h.portalService.NativeOpenAIChatCompletion(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(openaiSharedTypes.Error{
			Error: openaiSharedTypes.ErrorDetail{
				Message: fmt.Sprintf("请求失败: %v", err),
				Type:    "internal_error",
				Code:    "internal_error",
			},
		})
	}

	return c.JSON(resp)
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
//	@Failure      400      {object}  openaiSharedTypes.Error         "无效的请求体"
//	@Failure      500      {object}  openaiSharedTypes.Error         "请求失败"
//	@Router       /multi/native/v1/responses [post]
//	@Security     ApiKeyAuth
func (h *Handler) OpenAIResponses(c *fiber.Ctx) error {
	var req openaiResponsesTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(openaiSharedTypes.Error{
			Error: openaiSharedTypes.ErrorDetail{
				Message: fmt.Sprintf("无效的请求体: %v", err),
				Type:    "invalid_request_error",
				Code:    "invalid_request",
			},
		})
	}

	// 处理 User-Agent 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	applyUserAgent(req.Headers, h.userAgent, c)

	if req.Stream != nil && *req.Stream {
		return h.streamOpenAIResponses(c, &req, true)
	}

	resp, err := h.portalService.NativeOpenAIResponses(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(openaiSharedTypes.Error{
			Error: openaiSharedTypes.ErrorDetail{
				Message: fmt.Sprintf("请求失败: %v", err),
				Type:    "internal_error",
				Code:    "internal_error",
			},
		})
	}

	return c.JSON(resp)
}

func (h *Handler) streamOpenAIChat(c *fiber.Ctx, req *openaiChatTypes.Request, sendDone bool) error {
	setSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Context())
	eventChan := h.portalService.NativeOpenAIChatCompletionStream(ctx, req)

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
				h.sendOpenAIStreamError(w, "internal_error", fmt.Sprintf("异常: %v", r), "internal_error")
			}
		}()

		for event := range eventChan {
			data, err := json.Marshal(event)
			if err != nil {
				cancel()
				logger.Error("序列化流事件失败", "error", err)
				h.sendOpenAIStreamError(w, "internal_error", fmt.Sprintf("序列化流事件失败: %v", err), "internal_error")
				break
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				cancel()
				logger.Error("写入流事件失败", "error", err)
				h.sendOpenAIStreamError(w, "internal_error", fmt.Sprintf("写入流事件失败: %v", err), "internal_error")
				break
			}

			w.Flush()
		}

		if sendDone {
			if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
				cancel()
				logger.Error("写入流结束标识失败", "error", err)
				h.sendOpenAIStreamError(w, "internal_error", fmt.Sprintf("写入流结束标识失败: %v", err), "internal_error")
			}
			w.Flush()
		}

		return nil
	}))

	return nil
}

// sendOpenAIStreamError 发送流式错误响应
func (h *Handler) sendOpenAIStreamError(w *bufio.Writer, errorType, message, code string) {
	errResp := openaiSharedTypes.Error{
		Error: openaiSharedTypes.ErrorDetail{
			Message: message,
			Type:    errorType,
			Param:   nil,
			Code:    code,
		},
	}
	data, _ := json.Marshal(errResp)
	fmt.Fprintf(w, "data: %s\n\n", data)
	w.Flush()
}

func (h *Handler) streamOpenAIResponses(c *fiber.Ctx, req *openaiResponsesTypes.Request, sendDone bool) error {
	setSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Context())
	eventChan := h.portalService.NativeOpenAIResponsesStream(ctx, req)

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
				h.sendOpenAIStreamError(w, "internal_error", fmt.Sprintf("异常: %v", r), "internal_error")
			}
		}()

		for event := range eventChan {
			data, err := json.Marshal(event)
			if err != nil {
				cancel()
				logger.Error("序列化流事件失败", "error", err)
				h.sendOpenAIStreamError(w, "internal_error", fmt.Sprintf("序列化流事件失败: %v", err), "internal_error")
				break
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				cancel()
				logger.Error("写入流事件失败", "error", err)
				h.sendOpenAIStreamError(w, "internal_error", fmt.Sprintf("写入流事件失败: %v", err), "internal_error")
				break
			}

			w.Flush()
		}

		if sendDone {
			if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
				cancel()
				logger.Error("写入流结束标识失败", "error", err)
				h.sendOpenAIStreamError(w, "internal_error", fmt.Sprintf("写入流结束标识失败: %v", err), "internal_error")
			}
			w.Flush()
		}

		return nil
	}))

	return nil
}
