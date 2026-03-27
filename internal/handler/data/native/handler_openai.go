package native

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/internal/handler/data/common"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	"github.com/gin-gonic/gin"
)

// OpenAIChatCompletions 处理原生 OpenAI 聊天补全请求，路径为 POST /multi/native/v1/chat/completions。
// 解析请求体，处理 User-Agent 头部，根据 stream 参数决定返回流式或非流式响应。
// 成功时返回 200 和响应数据，失败时返回 400 或 500 错误。
//
//	@Summary      OpenAI 聊天补全
//	@Description  处理原生 OpenAI API 的 chat completions 请求，支持流式和非流式两种模式
//	@Tags         native-openai
//	@Accept       json
//	@Produce      json
//	@Param        request  body      openaiChatTypes.Request  true  "请求体"
//	@Success      200      {object}  openaiChatTypes.Response  "成功"
//	@Failure      400      {object}  common.OpenAIHTTPErrorResponse  "无效的请求体"
//	@Failure      500      {object}  common.OpenAIHTTPErrorResponse  "请求失败"
//	@Router       /multi/native/v1/chat/completions [post]
//	@Security     ApiKeyAuth
func (h *Handler) OpenAIChatCompletions(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "openai", "api_style", "native")
	var req openaiChatTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("解析 OpenAI Chat 请求体失败", "error", err)
		c.JSON(
			http.StatusBadRequest,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("无效的请求体: %v", err), http.StatusBadRequest, err),
		)
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		h.streamOpenAIChat(c, &req, true)
		return
	}

	resp, err := h.gatewayService.OpenAINativeChatCompletion(c.Request.Context(), &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "请求失败")
		c.JSON(
			mappedErr.StatusCode,
			common.NewOpenAIHTTPErrorResponse(mappedErr.Message, mappedErr.StatusCode, err),
		)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// OpenAIResponses 处理原生 OpenAI 响应请求，路径为 POST /multi/native/v1/responses。
// 解析请求体，处理 User-Agent 头部，根据 stream 参数决定返回流式或非流式响应。
// 成功时返回 200 和响应数据，失败时返回 400 或 500 错误。
//
//	@Summary      OpenAI 响应
//	@Description  处理原生 OpenAI API 的 responses 请求，支持流式和非流式两种模式
//	@Tags         native-openai
//	@Accept       json
//	@Produce      json
//	@Param        request  body      openaiResponsesTypes.Request  true  "请求体"
//	@Success      200      {object}  openaiResponsesTypes.Response  "成功"
//	@Failure      400      {object}  common.OpenAIHTTPErrorResponse  "无效的请求体"
//	@Failure      500      {object}  common.OpenAIHTTPErrorResponse  "请求失败"
//	@Router       /multi/native/v1/responses [post]
//	@Security     ApiKeyAuth
func (h *Handler) OpenAIResponses(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "openai", "api_style", "native")
	var req openaiResponsesTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("解析 OpenAI Responses 请求体失败", "error", err)
		c.JSON(
			http.StatusBadRequest,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("无效的请求体: %v", err), http.StatusBadRequest, err),
		)
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		h.streamOpenAIResponses(c, &req, true)
		return
	}

	resp, err := h.gatewayService.OpenAINativeResponses(c.Request.Context(), &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "请求失败")
		c.JSON(
			mappedErr.StatusCode,
			common.NewOpenAIHTTPErrorResponse(mappedErr.Message, mappedErr.StatusCode, err),
		)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) streamOpenAIChat(c *gin.Context, req *openaiChatTypes.Request, sendDone bool) {
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	resultChan := h.gatewayService.OpenAINativeChatCompletionStreamResult(ctx, req)

	if h.collector != nil {
		defer h.collector.DecrementConnection()
	}

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "openai", "api_style", "native", "flow", "stream")
	defer func() {
		if r := recover(); r != nil {
			streamFailed = true
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines)
			if err := common.WriteOpenAIChatSSEError(c.Writer, fmt.Sprintf("异常: %v", r), http.StatusInternalServerError, nil); err != nil {
				logger.Error("发送 OpenAI Chat 流式错误失败", "error", err)
			}
		}
	}()

	for result := range resultChan {
		if result.Event == nil {
			continue
		}

		data, err := json.Marshal(result.Event)
		if err != nil {
			streamFailed = true
			cancel()
			logger.Error("序列化流事件失败", "error", err)
			if sendErr := common.WriteOpenAIChatSSEError(c.Writer, fmt.Sprintf("序列化流事件失败: %v", err), http.StatusInternalServerError, err); sendErr != nil {
				logger.Error("发送 OpenAI Chat 流式错误失败", "error", sendErr)
			}
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			cancel()
			logger.Error("写入流事件失败", "error", err)
			if sendErr := common.WriteOpenAIChatSSEError(c.Writer, fmt.Sprintf("写入流事件失败: %v", err), http.StatusInternalServerError, err); sendErr != nil {
				logger.Error("发送 OpenAI Chat 流式错误失败", "error", sendErr)
			}
			break
		}

		flusher.Flush()

		if result.Done {
			break
		}
	}

	if sendDone && !streamFailed {
		if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
			logger.Error("写入流结束标识失败", "error", err)
		}
		flusher.Flush()
	}
}

func (h *Handler) streamOpenAIResponses(c *gin.Context, req *openaiResponsesTypes.Request, sendDone bool) {
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	resultChan := h.gatewayService.OpenAINativeResponsesStreamResult(ctx, req)

	if h.collector != nil {
		defer h.collector.DecrementConnection()
	}

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "openai", "api_style", "native", "flow", "stream")
	defer func() {
		if r := recover(); r != nil {
			streamFailed = true
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines)
			if err := common.WriteOpenAIResponsesSSEError(c.Writer, fmt.Sprintf("异常: %v", r), http.StatusInternalServerError, nil); err != nil {
				logger.Error("发送 OpenAI Responses 流式错误失败", "error", err)
			}
		}
	}()

	for result := range resultChan {
		if result.Event == nil {
			continue
		}

		if result.ErrorMessage != "" {
			streamFailed = true
			cancel()
			logger.Warn("上游返回 OpenAI Responses 流式错误事件", "error_message", result.ErrorMessage)
			if sendErr := common.WriteOpenAIResponsesSSEError(c.Writer, result.ErrorMessage, http.StatusInternalServerError, fmt.Errorf("%s", result.ErrorMessage)); sendErr != nil {
				logger.Error("发送 OpenAI Responses 流式错误失败", "error", sendErr)
			}
			break
		}

		data, err := json.Marshal(result.Event)
		if err != nil {
			streamFailed = true
			cancel()
			logger.Error("序列化流事件失败", "error", err)
			if sendErr := common.WriteOpenAIResponsesSSEError(c.Writer, fmt.Sprintf("序列化流事件失败: %v", err), http.StatusInternalServerError, err); sendErr != nil {
				logger.Error("发送 OpenAI Responses 流式错误失败", "error", sendErr)
			}
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			cancel()
			logger.Error("写入流事件失败", "error", err)
			if sendErr := common.WriteOpenAIResponsesSSEError(c.Writer, fmt.Sprintf("写入流事件失败: %v", err), http.StatusInternalServerError, err); sendErr != nil {
				logger.Error("发送 OpenAI Responses 流式错误失败", "error", sendErr)
			}
			break
		}

		flusher.Flush()

		if result.Done {
			break
		}
	}

	if sendDone && !streamFailed {
		if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
			logger.Error("写入流结束标识失败", "error", err)
		}
		flusher.Flush()
	}
}
