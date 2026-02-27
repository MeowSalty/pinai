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
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	"github.com/gofiber/fiber/v2"
)

// GeminiGenerateContent 处理 Gemini generateContent 请求，路径为 POST /multi/v1beta/models/{model}:generateContent。
// 解析请求体并从参数或查询字符串中获取模型名称，转换为统一格式后调用 ChatCompletion 服务。
// 成功时返回 200 状态码及生成内容响应，失败时返回 400/500 状态码。
//
// @Summary      生成内容
// @Description  调用 Gemini 模型生成内容，支持通过路径参数或查询参数指定模型
// @Tags         Gemini
// @Accept       json
// @Produce      json
// @Param        model    path      string                           true   "模型名称"
// @Param        request  body      geminiTypes.Request  true  "生成内容请求"
// @Success      200      {object}  geminiTypes.Response
// @Failure      400      {object}  fiber.Map
// @Failure      401      {object}  fiber.Map
// @Failure      500      {object}  fiber.Map
// @Router       /multi/v1beta/models/{model}:generateContent [post]
// @Security     ApiKeyAuth
func (h *Handler) GeminiGenerateContent(c *fiber.Ctx) error {
	var req geminiTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求体: %v", err),
		})
	}

	if req.Model == "" {
		req.Model = strings.TrimSpace(c.Params("model"))
	}
	if req.Model == "" {
		req.Model = strings.TrimSpace(c.Query("model"))
	}
	if req.Model == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "缺少模型查询参数",
		})
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	applyUserAgent(req.Headers, h.userAgent, c)

	resp, err := h.portalService.NativeGeminiGenerateContent(c.Context(), &req, portal.WithCompatMode())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	return c.JSON(resp)
}

// GeminiStreamGenerateContent 处理 Gemini streamGenerateContent 请求，路径为 POST /multi/v1beta/models/{model}:streamGenerateContent。
// 解析请求体并从参数或查询字符串中获取模型名称，转换为统一格式后返回流式响应。
// 成功时返回流式 SSE 响应，失败时返回 400/500 状态码。
//
// @Summary      流式生成内容
// @Description  调用 Gemini 模型流式生成内容，支持通过路径参数或查询参数指定模型
// @Tags         Gemini
// @Accept       json
// @Produce      text/event-stream
// @Param        model    path      string                           true   "模型名称"
// @Param        request  body      geminiTypes.Request  true  "生成内容请求"
// @Success      200      {object}  geminiTypes.Candidate
// @Failure      400      {object}  fiber.Map
// @Failure      401      {object}  fiber.Map
// @Failure      500      {object}  fiber.Map
// @Router       /multi/v1beta/models/{model}:streamGenerateContent [post]
// @Security     ApiKeyAuth
func (h *Handler) GeminiStreamGenerateContent(c *fiber.Ctx) error {
	var req geminiTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求体: %v", err),
		})
	}

	if req.Model == "" {
		req.Model = strings.TrimSpace(c.Params("model"))
	}
	if req.Model == "" {
		req.Model = strings.TrimSpace(c.Query("model"))
	}
	if req.Model == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "缺少模型查询参数",
		})
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	applyUserAgent(req.Headers, h.userAgent, c)

	return h.handleGeminiStreamResponse(c, &req)
}

// handleGeminiStreamResponse 处理 Gemini 流式响应。
// 设置 SSE 头部，通过 ChatCompletionStream 获取事件通道，将流式事件转换为 Gemini 格式并写入响应流。
// 包含 panic 恢复机制，发生错误时发送错误事件并记录日志。
func (h *Handler) handleGeminiStreamResponse(c *fiber.Ctx, req *geminiTypes.Request) error {
	setSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Context())
	eventChan := h.portalService.NativeGeminiStreamGenerateContent(ctx, req, portal.WithCompatMode())

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
				logger.Error("流式响应处理发生 panic", "panic", r, "stack", stackLines)
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

		if _, err := fmt.Fprintf(w, "data: [DONE]\n\n"); err != nil {
			cancel()
			logger.Error("写入流结束标记失败", "error", err)
		}

		w.Flush()
		return nil
	}))

	return nil
}
