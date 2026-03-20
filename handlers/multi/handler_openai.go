package multi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/handlers/multi/common"
	"github.com/MeowSalty/pinai/services/stats"
	portalLib "github.com/MeowSalty/portal"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	"github.com/gin-gonic/gin"
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
// @Failure      400      {object}  common.OpenAIHTTPErrorResponse
// @Failure      401      {object}  common.OpenAIHTTPErrorResponse
// @Failure      500      {object}  common.OpenAIHTTPErrorResponse
// @Router       /multi/v1/chat/completions [post]
// @Security     ApiKeyAuth
func (h *Handler) ChatCompletions(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)

	// 解析请求
	var req openaiChatTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("OpenAI ChatCompletion 请求参数校验失败", "error", err)
		c.JSON(
			http.StatusBadRequest,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("无效的请求格式：%v", err), http.StatusBadRequest, err),
		)
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		// 流式响应
		h.streamOpenAIChat(c, &req, true)
		return
	}

	// 非流式响应
	resp, err := h.portalService.NativeOpenAIChatCompletion(c.Request.Context(), &req, portalLib.WithCompatMode())
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("处理请求时出错：%v", err), http.StatusInternalServerError, err),
		)
		return
	}

	c.JSON(http.StatusOK, resp)
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
// @Failure      400      {object}  common.OpenAIHTTPErrorResponse
// @Failure      401      {object}  common.OpenAIHTTPErrorResponse
// @Failure      500      {object}  common.OpenAIHTTPErrorResponse
// @Router       /multi/v1/responses [post]
// @Security     ApiKeyAuth
func (h *Handler) Responses(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)

	var req openaiResponsesTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("OpenAI Responses 请求参数校验失败", "error", err)
		c.JSON(
			http.StatusBadRequest,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("无效的请求格式：%v", err), http.StatusBadRequest, err),
		)
		return
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		h.streamOpenAIResponses(c, &req, true)
		return
	}

	resp, err := h.portalService.NativeOpenAIResponses(c.Request.Context(), &req, portalLib.WithCompatMode())
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("处理请求时出错：%v", err), http.StatusInternalServerError, err),
		)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) streamOpenAIChat(c *gin.Context, req *openaiChatTypes.Request, sendDone bool) {
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	eventChan := h.portalService.NativeOpenAIChatCompletionStream(ctx, req, portalLib.WithCompatMode())

	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)
	defer func() {
		if r := recover(); r != nil {
			streamFailed = true
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("流式响应处理发生 panic",
				"panic", r,
				"stack", stackLines,
			)
			if err := common.WriteOpenAIChatSSEError(c.Writer, fmt.Sprintf("服务器内部错误: %v", r), http.StatusInternalServerError, nil); err != nil {
				logger.Error("发送 OpenAI Chat 流式错误失败", "error", err)
			}
		}
	}()

	for event := range eventChan {
		data, err := json.Marshal(event)
		if err != nil {
			streamFailed = true
			cancel()
			logger.Error("无法序列化事件", "error", err)
			if sendErr := common.WriteOpenAIChatSSEError(c.Writer, fmt.Sprintf("无法序列化事件: %v", err), http.StatusInternalServerError, err); sendErr != nil {
				logger.Error("发送 OpenAI Chat 流式错误失败", "error", sendErr)
			}
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			cancel()
			logger.Error("写入流式响应失败", "error", err)
			if sendErr := common.WriteOpenAIChatSSEError(c.Writer, fmt.Sprintf("写入流式响应失败: %v", err), http.StatusInternalServerError, err); sendErr != nil {
				logger.Error("发送 OpenAI Chat 流式错误失败", "error", sendErr)
			}
			break
		}

		flusher.Flush()
	}

	if sendDone && !streamFailed {
		if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
			cancel()
			logger.Error("写入流结束标记失败", "error", err)
		}
		flusher.Flush()
	}
}

func (h *Handler) streamOpenAIResponses(c *gin.Context, req *openaiResponsesTypes.Request, sendDone bool) {
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	eventChan := h.portalService.NativeOpenAIResponsesStream(ctx, req, portalLib.WithCompatMode())

	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method)
	defer func() {
		if r := recover(); r != nil {
			streamFailed = true
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("流式响应处理发生 panic",
				"panic", r,
				"stack", stackLines,
			)
			if err := common.WriteOpenAIResponsesSSEError(c.Writer, fmt.Sprintf("服务器内部错误: %v", r), http.StatusInternalServerError, nil); err != nil {
				logger.Error("发送 OpenAI Responses 流式错误失败", "error", err)
			}
		}
	}()

	for event := range eventChan {
		data, err := json.Marshal(event)
		if err != nil {
			streamFailed = true
			cancel()
			logger.Error("无法序列化事件", "error", err)
			if sendErr := common.WriteOpenAIResponsesSSEError(c.Writer, fmt.Sprintf("无法序列化事件: %v", err), http.StatusInternalServerError, err); sendErr != nil {
				logger.Error("发送 OpenAI Responses 流式错误失败", "error", sendErr)
			}
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			cancel()
			logger.Error("写入流式响应失败", "error", err)
			if sendErr := common.WriteOpenAIResponsesSSEError(c.Writer, fmt.Sprintf("写入流式响应失败: %v", err), http.StatusInternalServerError, err); sendErr != nil {
				logger.Error("发送 OpenAI Responses 流式错误失败", "error", sendErr)
			}
			break
		}

		flusher.Flush()
	}

	if sendDone && !streamFailed {
		if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
			cancel()
			logger.Error("写入流结束标记失败", "error", err)
		}
		flusher.Flush()
	}
}
