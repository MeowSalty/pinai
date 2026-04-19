package native

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

// OpenAIChatCompletions 处理原生 OpenAI 聊天补全请求，路径为 POST /multi/native/v1/chat/completions。
// 解析请求体，处理 User-Agent 头部，根据 stream 参数决定返回流式或非流式响应。
// 非流式错误通过 HTTP JSON 返回；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE error 事件返回。
//
//	@Summary      OpenAI 聊天补全
//	@Description  处理原生 OpenAI API 的 chat completions 请求：非流式返回 JSON；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE error 事件返回
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
	logCtx := common.NewRequestLogContext(c, "openai", "native", "chat_completions").
		WithExtra(map[string]string{"protocol_mode": "auto"})
	logger := logCtx.EnrichLogger(h.logger)

	var req openaiChatTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("解析 OpenAI Chat 请求体失败", "error", err)
		c.JSON(
			http.StatusBadRequest,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("无效的请求体: %v", err), http.StatusBadRequest, err),
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
		h.streamOpenAIChat(c, &req, logCtx, true)
		return
	}

	ctx := logCtx.WithContext(c.Request.Context())
	resp, err := h.gatewayService.OpenAINativeChatCompletion(ctx, &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "请求失败")
		c.JSON(
			mappedErr.StatusCode,
			common.NewOpenAIHTTPErrorResponse(mappedErr.Message, mappedErr.StatusCode, err, &mappedErr),
		)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// OpenAIResponses 处理原生 OpenAI 响应请求，路径为 POST /multi/native/v1/responses。
