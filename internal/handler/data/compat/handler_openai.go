package multi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	"github.com/MeowSalty/pinai/internal/handler/data/common"
	openaiChatTypes "github.com/MeowSalty/portal/request/adapter/openai/types/chat"
	openaiResponsesTypes "github.com/MeowSalty/portal/request/adapter/openai/types/responses"
	"github.com/gin-gonic/gin"
)

// ChatCompletions 处理 OpenAI 聊天完成请求，路径为 POST /multi/v1/chat/completions。
// 解析请求体并转换为统一格式，根据 stream 参数决定返回流式或非流式响应。
// 非流式错误通过 HTTP JSON 返回；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE error 事件返回。
//
// @Summary      聊天完成
// @Description  创建聊天完成响应：非流式返回 JSON；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE error 事件返回
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
	logCtx := common.NewRequestLogContext(c, "openai", "compat", "chat_completions").
		WithExtra(map[string]string{"protocol_mode": "auto"})
	logger := logCtx.EnrichLogger(h.logger)

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

	logCtx = logCtx.WithModel(req.Model)

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		// 流式响应
		h.streamOpenAIChat(c, &req, logCtx, true)
		return
	}

	// 非流式响应
	if h.collector != nil {
		h.collector.IncrementConnection()
		defer h.collector.DecrementConnection()
	}

	ctx := logCtx.WithContext(c.Request.Context())
	resp, err := h.gatewayService.OpenAICompatChatCompletion(ctx, &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "处理请求时出错")
		c.JSON(
			mappedErr.StatusCode,
			common.NewOpenAIHTTPErrorResponse(mappedErr.Message, mappedErr.StatusCode, err, &mappedErr),
		)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Responses 处理 OpenAI Responses API 请求，路径为 POST /multi/v1/responses。
