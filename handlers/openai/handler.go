package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/services/portal"
	"github.com/MeowSalty/pinai/services/stats"
	"github.com/MeowSalty/portal/logger"
	openaiChatConverter "github.com/MeowSalty/portal/request/adapter/openai/converter/chat"
	openaiResponsesConverter "github.com/MeowSalty/portal/request/adapter/openai/converter/responses"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	portalTypes "github.com/MeowSalty/portal/request/adapter/types"
	"github.com/gin-gonic/gin"
)

// OpenAIHandler 结构体定义了 OpenAI 兼容 API 的处理器
//
// 该结构体封装了处理 OpenAI 兼容 API 请求所需的服务和日志记录器
type OpenAIHandler struct {
	// portalService AI 网关服务实例，用于处理 AI 相关请求
	portalService portal.Service
	// userAgent User-Agent 配置，用于控制请求的 User-Agent 头部
	userAgent string
	logger    *slog.Logger
}

// New 创建并初始化一个新的 OpenAI API 处理器实例
//
// 该函数使用依赖注入的方式创建 OpenAIHandler 实例
//
// 参数：
//   - portalService: AI 网关服务实例，用于处理 AI 相关请求
//   - userAgent: User-Agent 配置，空则透传客户端 UA，"default" 使用 Go net/http 默认值，其他字符串则复写
//
// 返回值：
//   - *OpenAIHandler: 初始化后的 OpenAI 处理器实例
func New(portalService portal.Service, userAgent string, logger *slog.Logger) *OpenAIHandler {
	return &OpenAIHandler{
		portalService: portalService,
		userAgent:     userAgent,
		logger:        logger,
	}
}

// ListModels 处理获取可用模型列表的请求
// @Summary      列出模型
// @Description  获取所有可用的 AI 模型列表
// @Tags         OpenAI
// @Accept       json
// @Produce      json
// @Success      200  {object}  ModelList
// @Failure      500  {object}  gin.H
// @Router       /openai/v1/models [get]
func ListModels(c *gin.Context) {
	q := query.Q
	m := q.Model

	models, err := m.WithContext(c.Request.Context()).Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("无法获取模型列表：%v", err),
		})
		return
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

	c.JSON(http.StatusOK, modelList)
}

// ChatCompletions 处理聊天完成请求
// @Summary      聊天完成
// @Description  创建聊天完成响应
// @Tags         OpenAI
// @Accept       json
// @Produce      json
// @Param        request  body      openaiTypes.ChatCompletionRequest  true  "聊天完成请求"
// @Success      200      {object}  openaiTypes.ChatCompletionResponse
// @Failure      400      {object}  gin.H
// @Failure      401      {object}  gin.H
// @Failure      500      {object}  gin.H
// @Router       /openai/v1/chat/completions [post]
func (h *OpenAIHandler) ChatCompletions(c *gin.Context) {
	// 解析请求
	var req openaiChatTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
		return
	}

	// 转换请求格式
	portalReq, err := openaiChatConverter.RequestToContract(&req)

	// 处理 User-Agent 头部
	if portalReq.Headers == nil {
		portalReq.Headers = make(map[string]string)
	}

	// 根据配置处理 User-Agent
	switch h.userAgent {
	case "":
		// 空字符串：透传客户端的 User-Agent
		if userAgent := c.GetHeader("User-Agent"); userAgent != "" {
			portalReq.Headers["User-Agent"] = userAgent
		}
	case "default":
		// "default"：不设置 User-Agent，使用 Go net/http 默认值
		// 不添加 User-Agent 到 Headers 中
	default:
		// 其他字符串：使用配置的字符串复写 User-Agent
		portalReq.Headers["User-Agent"] = h.userAgent
	}

	if portalReq.Stream != nil && *portalReq.Stream {
		// 流式响应
		h.handleStreamResponse(c, portalReq)
		return
	}

	// 非流式响应
	resp, err := h.portalService.ChatCompletion(c.Request.Context(), portalReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
		return
	}

	// 转换响应格式
	openaiResp, err := openaiChatConverter.ResponseFromContract(resp, logger.Default())

	c.JSON(http.StatusOK, openaiResp)
}