// 解析请求体，处理 User-Agent 头部，根据 stream 参数决定返回流式或非流式响应。
// 非流式错误通过 HTTP JSON 返回；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE typed event response.error 返回。
//
//	@Summary      OpenAI 响应
//	@Description  处理原生 OpenAI API 的 responses 请求：非流式返回 JSON；流式模式下建流前错误返回 HTTP JSON，建流后错误通过 SSE typed event response.error 返回
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
	logCtx := common.NewRequestLogContext(c, "openai", "native", "responses").
		WithExtra(map[string]string{"protocol_mode": "auto"})
	logger := logCtx.EnrichLogger(h.logger)

	var req openaiResponsesTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("解析 OpenAI Responses 请求体失败", "error", err)
		c.JSON(
			http.StatusBadRequest,
			common.NewOpenAIHTTPErrorResponse(fmt.Sprintf("无效的请求体: %v", err), http.StatusBadRequest, err),
		)
		return
	}

	if req.Model != nil {
		logCtx = logCtx.WithModel(*req.Model)
	}

	// 处理并透传 HTTP 头部
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
	resp, err := h.gatewayService.OpenAINativeResponses(ctx, &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "请求失败")
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
	resultChan := h.gatewayService.OpenAINativeChatCompletionStreamResult(ctx, req)

	connectionReleased := false
	releaseConnection := func() {
		if h.collector == nil || connectionReleased {
			return
		}
		h.collector.DecrementConnection()
		connectionReleased = true
	}
	if h.collector != nil {
		h.collector.IncrementConnection()
	}
	defer releaseConnection()

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false
	streamWriterBroken := false
	streamStarted := false

	logger := streamLogCtx.EnrichLogger(h.logger)
	writeChatSSEError := func(message string, status int, err error, protocolErr ...*gateway.DataPlaneError) {
		if streamWriterBroken {
			logger.Error("OpenAI Chat 原生流式连接已不可恢复，跳过补写错误事件",
				"stream_phase", "writer_failed",
				"error_write", "skipped",
			)
			return
		}

		sendErr := common.WriteOpenAIChatSSEError(c.Writer, message, status, err, protocolErr...)
		if sendErr != nil {
			if common.IsOpenAIStreamWriteError(sendErr) {
				streamWriterBroken = true
				logger.Error("补写 OpenAI Chat 原生流式错误事件失败，连接已不可恢复",
					"error", sendErr,
					"stream_phase", "writer_failed",
					"error_write", "failed",
				)
				return
			}
			logger.Error("补写 OpenAI Chat 原生流式错误事件失败", "error", sendErr, "stream_phase", "streaming")
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
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines, "stream_phase", "panic")
			if !streamStarted {
				c.JSON(http.StatusInternalServerError, common.NewOpenAIHTTPErrorResponse("服务器内部错误", http.StatusInternalServerError, panicErr))
				return
			}

			if streamWriterBroken {
				logger.Error("panic 后 OpenAI Chat 原生流式连接已不可恢复，跳过补写错误事件",
					"stream_phase", "panic",
					"error_write", "skipped",
				)
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
			logger.Error("序列化流事件失败", "error", err, "stream_phase", "streaming")
			if streamWriterBroken {
				logger.Error("OpenAI Chat 原生流式连接已不可恢复，序列化失败后跳过补写错误事件",
					"stream_phase", "writer_failed",
					"error_write", "skipped",
				)
				return true
			}
			writeChatSSEError(fmt.Sprintf("序列化流事件失败: %v", err), http.StatusInternalServerError, err)
			return true
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			streamWriterBroken = true
			cancel()
			logger.Error("写入 OpenAI Chat 原生流式响应失败，连接已不可恢复",
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
			logger.Error("写入 OpenAI Chat 原生流结束标识失败，连接已不可恢复",
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
	resultChan := h.gatewayService.OpenAINativeResponsesStreamResult(ctx, req)

	connectionReleased := false
	releaseConnection := func() {
		if h.collector == nil || connectionReleased {
			return
		}
		h.collector.DecrementConnection()
		connectionReleased = true
	}
	if h.collector != nil {
		h.collector.IncrementConnection()
	}
	defer releaseConnection()

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false
	streamWriterBroken := false
	streamStarted := false

	logger := streamLogCtx.EnrichLogger(h.logger)
	writeResponsesSSEError := func(message string, status int, err error, protocolErr ...*gateway.DataPlaneError) {
		if streamWriterBroken {
			logger.Error("OpenAI Responses 原生流式连接已不可恢复，跳过补写错误事件",
				"stream_phase", "writer_failed",
				"error_write", "skipped",
			)
			return
		}

		sendErr := common.WriteOpenAIResponsesTypedEventError(c.Writer, message, status, err, protocolErr...)
		if sendErr != nil {
			if common.IsOpenAIStreamWriteError(sendErr) {
				streamWriterBroken = true
				logger.Error("补写 OpenAI Responses 原生流式错误事件失败，连接已不可恢复",
					"error", sendErr,
					"stream_phase", "writer_failed",
					"error_write", "failed",
				)
				return
			}
			logger.Error("补写 OpenAI Responses 原生流式错误事件失败", "error", sendErr, "stream_phase", "streaming")
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
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines, "stream_phase", "panic")
			if !streamStarted {
				c.JSON(http.StatusInternalServerError, common.NewOpenAIHTTPErrorResponse("服务器内部错误", http.StatusInternalServerError, panicErr))
				return
			}

			if streamWriterBroken {
				logger.Error("panic 后 OpenAI Responses 原生流式连接已不可恢复，跳过补写错误事件",
					"stream_phase", "panic",
					"error_write", "skipped",
				)
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
			logger.Error("序列化流事件失败", "error", err, "stream_phase", "streaming")
			if streamWriterBroken {
				logger.Error("OpenAI Responses 原生流式连接已不可恢复，序列化失败后跳过补写错误事件",
					"stream_phase", "writer_failed",
					"error_write", "skipped",
				)
				return true
			}
			writeResponsesSSEError(fmt.Sprintf("序列化流事件失败: %v", err), http.StatusInternalServerError, err)
			return true
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			streamWriterBroken = true
			cancel()
			logger.Error("写入 OpenAI Responses 原生流式响应失败，连接已不可恢复",
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
			logger.Error("写入 OpenAI Responses 原生流结束标识失败，连接已不可恢复",
				"error", err,
				"stream_phase", "writer_failed",
			)
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}
