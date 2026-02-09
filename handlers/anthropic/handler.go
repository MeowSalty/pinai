package anthropic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/services/portal"
	"github.com/MeowSalty/pinai/services/stats"
	"github.com/MeowSalty/portal/logger"
	"github.com/MeowSalty/portal/request/adapter/anthropic/converter"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	portalTypes "github.com/MeowSalty/portal/request/adapter/types"
	"github.com/gofiber/fiber/v2"
)

// AnthropicHandler 结构体定义了 Anthropic 兼容 API 的处理器
//
// 该结构体封装了处理 Anthropic 兼容 API 请求所需的服务和日志记录器
type AnthropicHandler struct {
	// portal AI 网关服务实例，用于处理 AI 相关请求
	portal portal.Service
	// userAgent User-Agent 配置，用于控制请求的 User-Agent 头部
	userAgent string
}

// New 创建并初始化一个新的 Anthropic API 处理器实例
//
// 该函数使用依赖注入的方式创建 AnthropicHandler 实例
//
// 参数：
//   - portal: AI 网关服务实例，用于处理 AI 相关请求
//   - userAgent: User-Agent 配置，空则透传客户端 UA，"default" 使用 fasthttp 默认值，其他字符串则复写
//
// 返回值：
//   - *AnthropicHandler: 初始化后的 Anthropic 处理器实例
func New(portal portal.Service, userAgent string) *AnthropicHandler {
	return &AnthropicHandler{
		portal:    portal,
		userAgent: userAgent,
	}
}

// ListModels 处理获取可用模型列表的请求
// @Summary      列出模型
// @Description  获取所有可用的 AI 模型列表
// @Tags         Anthropic
// @Accept       json
// @Produce      json
// @Success      200  {object}  ModelList
// @Failure      500  {object}  fiber.Map
// @Router       /anthropic/v1/models [get]
func ListModels(c *fiber.Ctx) error {
	q := query.Q
	m := q.Model

	models, err := m.WithContext(c.Context()).Find()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("无法获取模型列表：%v", err),
		})
	}

	modelList := ModelList{
		Object: "list",
		Data:   make([]Model, 0, len(models)),
	}

	for _, model := range models {
		modelID := model.Name
		if model.Alias != "" {
			modelID = model.Alias
		}

		modelList.Data = append(modelList.Data, Model{
			ID:     modelID,
			Object: "model",
		})
	}

	return c.JSON(modelList)
}

// Messages 处理消息完成请求
// @Summary      消息完成
// @Description  创建消息完成响应
// @Tags         Anthropic
// @Accept       json
// @Produce      json
// @Param        request  body      anthropicTypes.MessageRequest  true  "消息请求"
// @Success      200      {object}  anthropicTypes.MessageResponse
// @Failure      400      {object}  fiber.Map
// @Failure      401      {object}  fiber.Map
// @Failure      500      {object}  fiber.Map
// @Router       /anthropic/v1/messages [post]
func (h *AnthropicHandler) Messages(c *fiber.Ctx) error {
	// 解析请求
	var req anthropicTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
	}

	// 转换请求格式
	portalReq, err := converter.RequestToContract(&req)

	// 处理 User-Agent 头部
	if portalReq.Headers == nil {
		portalReq.Headers = make(map[string]string)
	}

	// 根据配置处理 User-Agent
	switch h.userAgent {
	case "":
		// 空字符串：透传客户端的 User-Agent
		if userAgent := c.Get("User-Agent"); userAgent != "" {
			portalReq.Headers["User-Agent"] = userAgent
		}
	case "default":
		// "default"：不设置 User-Agent，使用 fasthttp 默认值
		// 不添加 User-Agent 到 Headers 中
	default:
		// 其他字符串：使用配置的字符串复写 User-Agent
		portalReq.Headers["User-Agent"] = h.userAgent
	}

	if portalReq.Stream != nil && *portalReq.Stream {
		// 流式响应
		return h.handleStreamResponse(c, portalReq)
	}

	// 非流式响应
	resp, err := h.portal.ChatCompletion(c.Context(), portalReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	// 转换响应格式
	anthropicResp, err := converter.ResponseFromContract(resp, logger.Default())

	return c.JSON(anthropicResp)
}

// handleStreamResponse 处理流式响应
func (h *AnthropicHandler) handleStreamResponse(c *fiber.Ctx, req *portalTypes.RequestContract) error {
	// 设置流式响应头
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(c.Context())

	// 获取流式响应通道
	eventChan, err := h.portal.ChatCompletionStream(ctx, req)
	if err != nil {
		cancel()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("启动流式传输时出错：%v", err),
		})
	}

	// 使用流式跟踪包装器，确保在流结束时减少连接数
	collector := stats.GetCollector()
	c.Context().SetBodyStreamWriter(collector.WithStreamTracking(func(w *bufio.Writer) error {
		// 添加 defer recover 来捕获流式处理中的 panic
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				// 将堆栈信息按行分割，以数组形式记录，提高 JSON 日志可读性
				stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
				slog.Error("流式响应处理发生 panic",
					"panic", r,
					"path", c.Path(),
					"method", c.Method(),
					"body", string(c.Body()),
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
					// logger.Error("无法序列化错误事件", "error", marshalErr)
					break
				}

				// 发送错误事件
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes)); err != nil {
					// logger.Error("无法发送错误事件，写入流失败", "error", err)
					break
				}
				w.Flush()
				break
			}

			// 转换为 Anthropic 格式
			anthropicEvent, err := converter.StreamEventFromContract(event, nil)
			if err != nil {
				cancel()
				slog.Error("无法转换流式响应", "error", err)
				break
			}

			// 发送事件
			data, _ := json.Marshal(anthropicEvent)
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)

			if err != nil {
				cancel()
				slog.Error("写入流式响应失败", "error", err)
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
			slog.Error("写入流结束标记失败", "error", err)
		}

		w.Flush()
		return nil
	}))

	return nil
}
