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
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	"github.com/gin-gonic/gin"
)

// GeminiGenerateContent 处理 Gemini generateContent 请求，路径为 POST /multi/v1beta/models/{model}:generateContent。
// 解析请求体并从参数或查询字符串中获取模型名称，转换为统一格式后调用网关服务。
// 该接口为非流式请求，错误统一以 HTTP JSON 返回。
//
// @Summary      生成内容
// @Description  调用 Gemini 模型生成内容（非流式）；校验失败、上游协议错误与网关错误均通过 HTTP JSON 返回
// @Tags         Gemini
// @Accept       json
// @Produce      json
// @Param        model    path      string                           true   "模型名称"
// @Param        request  body      geminiTypes.Request  true  "生成内容请求"
// @Success      200      {object}  geminiTypes.Response
// @Failure      400      {object}  geminiTypes.ErrorResponse
// @Failure      401      {object}  geminiTypes.ErrorResponse
// @Failure      500      {object}  geminiTypes.ErrorResponse
// @Router       /multi/v1beta/models/{model}:generateContent [post]
// @Security     ApiKeyAuth
func (h *Handler) GeminiGenerateContent(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "compat", "request_name", "generate_content", "protocol_mode", "json")

	var req geminiTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Gemini generateContent 请求参数校验失败", "error", err)
		common.WriteGeminiJSONError(c, http.StatusBadRequest, fmt.Sprintf("无效的请求体: %v", err), err)
		return
	}

	if req.Model == "" {
		req.Model = strings.TrimSpace(c.GetString("gemini_model"))
	}
	if req.Model == "" {
		req.Model = strings.TrimSpace(c.Query("model"))
	}
	if req.Model == "" {
		logger.Warn("Gemini generateContent 缺少模型参数")
		common.WriteGeminiJSONError(c, http.StatusBadRequest, "缺少模型查询参数", nil)
		return
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	resp, err := h.gatewayService.GeminiCompatGenerateContent(c.Request.Context(), &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "处理请求时出错")
		logger.Warn("Gemini generateContent 请求失败，返回 HTTP JSON 错误",
			"status_code", mappedErr.StatusCode,
			"error_type", mappedErr.ErrorType,
			"error_code", mappedErr.ErrorCode,
		)
		common.WriteGeminiJSONError(c, mappedErr.StatusCode, mappedErr.Message, err, &mappedErr)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GeminiStreamGenerateContent 处理 Gemini streamGenerateContent 请求，路径为 POST /multi/v1beta/models/{model}:streamGenerateContent。
// 解析请求体并从参数或查询字符串中获取模型名称，转换为统一格式后返回流式响应。
// 建流前错误以 HTTP JSON 返回；建流后若出现协议错误或写入错误，将直接终止流，不伪造 JSON error chunk。
//
// @Summary      流式生成内容
// @Description  调用 Gemini 模型流式生成内容；建流前错误返回 HTTP JSON，建流后错误终止流（不写入伪造的 JSON error chunk）
// @Tags         Gemini
// @Accept       json
// @Produce      text/event-stream
// @Param        model    path      string                           true   "模型名称"
// @Param        request  body      geminiTypes.Request  true  "生成内容请求"
// @Success      200      {string}  string
// @Failure      400      {object}  geminiTypes.ErrorResponse
// @Failure      401      {object}  geminiTypes.ErrorResponse
// @Failure      500      {object}  geminiTypes.ErrorResponse
// @Router       /multi/v1beta/models/{model}:streamGenerateContent [post]
// @Security     ApiKeyAuth
func (h *Handler) GeminiStreamGenerateContent(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "compat", "request_name", "stream_generate_content", "protocol_mode", "json")

	var req geminiTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Gemini streamGenerateContent 请求参数校验失败", "error", err)
		common.WriteGeminiJSONError(c, http.StatusBadRequest, fmt.Sprintf("无效的请求体: %v", err), err)
		return
	}

	if req.Model == "" {
		req.Model = strings.TrimSpace(c.GetString("gemini_model"))
	}
	if req.Model == "" {
		req.Model = strings.TrimSpace(c.Query("model"))
	}
	if req.Model == "" {
		logger.Warn("Gemini streamGenerateContent 缺少模型参数")
		common.WriteGeminiJSONError(c, http.StatusBadRequest, "缺少模型查询参数", nil)
		return
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	h.handleGeminiStreamResponse(c, &req)
}

