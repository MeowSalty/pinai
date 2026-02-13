package multi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/services/portal"
	"github.com/MeowSalty/pinai/services/stats"
	"github.com/MeowSalty/portal/logger"
	openaiChatConverter "github.com/MeowSalty/portal/request/adapter/openai/converter/chat"
	openaiResponsesConverter "github.com/MeowSalty/portal/request/adapter/openai/converter/responses"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	portalTypes "github.com/MeowSalty/portal/request/adapter/types"
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

	// 转换请求格式
	portalReq, err := openaiChatConverter.RequestToContract(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("请求转换失败：%v", err),
		})
	}

	// 处理 User-Agent 头部
	if portalReq.Headers == nil {
		portalReq.Headers = make(map[string]string)
	}
	applyUserAgent(portalReq.Headers, h.userAgent, c)

	if portalReq.Stream != nil && *portalReq.Stream {
		// 流式响应
		return h.handleStreamResponse(c, portalReq)
	}

	// 非流式响应
	resp, err := h.portalService.ChatCompletion(c.Context(), portalReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	// 转换响应格式
	openaiResp, err := openaiChatConverter.ResponseFromContract(resp, logger.Default())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("响应转换失败：%v", err),
		})
	}

	return c.JSON(openaiResp)
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

	portalReq, err := openaiResponsesConverter.RequestToContract(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("请求转换失败：%v", err),
		})
	}

	if portalReq.Headers == nil {
		portalReq.Headers = make(map[string]string)
	}
	applyUserAgent(portalReq.Headers, h.userAgent, c)

	if portalReq.Stream != nil && *portalReq.Stream {
		return h.handleResponsesStream(c, portalReq)
	}

	resp, err := h.portalService.ChatCompletion(c.Context(), portalReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	openaiResp, err := openaiResponsesConverter.ResponseFromContract(resp, logger.Default())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("响应转换失败：%v", err),
		})
	}
	return c.JSON(openaiResp)
}

// handleStreamResponse 处理 OpenAI ChatCompletions 流式响应。
// 设置 SSE 头部，通过 ChatCompletionStream 获取事件通道，将流式事件转换为 OpenAI 格式并写入响应流。
// 包含 panic 恢复机制，发生错误时发送错误事件并记录日志。
func (h *Handler) handleStreamResponse(c *fiber.Ctx, req *portalTypes.RequestContract) error {
	setSSEHeaders(c)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(c.Context())

	// 获取流式响应通道
	eventChan, err := h.portalService.ChatCompletionStream(ctx, req)
	if err != nil {
		cancel()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("启动流式传输时出错：%v", err),
		})
	}

	// 使用流式跟踪包装器，确保在流结束时减少连接数
	collector := stats.GetCollector()
	path := c.Path()
	method := c.Method()
	body := append([]byte(nil), c.Body()...)
	c.Context().SetBodyStreamWriter(collector.WithStreamTracking(func(w *bufio.Writer) error {
		// 创建日志记录器
		logger := h.logger.With("path", path, "method", method, "body", string(body))
		// 添加 defer recover 来捕获流式处理中的 panic
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				// 将堆栈信息按行分割，以数组形式记录，提高 JSON 日志可读性
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

		isErr := false
		for event := range eventChan {
			// 检查是否有错误字段
			if event.Error != nil {
				isErr = true

				// 序列化错误事件
				message := fmt.Sprintf("%v", event.Error)
				sendStreamError(w, "internal_error", message, "internal_error")
				break
			}

			// 转换为 OpenAI 格式
			openaiEvent, err := openaiChatConverter.StreamEventFormContract(event, portal.NewSlogAdapter(logger))
			if err != nil {
				cancel()
				logger.Error("无法转换事件", "error", err)
				break
			}

			// 序列化事件
			data, err := json.Marshal(openaiEvent)
			if err != nil {
				cancel()
				logger.Error("无法序列化事件", "error", err)
				sendStreamError(w, "internal_error", fmt.Sprintf("无法序列化事件: %v", err), "internal_error")
				break
			}

			// 发送事件
			_, err = fmt.Fprintf(w, "data: %s\n\n", data)
			if err != nil {
				cancel()
				logger.Error("写入流式响应失败", "error", err)
				sendStreamError(w, "internal_error", fmt.Sprintf("写入流式响应失败: %v", err), "internal_error")
				break
			}

			// 刷新缓冲区
			w.Flush()
		}
		if isErr {
			return nil
		}

		// 发送流结束标记
		_, err = fmt.Fprintf(w, "data: [DONE]\n\n")
		if err != nil {
			cancel()
			logger.Error("写入流结束标记失败", "error", err)
		}

		w.Flush()
		return nil
	}))

	return nil
}

// handleResponsesStream 处理 OpenAI Responses API 流式响应。
// 设置 SSE 头部，通过 ChatCompletionStream 获取事件通道，将流式事件转换为 Responses API 格式并写入响应流。
// 包含 panic 恢复机制，发生错误时发送错误事件并记录日志。
func (h *Handler) handleResponsesStream(c *fiber.Ctx, req *portalTypes.RequestContract) error {
	setSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Context())

	eventChan, err := h.portalService.ChatCompletionStream(ctx, req)
	if err != nil {
		cancel()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("启动流式传输时出错：%v", err),
		})
	}

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
					"path", c.Path(),
					"method", c.Method(),
					"body", string(c.Body()),
					"stack", stackLines,
				)
				sendStreamError(w, "internal_error", fmt.Sprintf("流式响应处理错误: %v", r), "internal_error")
			}
		}()

		isErr := false
		streamCtx := portalTypes.NewStreamIndexContext()
		for event := range eventChan {
			if event.Error != nil {
				isErr = true

				message := fmt.Sprintf("%v", event.Error)
				sendStreamError(w, "internal_error", message, "internal_error")
				break
			}

			openaiEvents, err := openaiResponsesConverter.StreamEventFormContract(event, portal.NewSlogAdapter(logger), streamCtx)
			if err != nil {
				cancel()
				logger.Error("无法转换事件", "error", err)
				break
			}

			for _, openaiEvent := range openaiEvents {
				data, err := json.Marshal(openaiEvent)
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
		}
		if isErr {
			return nil
		}

		if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
			cancel()
			logger.Error("写入流结束标记失败", "error", err)
		}

		w.Flush()
		return nil
	}))

	return nil
}
