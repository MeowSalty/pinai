package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/handlers/openai/types"
	"github.com/MeowSalty/pinai/services"
	"github.com/MeowSalty/portal/adapter/openai"
	openaiTypes "github.com/MeowSalty/portal/adapter/openai/types"
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
// @Success      200  {object}  types.ModelList
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

	modelList := types.ModelList{
		Object: "list",
		Data:   make([]types.Model, 0, len(models)),
	}

	for _, model := range models {
		modelList.Data = append(modelList.Data, types.Model{
			ID:     model.Name,
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
	var req openaiTypes.ChatCompletionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
	}

	// 转换请求格式
	portalReq := openai.ChatCompletionRequestToRequest(&req)

	// 调用 AI 网关服务
	ctx := context.Background()

	if req.Stream {
		// 流式响应
		return h.handleStreamResponse(c, ctx, portalReq)
	}

	// 非流式响应
	resp, err := h.aigatewayService.ChatCompletion(ctx, portalReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
	}

	// 转换响应格式
	openaiResp := openai.ResponseToChatCompletionResponse(resp)

	return c.JSON(openaiResp)
}

// handleStreamResponse 处理流式响应
func (h *OpenAIHandler) handleStreamResponse(c *fiber.Ctx, ctx context.Context, req *portalTypes.Request) error {
	// 设置流式响应头
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// 获取流式响应通道
	responseChan, err := h.aigatewayService.ChatCompletionStream(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("启动流式传输时出错：%v", err),
		})
	}

	// 获取原始 TCP 连接
	conn := c.Context().Response.BodyWriter()
	writer := bufio.NewWriterSize(conn, 1024)

	// 发送流式事件
	var lastResp *portalTypes.Response
	for resp := range responseChan {
		// 转换为 OpenAI 格式
		openaiResp := openai.ResponseToChatCompletionResponse(resp)

		// 发送事件
		data, _ := json.Marshal(openaiResp)
		_, err := fmt.Fprintf(writer, "data: %s\n\n", data)
		if err != nil {
			slog.Error("写入流式响应失败", "error", err)
			break
		}

		// 刷新缓冲区
		writer.Flush()
		lastResp = resp
	}

	// 发送结束标记和最终统计信息
	if lastResp != nil {
		// 添加结束原因
		if len(lastResp.Choices) > 0 {
			finishReason := "stop"
			lastResp.Choices[0].FinishReason = &finishReason
		}

		// 转换并发送最终消息
		finalResp := openai.ResponseToChatCompletionResponse(lastResp)
		finalData, _ := json.Marshal(finalResp)
		fmt.Fprintf(writer, "data: %s\n\n", finalData)
	}

	// 发送流结束标记
	_, err = fmt.Fprintf(writer, "data: [DONE]\n\n")
	if err != nil {
		slog.Error("写入流结束标记失败", "error", err)
	}

	writer.Flush()
	return nil
}