// handleGeminiStreamResponse 处理 Gemini 流式响应。
// 设置 SSE 头部，通过 ChatCompletionStream 获取事件通道，将流式事件转换为 Gemini 格式并写入响应流。
// 包含 panic 恢复机制，发生错误时发送错误事件并记录日志。
func (h *Handler) handleGeminiStreamResponse(c *gin.Context, req *geminiTypes.Request) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	resultChan := h.gatewayService.GeminiCompatGenerateContentStreamResult(ctx, req)

	if h.collector != nil {
		defer h.collector.DecrementConnection()
	}

	flusher, _ := c.Writer.(http.Flusher)
	streamFailed := false
	streamStarted := false
	canWriteErrorChunk := true

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "compat", "request_name", "stream_generate_content", "protocol_mode", "sse", "flow", "stream")
	writeGeminiStreamError := func(message string, status int, err error, protocolErr ...*gateway.DataPlaneError) {
		if !canWriteErrorChunk {
			logger.Error("Gemini 流式连接已不可恢复，跳过错误块补写", "stream_phase", "streaming")
			return
		}
		sendErr := common.WriteGeminiStreamError(c.Writer, message, status, err, protocolErr...)
		if sendErr != nil {
			if common.IsGeminiStreamWriteError(sendErr) {
				canWriteErrorChunk = false
				logger.Error("发送 Gemini 流式错误块失败，连接已不可恢复", "error", sendErr)
				return
			}
			logger.Error("发送 Gemini 流式错误块失败", "error", sendErr)
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
			logger.Error("流式响应处理发生 panic", "panic", r, "stack", stackLines, "stream_phase", "panic")
			if !streamStarted {
				common.WriteGeminiJSONError(c, http.StatusInternalServerError, "服务器内部错误", panicErr)
				return
			}
			writeGeminiStreamError("服务器内部错误", http.StatusInternalServerError, panicErr)
		}
	}()

	firstResult, ok := <-resultChan
	if !ok {
		return
	}

	if firstResult.ProtocolError != nil && firstResult.ProtocolError.ShouldProxyAsHTTPError {
		logger.Warn("Gemini 流式建流前收到可代理 HTTP 协议错误",
			"status_code", firstResult.ProtocolError.StatusCode,
			"error_type", firstResult.ProtocolError.ErrorType,
			"error_code", firstResult.ProtocolError.ErrorCode,
			"stream_phase", "pre_stream",
		)
		common.WriteGeminiJSONError(
			c,
			firstResult.ProtocolError.StatusCode,
			firstResult.ProtocolError.Message,
			nil,
			firstResult.ProtocolError,
		)
		return
	}

	common.SetBaseSSEHeaders(c)
	streamStarted = true

	writeResult := func(result gateway.GeminiStreamResult) bool {
		if result.ProtocolError != nil {
			streamFailed = true
			cancel()
			logger.Warn("Gemini 流中收到协议错误，终止流",
				"status_code", result.ProtocolError.StatusCode,
				"error_type", result.ProtocolError.ErrorType,
				"error_code", result.ProtocolError.ErrorCode,
				"stream_phase", "streaming",
				"terminal", result.Terminal,
				"done", result.Done,
			)
			writeGeminiStreamError(result.ProtocolError.Message, result.ProtocolError.StatusCode, nil, result.ProtocolError)
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
			writeGeminiStreamError(fmt.Sprintf("无法序列化事件: %v", err), http.StatusInternalServerError, err)
			return true
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			canWriteErrorChunk = false
			cancel()
			logger.Error("写入流式响应失败", "error", err, "stream_phase", "streaming")
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

	if flusher != nil && !streamFailed {
		flusher.Flush()
	}
}