// 解析请求体并转换为统一格式，根据 stream 参数决定返回流式或非流式响应。
// 非流式错误通过 HTTP JSON 返回；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE typed event response.error 返回。
//
// @Summary      Responses
// @Description  创建 Responses API 响应：非流式返回 JSON；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE typed event response.error 返回
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
	logCtx := common.NewRequestLogContext(c, "openai", "compat", "responses").
		WithExtra(map[string]string{"protocol_mode": "auto"})
	logger := logCtx.EnrichLogger(h.logger)

	var req openaiResponsesTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("OpenAI Responses 请求参数校验失败", "error", err)
		c.JSON(
			http.StatusBadRequest,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("无效的请求格式：%v", err), http.StatusBadRequest, err),
		)
		return
	}

	if req.Model != nil {
		logCtx = logCtx.WithModel(*req.Model)
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Stream != nil && *req.Stream {
		h.streamOpenAIResponses(c, &req, logCtx, true)
		return
	}

	if h.collector != nil {
		h.collector.IncrementConnection()
		defer h.collector.DecrementConnection()
	}

	ctx := logCtx.WithContext(c.Request.Context())
	resp, err := h.gatewayService.OpenAICompatResponses(ctx, &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "处理请求时出错")
		c.JSON(
			mappedErr.StatusCode,
			common.NewOpenAIHTTPErrorResponse(mappedErr.Message, mappedErr.StatusCode, err, &mappedErr),
		)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) streamOpenAIChat(c *gin.Context, req *openaiChatTypes.Request, logCtx common.RequestLogContext, sendDone bool) {
	streamLogCtx := logCtx.WithExtra(map[string]string{"protocol_mode": "sse", "flow": "stream"})
	ctx := streamLogCtx.WithContext(c.Request.Context())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	resultChan := h.gatewayService.OpenAICompatChatCompletionStreamResult(ctx, req)

	connectionCounted := false
	releaseConnection := func() {
		if h.collector != nil && connectionCounted {
			h.collector.DecrementConnection()
			connectionCounted = false
		}
	}
	if h.collector != nil {
		h.collector.IncrementConnection()
		connectionCounted = true
	}
	defer releaseConnection()

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false
	streamWriterBroken := false
	streamStarted := false

	logger := streamLogCtx.EnrichLogger(h.logger)
	writeChatSSEError := func(message string, status int, err error, protocolErr ...*gateway.DataPlaneError) {
		if streamWriterBroken {
			return
		}

		sendErr := common.WriteOpenAIChatSSEError(c.Writer, message, status, err, protocolErr...)
		if sendErr != nil {
			if common.IsOpenAIStreamWriteError(sendErr) {
				streamWriterBroken = true
				logger.Error("补写 OpenAI Chat 流式错误事件失败，连接已不可恢复",
					"error", sendErr,
					"stream_phase", "writer_failed",
					"error_write", "failed",
				)
				return
			}
			logger.Error("补写 OpenAI Chat 流式错误事件失败", "error", sendErr, "stream_phase", "streaming")
			return
		}

		if flusher != nil {
			flusher.Flush()
		}
	}
	defer func() {
		if r := recover(); r != nil {
			streamFailed = true
			cancel()
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			panicErr := fmt.Errorf("panic: %v", r)
			logger.Error("流式响应处理发生 panic",
				"panic", r,
				"stack", stackLines,
				"stream_phase", "panic",
			)
			if !streamStarted {
				c.JSON(http.StatusInternalServerError, common.NewOpenAIHTTPErrorResponse("服务器内部错误", http.StatusInternalServerError, panicErr))
				return
			}

			if streamWriterBroken {
				return
			}

			writeChatSSEError("服务器内部错误", http.StatusInternalServerError, panicErr)
		}
	}()

	firstResult, ok := <-resultChan
	if !ok {
		return
	}

	if firstResult.ProtocolError != nil && firstResult.ProtocolError.ShouldProxyAsHTTPError {
		c.JSON(
			firstResult.ProtocolError.StatusCode,
			common.NewOpenAIHTTPErrorResponse(firstResult.ProtocolError.Message, firstResult.ProtocolError.StatusCode, nil, firstResult.ProtocolError),
		)
		return
	}

	common.SetBaseSSEHeaders(c)
	streamStarted = true

	writeResult := func(result gateway.OpenAIChatStreamResult) bool {
		if result.ProtocolError != nil {
			streamFailed = true
			cancel()
			writeChatSSEError(result.ProtocolError.Message, result.ProtocolError.StatusCode, nil, result.ProtocolError)
			return true
		}

		if result.Event == nil {
			return false
		}

		data, err := json.Marshal(result.Event)
		if err != nil {
			streamFailed = true
			cancel()
			logger.Error("无法序列化事件", "error", err, "stream_phase", "streaming")
			if streamWriterBroken {
				return true
			}
			writeChatSSEError(fmt.Sprintf("无法序列化事件: %v", err), http.StatusInternalServerError, err)
			return true
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			streamWriterBroken = true
			cancel()
			logger.Error("写入 OpenAI Chat 流式响应失败，连接已不可恢复",
				"error", err,
				"stream_phase", "writer_failed",
			)
			return true
		}

		if flusher != nil {
			flusher.Flush()
		}

		return result.Done || result.Terminal
	}

	if writeResult(firstResult) {
		return
	}

	for result := range resultChan {
		if writeResult(result) {
			break
		}
	}

	if sendDone && !streamFailed && streamStarted {
		if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
			streamWriterBroken = true
			cancel()
			logger.Error("写入 OpenAI Chat 流结束标记失败，连接已不可恢复",
				"error", err,
				"stream_phase", "writer_failed",
			)
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func (h *Handler) streamOpenAIResponses(c *gin.Context, req *openaiResponsesTypes.Request, logCtx common.RequestLogContext, sendDone bool) {
	streamLogCtx := logCtx.WithExtra(map[string]string{"protocol_mode": "sse", "flow": "stream"})
	ctx := streamLogCtx.WithContext(c.Request.Context())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	resultChan := h.gatewayService.OpenAICompatResponsesStreamResult(ctx, req)

	connectionCounted := false
	releaseConnection := func() {
		if h.collector != nil && connectionCounted {
			h.collector.DecrementConnection()
			connectionCounted = false
		}
	}
	if h.collector != nil {
		h.collector.IncrementConnection()
		connectionCounted = true
	}
	defer releaseConnection()

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false
	streamWriterBroken := false
	streamStarted := false

	logger := streamLogCtx.EnrichLogger(h.logger)
	writeResponsesSSEError := func(message string, status int, err error, protocolErr ...*gateway.DataPlaneError) {
		if streamWriterBroken {
			return
		}

		sendErr := common.WriteOpenAIResponsesTypedEventError(c.Writer, message, status, err, protocolErr...)
		if sendErr != nil {
			if common.IsOpenAIStreamWriteError(sendErr) {
				streamWriterBroken = true
				logger.Error("补写 OpenAI Responses 流式错误事件失败，连接已不可恢复",
					"error", sendErr,
					"stream_phase", "writer_failed",
					"error_write", "failed",
				)
				return
			}
			logger.Error("补写 OpenAI Responses 流式错误事件失败", "error", sendErr, "stream_phase", "streaming")
			return
		}

		if flusher != nil {
			flusher.Flush()
		}
	}
	defer func() {
		if r := recover(); r != nil {
			streamFailed = true
			cancel()
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			panicErr := fmt.Errorf("panic: %v", r)
			logger.Error("流式响应处理发生 panic",
				"panic", r,
				"stack", stackLines,
				"stream_phase", "panic",
			)
			if !streamStarted {
				c.JSON(http.StatusInternalServerError, common.NewOpenAIHTTPErrorResponse("服务器内部错误", http.StatusInternalServerError, panicErr))
				return
			}

			if streamWriterBroken {
				return
			}

			writeResponsesSSEError("服务器内部错误", http.StatusInternalServerError, panicErr)
		}
	}()

	firstResult, ok := <-resultChan
	if !ok {
		return
	}

	if firstResult.ProtocolError != nil && firstResult.ProtocolError.ShouldProxyAsHTTPError {
		c.JSON(
			firstResult.ProtocolError.StatusCode,
			common.NewOpenAIHTTPErrorResponse(firstResult.ProtocolError.Message, firstResult.ProtocolError.StatusCode, nil, firstResult.ProtocolError),
		)
		return
	}

	common.SetBaseSSEHeaders(c)
	streamStarted = true

	writeResult := func(result gateway.OpenAIResponsesStreamResult) bool {
		if result.ProtocolError != nil {
			streamFailed = true
			cancel()
			writeResponsesSSEError(result.ProtocolError.Message, result.ProtocolError.StatusCode, nil, result.ProtocolError)
			return true
		}

		if result.Event == nil {
			return false
		}

		data, err := json.Marshal(result.Event)
		if err != nil {
			streamFailed = true
			cancel()
			logger.Error("无法序列化事件", "error", err, "stream_phase", "streaming")
			if streamWriterBroken {
				return true
			}
			writeResponsesSSEError(fmt.Sprintf("无法序列化事件: %v", err), http.StatusInternalServerError, err)
			return true
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			streamWriterBroken = true
			cancel()
			logger.Error("写入 OpenAI Responses 流式响应失败，连接已不可恢复",
				"error", err,
				"stream_phase", "writer_failed",
			)
			return true
		}

		if flusher != nil {
			flusher.Flush()
		}

		return result.Done || result.Terminal
	}

	if writeResult(firstResult) {
		return
	}

	for result := range resultChan {
		if writeResult(result) {
			break
		}
	}

	if sendDone && !streamFailed && streamStarted {
		if _, err := fmt.Fprintf(c.Writer, "data: [DONE]\n\n"); err != nil {
			streamWriterBroken = true
			cancel()
			logger.Error("写入 OpenAI Responses 流结束标记失败，连接已不可恢复",
				"error", err,
				"stream_phase", "writer_failed",
			)
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}
