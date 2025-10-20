package anthropic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/services"
	"github.com/MeowSalty/portal/request/adapter/anthropic/converter"
	anthropicTypes "github.com/MeowSalty/portal/request/adapter/anthropic/types"
	portalTypes "github.com/MeowSalty/portal/types"
	"github.com/gofiber/fiber/v2"
)

// AnthropicHandler 结构体定义了 Anthropic 兼容 API 的处理器
//
// 该结构体封装了处理 Anthropic 兼容 API 请求所需的服务和日志记录器
type AnthropicHandler struct {
	// portal AI 网关服务实例，用于处理 AI 相关请求
	portal services.PortalService
}

// New 创建并初始化一个新的 Anthropic API 处理器实例
//
// 该函数使用依赖注入的方式创建 AnthropicHandler 实例
//
// 参数：
//   - aigatewayService: AI 网关服务实例，用于处理 AI 相关请求
//
// 返回值：
//   - *AnthropicHandler: 初始化后的 Anthropic 处理器实例
func New(portal services.PortalService) *AnthropicHandler {
	return &AnthropicHandler{
		portal: portal,
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

	portalReq := req.ConvertCoreRequest()

	if *req.Stream {
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
	anthropicResp := converter.ConvertResponse(resp)

	return c.JSON(anthropicResp)
}

// handleStreamResponse 处理流式响应
func (h *AnthropicHandler) handleStreamResponse(c *fiber.Ctx, req *portalTypes.Request) error {
	// 设置流式响应头
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(c.Context())

	// 获取流式响应通道
	responseChan, err := h.portal.ChatCompletionStream(ctx, req)
	if err != nil {
		cancel()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("启动流式传输时出错：%v", err),
		})
	}

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		isErr := false
		converterTool := converter.NewStreamEventConverter()
		for resp := range responseChan {
			// 检查是否有错误字段
			if len(resp.Choices) > 0 && resp.Choices[0].Error != nil {
				isErr = true
				// 构造并发送错误事件给客户端
				errorEvent := map[string]any{
					"type": "error",
					"error": map[string]any{
						"type":    "stream_error",
						"message": resp.Choices[0].Error.Message,
					},
				}

				// 序列化错误事件
				jsonBytes, marshalErr := json.Marshal(errorEvent)
				if marshalErr != nil {
					slog.Error("无法序列化错误事件", "error", marshalErr)
					break
				}

				// 发送错误事件
				if _, err := fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes)); err != nil {
					slog.Error("无法发送错误事件，写入流失败", "error", err)
					break
				}
				w.Flush()
				break
			}

			// 转换为 Anthropic 格式
			anthropicResp := converterTool.ConvertStreamEvents(resp)

			var err error
			for _, resp := range anthropicResp {
				// 发送事件
				data, _ := json.Marshal(resp)
				_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", resp.Type, data)
			}
			if err != nil {
				cancel()
				slog.Error("写入流式响应失败", "error", err)
				break
			}

			// 刷新缓冲区
			w.Flush()
		}
		if isErr {
			return
		}

		// 发送流结束标记
		_, err = fmt.Fprintf(w, "data: [DONE]\n\n")
		if err != nil {
			cancel()
			slog.Error("写入流结束标记失败", "error", err)
		}

		w.Flush()
	})

	return nil
}