// Responses 处理 Responses API 请求
// @Summary      Responses
// @Description  创建 Responses API 响应
// @Tags         OpenAI
// @Accept       json
// @Produce      json
// @Param        request  body      openaiResponsesTypes.Request  true  "Responses 请求"
// @Success      200      {object}  openaiResponsesTypes.Response
// @Failure      400      {object}  gin.H
// @Failure      401      {object}  gin.H
// @Failure      500      {object}  gin.H
// @Router       /openai/v1/responses [post]
func (h *OpenAIHandler) Responses(c *gin.Context) {
	var req openaiResponsesTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("无效的请求格式： %v", err),
		})
		return
	}

	portalReq, err := openaiResponsesConverter.RequestToContract(&req)

	if portalReq.Headers == nil {
		portalReq.Headers = make(map[string]string)
	}

	switch h.userAgent {
	case "":
		if userAgent := c.GetHeader("User-Agent"); userAgent != "" {
			portalReq.Headers["User-Agent"] = userAgent
		}
	case "default":
		// "default"：不设置 User-Agent，使用 Go net/http 默认值
	default:
		portalReq.Headers["User-Agent"] = h.userAgent
	}

	if portalReq.Stream != nil && *portalReq.Stream {
		h.handleResponsesStream(c, portalReq)
		return
	}

	resp, err := h.portalService.ChatCompletion(c.Request.Context(), portalReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("处理请求时出错：%v", err),
		})
		return
	}

	openaiResp, err := openaiResponsesConverter.ResponseFromContract(resp, logger.Default())
	c.JSON(http.StatusOK, openaiResp)
}

// handleStreamResponse 处理流式响应
func (h *OpenAIHandler) handleStreamResponse(c *gin.Context, req *portalTypes.RequestContract) {
	// 设置流式响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 获取流式响应通道
	eventChan, err := h.portalService.ChatCompletionStream(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("启动流式传输时出错：%v", err),
		})
		return
	}

	// 确保在流结束时减少连接数
	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)

	path := c.Request.URL.Path
	method := c.Request.Method
	logger := h.logger.With("path", path, "method", method)

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
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonBytes))
				if flusher != nil {
					flusher.Flush()
				}
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
				logger.Error("无法序列化错误事件", "error", marshalErr)
				break
			}

			// 发送错误事件
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonBytes)); err != nil {
				logger.Error("无法发送错误事件，写入流失败", "error", err)
				break
			}
			if flusher != nil {
				flusher.Flush()
			}
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
			slog.Error("无法序列化事件", "error", err)
			break
		}

		// 发送事件
		_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		if err != nil {
			cancel()
			logger.Error("写入流式响应失败", "error", err)
			break
		}

		// 刷新缓冲区
		if flusher != nil {
			flusher.Flush()
		}
	}
	if isErr {
		return
	}

	// 发送流结束标记
	_, err = fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	if err != nil {
		cancel()
		logger.Error("写入流结束标记失败", "error", err)
	}

	if flusher != nil {
		flusher.Flush()
	}
}

// handleResponsesStream 处理 Responses API 流式响应
func (h *OpenAIHandler) handleResponsesStream(c *gin.Context, req *portalTypes.RequestContract) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	eventChan, err := h.portalService.ChatCompletionStream(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("启动流式传输时出错：%v", err),
		})
		return
	}

	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)

	path := c.Request.URL.Path
	method := c.Request.Method
	logger := h.logger.With("path", path, "method", method)

	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("流式响应处理发生 panic",
				"panic", r,
				"path", path,
				"method", method,
				"stack", stackLines,
			)
			errorEvent := openaiResponsesTypes.ResponseError{
				Code:    openaiResponsesTypes.ResponseErrorCodeServerError,
				Message: fmt.Sprintf("流式响应处理错误: %v", r),
			}
			if jsonBytes, err := json.Marshal(errorEvent); err == nil {
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonBytes))
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
	}()

	isErr := false
	streamCtx := portalTypes.NewStreamIndexContext()
	for event := range eventChan {
		if event.Error != nil {
			isErr = true

			jsonBytes, marshalErr := json.Marshal(event.Error)
			if marshalErr != nil {
				logger.Error("无法序列化错误事件", "error", marshalErr)
				break
			}

			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonBytes)); err != nil {
				logger.Error("无法发送错误事件，写入流失败", "error", err)
				break
			}

			if flusher != nil {
				flusher.Flush()
			}
			break
		}

		openaiEvents, err := openaiResponsesConverter.StreamEventFormContract(event, portal.NewSlogAdapter(logger), streamCtx)
		if err != nil {
			cancel()
			logger.Error("无法转换事件", "error", err)
			break
		}

		for _, openaiEvent := range openaiEvents {
			data, _ := json.Marshal(openaiEvent)
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
				cancel()
				logger.Error("写入流式响应失败", "error", err)
				break
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
	if isErr {
		return
	}

	if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
		cancel()
		logger.Error("写入流结束标记失败", "error", err)
	}

	if flusher != nil {
		flusher.Flush()
	}
}
