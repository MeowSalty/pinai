package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/services"
	statsService "github.com/MeowSalty/pinai/services/stats"
	"github.com/MeowSalty/portal/request/adapter/openai/converter"
	openaiTypes "github.com/MeowSalty/portal/request/adapter/openai/types"
	portalTypes "github.com/MeowSalty/portal/types"
	"github.com/gofiber/fiber/v2"
)

// OpenAIHandler 结构体定义了 OpenAI 兼容 API 的处理器
//
// 该结构体封装了处理 OpenAI 兼容 API 请求所需的服务和日志记录器
type OpenAIHandler struct {
	// aigatewayService AI 网关服务实例，用于处理 AI 相关请求
	aigatewayService services.PortalService
}

// New 创建并初始化一个新的 OpenAI API 处理器实例
//
// 该函数使用依赖注入的方式创建 OpenAIHandler 实例
//
// 参数：
//   - aigatewayService: AI 网关服务实例，用于处理 AI 相关请求
//
// 返回值：
//   - *OpenAIHandler: 初始化后的 OpenAI 处理器实例
func New(aigatewayService services.PortalService) *OpenAIHandler {
	return &OpenAIHandler{
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
// @Router       /openai/v1/models [get]
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

// ChatCompletions 处理聊天完成请求
// @Summary      聊天完成
// @Description  创建聊天完成响应
// @Tags         OpenAI
// @Accept       json
// @Produce      json
// @Param        request  body      openaiTypes.ChatCompletionRequest  true  "聊天完成请求"
// @Success      200      {object}  openaiTypes.ChatCompletionResponse
// @Failure      400      {object}  fiber.Map
// @Failure      401      {object}  fiber.Map
// @Failure      500      {object}  fiber.Map
// @Router       /openai/v1/chat/completions [post]
func (h *OpenAIHandler) ChatCompletions(c *fiber.Ctx) error {
	// 解析请求
	var req openaiTypes.Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
	}

	// 转换请求格式
	portalReq := converter.ConvertCoreRequest(&req)

	if *req.Stream {
		// 流式响应
		return h.handleStreamResponse(c, portalReq)
	}

	// 非流式响应
	resp, err := h.aigatewayService.ChatCompletion(c.Context(), portalReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	// 转换响应格式
	openaiResp := converter.ConvertResponse(resp)

	return c.JSON(openaiResp)
}

// handleStreamResponse 处理流式响应
func (h *OpenAIHandler) handleStreamResponse(c *fiber.Ctx, req *portalTypes.Request) error {
	// 设置流式响应头
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(c.Context())

	// 获取流式响应通道
	responseChan, err := h.aigatewayService.ChatCompletionStream(ctx, req)
	if err != nil {
		cancel()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("启动流式传输时出错：%v", err),
		})
	}

	// 使用流式跟踪包装器，确保在流结束时减少连接数
	collector := statsService.GetCollector()
	c.Context().SetBodyStreamWriter(collector.WithStreamTracking(func(w *bufio.Writer) error {
		// var lastResp *portalTypes.Response
		isErr := false
		for resp := range responseChan {
			// 检查是否有错误字段
			if len(resp.Choices) > 0 && resp.Choices[0].Error != nil {
				isErr = true
				// 构造并发送错误事件给客户端
				errorEvent := map[string]any{
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

			// 转换为 OpenAI 格式
			openaiResp := converter.ConvertResponse(resp)

			// 发送事件
			data, _ := json.Marshal(openaiResp)
			_, err := fmt.Fprintf(w, "data: %s\n\n", data)
			if err != nil {
				cancel()
				slog.Error("写入流式响应失败", "error", err)
				break
			}

			// 刷新缓冲区
			w.Flush()
			// lastResp = resp
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
