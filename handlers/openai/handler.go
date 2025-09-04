package openai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database"
	"github.com/MeowSalty/pinai/handlers/openai/types"
	"github.com/MeowSalty/pinai/services"
	"github.com/MeowSalty/portal/adapter/openai"
	openaiTypes "github.com/MeowSalty/portal/adapter/openai/types"
	portalTypes "github.com/MeowSalty/portal/types"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

// OpenAIHandler 结构体定义了 OpenAI 兼容 API 的处理器
//
// 该结构体封装了处理 OpenAI 兼容 API 请求所需的服务和日志记录器
type OpenAIHandler struct {
	// logger 日志记录器实例，用于记录处理过程中的日志信息
	logger *slog.Logger

	// aigatewayService AI 网关服务实例，用于处理 AI 相关请求
	aigatewayService services.PortalService
}

// New 创建并初始化一个新的 OpenAI API 处理器实例
//
// 该函数使用依赖注入的方式创建 OpenAIHandler 实例，符合 Go 语言的最佳实践
//
// 参数：
//   - aigatewayService: AI 网关服务实例，用于处理 AI 相关请求
//   - logger: 日志记录器实例，用于记录处理过程中的日志信息
//
// 返回值：
//   - *OpenAIHandler: 初始化后的 OpenAI 处理器实例
func New(aigatewayService services.PortalService, logger *slog.Logger) *OpenAIHandler {
	return &OpenAIHandler{
		logger:           logger,
		aigatewayService: aigatewayService,
	}
}

// ListModels 处理获取可用模型列表的请求
// @Summary      列出模型
// @Description  获取所有可用的 AI 模型列表
// @Tags         OpenAI
// @Accept       json
// @Produce      json
// @Success      200  {object}  ModelList
// @Failure      500  {object}  fiber.Map
// @Router       /v1/models [get]
func ListModels(c *fiber.Ctx) error {
	q := database.Q
	ctx := c.Context()

	dbModels, err := q.WithContext(ctx).Model.Find()
	if err != nil {
		slog.ErrorContext(ctx, "查询模型失败", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "查询模型失败",
		})
	}

	data := make([]types.Model, len(dbModels))
	for i, m := range dbModels {
		modelID := m.Name
		if m.Alias != "" {
			modelID = m.Alias
		}
		data[i] = types.Model{
			ID:      modelID,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "p-org",
		}
	}

	return c.JSON(types.ModelList{
		Object: "list",
		Data:   data,
	})
}

// ChatCompletions 处理聊天补全请求
//
// 该方法根据请求参数决定使用流式还是非流式方式处理聊天补全请求。
// 支持 OpenAI 兼容的聊天补全接口，可以处理各种模型的请求。
//
// 参数：
//   - c: Fiber 上下文对象，包含 HTTP 请求和响应相关信息
//
// 返回值：
//   - error: 处理过程中可能发生的错误
func (h *OpenAIHandler) ChatCompletions(c *fiber.Ctx) error {
	// 解析请求体到 ChatCompletionRequest 结构体
	var req openaiTypes.ChatCompletionRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.ErrorContext(c.Context(), "请求解析失败", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "无法解析请求"})
	}

	// 将请求转换为内部使用的 Request 结构体
	chatReq := openai.ChatCompletionRequestToRequest(&req)

	// 记录请求处理开始日志
	h.logger.InfoContext(c.Context(), "开始处理聊天完成请求",
		"model", req.Model,
		"message_count", len(req.Messages),
		"stream", req.Stream,
	)

	// 根据是否启用流式传输选择不同的处理方法
	if !req.Stream {
		return h.handleNonStream(c, chatReq)
	}

	return h.handleStream(c, chatReq)
}

// handleNonStream 处理非流式的聊天补全请求
//
// 该方法处理非流式的聊天补全请求，一次性返回完整的响应结果。
//
// 参数：
//   - c: Fiber 上下文对象，包含 HTTP 请求和响应相关信息
//   - req: 聊天补全请求对象
//
// 返回值：
//   - error: 处理过程中可能发生的错误
func (h *OpenAIHandler) handleNonStream(c *fiber.Ctx, req *portalTypes.Request) error {
	// 调用 AI 网关服务处理聊天补全请求
	resp, err := h.aigatewayService.ChatCompletion(c.Context(), req)
	if err != nil {
		h.logger.ErrorContext(c.Context(), "聊天完成处理失败", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "处理请求时发生内部错误"})
	}

	// 将响应转换为 OpenAI 兼容格式
	response := openai.ResponseToChatCompletionResponse(resp)

	// 记录处理成功日志，包含 token 使用情况
	if response != nil {
		h.logger.InfoContext(c.Context(), "聊天完成处理成功",
			"prompt_tokens", response.Usage.PromptTokens,
			"completion_tokens", response.Usage.CompletionTokens,
			"total_tokens", response.Usage.TotalTokens,
		)
	}

	// 返回 JSON 格式的响应
	return c.JSON(response)
}

// handleStream 处理流式的聊天补全请求
//
// 该方法处理流式的聊天补全请求，通过 Server-Sent Events (SSE) 实时返回响应结果。
//
// 参数：
//   - c: Fiber 上下文对象，包含 HTTP 请求和响应相关信息
//   - req: 聊天补全请求对象
//
// 返回值：
//   - error: 处理过程中可能发生的错误
func (h *OpenAIHandler) handleStream(c *fiber.Ctx, req *portalTypes.Request) error {
	// 设置流式响应的 HTTP 头信息
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// 调用 AI 网关服务启动聊天补全流程
	stream, err := h.aigatewayService.ChatCompletionStream(c.Context(), req)
	if err != nil {
		h.logger.ErrorContext(c.Context(), "无法启动聊天完成流", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "无法启动聊天完成流"})
	}

	h.logger.InfoContext(c.Context(), "启动聊天完成流成功")

	// 使用 StreamWriter 处理流式响应
	c.Status(fiber.StatusOK).Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// 遍历流中的每个数据块
		for chunk := range stream {
			// 检查是否有错误信息
			if len(chunk.Choices) != 0 && chunk.Choices[0].Error != nil {
				h.logger.ErrorContext(c.Context(), "从流中收到错误", "error", chunk.Choices[0].Error)
				// 构造并发送错误事件给客户端
				errorEvent := map[string]interface{}{
					"error": map[string]interface{}{
						"type":    "stream_error",
						"message": chunk.Choices[0].Error.Message,
					},
				}

				// 序列化错误事件
				jsonBytes, marshalErr := json.Marshal(errorEvent)
				if marshalErr != nil {
					h.logger.ErrorContext(c.Context(), "无法序列化错误事件", "error", marshalErr)
					break
				}

				// 发送错误事件
				fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
				w.Flush()
				break
			}

			// 序列化数据块
			jsonBytes, err := json.Marshal(chunk)
			if err != nil {
				h.logger.ErrorContext(c.Context(), "无法序列化 SSE 事件", "error", err)
				continue
			}

			// 写入 SSE 格式的数据
			fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))

			// 刷新写入器，确保数据被发送
			if err := w.Flush(); err != nil {
				h.logger.ErrorContext(c.Context(), "无法刷新写入器", "error", err)
				break // 客户端可能已断开连接
			}
		}

		// 发送流结束标记
		fmt.Fprintf(w, "data: [DONE]\n\n")
		if err := w.Flush(); err != nil {
			h.logger.ErrorContext(c.Context(), "无法刷新最后的 [DONE] 消息", "error", err)
		}
	}))

	return nil
}
