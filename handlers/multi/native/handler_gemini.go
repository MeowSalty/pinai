package native

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/services/stats"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	"github.com/gofiber/fiber/v2"
)

// GeminiGenerateContent 处理原生 Gemini generateContent 请求，路径为 POST /multi/native/v1beta/models/:model:generateContent。
// 解析请求体，处理 User-Agent 头部，从路径参数或查询参数中获取模型名称。
// 成功时返回 200 和响应数据，失败时返回 400 或 500 错误。
//
//	@Summary      生成 Gemini 内容
//	@Description  处理原生 Gemini API 的 generateContent 请求，非流式模式
//	@Tags         native-gemini
//	@Accept       json
//	@Produce      json
//	@Param        model    path      string                  true  "模型名称"
//	@Param        request  body      geminiTypes.Request     true  "请求体"
//	@Success      200      {object}  geminiTypes.Response    "成功"
//	@Failure      400      {object}  map[string]string       "无效的请求体或缺少模型参数"
//	@Failure      500      {object}  map[string]string       "请求失败"
//	@Router       /multi/native/v1beta/models/{model}:generateContent [post]
//	@Security     ApiKeyAuth
func (h *Handler) GeminiGenerateContent(c *fiber.Ctx) error {
	var req geminiTypes.Request
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

	resp, err := h.portalService.NativeGeminiGenerateContent(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("请求失败: %v", err),
		})
	}

	return c.JSON(resp)
}

// GeminiStreamGenerateContent 处理原生 Gemini streamGenerateContent 请求，路径为 POST /multi/native/v1beta/models/:model:streamGenerateContent。
// 解析请求体，处理 User-Agent 头部，从路径参数或查询参数中获取模型名称，返回流式响应。
// 成功时返回 200 和流式事件数据，失败时返回 400 错误。
//
//	@Summary      流式生成 Gemini 内容
//	@Description  处理原生 Gemini API 的 streamGenerateContent 请求，流式模式
//	@Tags         native-gemini
//	@Accept       json
//	@Produce      text/event-stream
//	@Param        model    path      string              true  "模型名称"
//	@Param        request  body      geminiTypes.Request true  "请求体"
//	@Success      200      {string}  string              "流式事件数据"
//	@Failure      400      {object}  map[string]string   "无效的请求体或缺少模型参数"
//	@Router       /multi/native/v1beta/models/{model}:streamGenerateContent [post]
//	@Security     ApiKeyAuth
func (h *Handler) GeminiStreamGenerateContent(c *fiber.Ctx) error {
	var req geminiTypes.Request
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

	return h.streamGemini(c, &req)
}

func (h *Handler) streamGemini(c *fiber.Ctx, req *geminiTypes.Request) error {
	setSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Context())
	eventChan := h.portalService.NativeGeminiStreamGenerateContent(ctx, req)

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
			data, err := json.Marshal(event)
			if err != nil {
				cancel()
				logger.Error("序列化流事件失败", "error", err)
				break
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				cancel()
				logger.Error("写入流事件失败", "error", err)
				break
			}

			w.Flush()
		}

		return nil
	}))

	return nil
}
