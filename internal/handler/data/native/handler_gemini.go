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
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	"github.com/gin-gonic/gin"
)

// GeminiGenerateContent 处理原生 Gemini generateContent 请求，路径为 POST /multi/native/v1beta/models/:model:generateContent。
// 解析请求体，处理 User-Agent 头部，从路径参数或查询参数中获取模型名称。
// 成功时返回 200 和响应数据，错误统一以 HTTP JSON 返回。
//
//	@Summary      生成 Gemini 内容
//	@Description  处理原生 Gemini API 的 generateContent 请求（非流式）；校验失败、上游协议错误与网关错误均通过 HTTP JSON 返回
//	@Tags         native-gemini
//	@Accept       json
//	@Produce      json
//	@Param        model    path      string                  true  "模型名称"
//	@Param        request  body      geminiTypes.Request     true  "请求体"
//	@Success      200      {object}  geminiTypes.Response    "成功"
//	@Failure      400      {object}  geminiTypes.ErrorResponse  "无效的请求体或缺少模型参数"
//	@Failure      500      {object}  geminiTypes.ErrorResponse  "请求失败"
//	@Router       /multi/native/v1beta/models/{model}:generateContent [post]
//	@Security     ApiKeyAuth
func (h *Handler) GeminiGenerateContent(c *gin.Context) {
	logCtx := common.NewRequestLogContext(c, "gemini", "native", "generate_content").
		WithExtra(map[string]string{"protocol_mode": "json"})
	logger := logCtx.EnrichLogger(h.logger)

	var req geminiTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("请求参数绑定失败", "error", err)
		common.WriteGeminiJSONError(c, http.StatusBadRequest, fmt.Sprintf("无效的请求体: %v", err), err)
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Model == "" {
		req.Model = strings.TrimSpace(c.GetString("gemini_model"))
	}
	if req.Model == "" {
		req.Model = strings.TrimSpace(c.Query("model"))
	}
	if req.Model == "" {
		logger.Warn("缺少模型参数", "error", "缺少模型参数")
		common.WriteGeminiJSONError(c, http.StatusBadRequest, "缺少模型查询参数", nil)
		return
	}

	logCtx = logCtx.WithModel(req.Model)

	if h.collector != nil {
		h.collector.IncrementConnection()
		defer h.collector.DecrementConnection()
	}

	ctx := logCtx.WithContext(c.Request.Context())
	resp, err := h.gatewayService.GeminiNativeGenerateContent(ctx, &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "请求失败")
		common.WriteGeminiJSONError(c, mappedErr.StatusCode, mappedErr.Message, err, &mappedErr)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GeminiStreamGenerateContent 处理原生 Gemini streamGenerateContent 请求，路径为 POST /multi/native/v1beta/models/:model:streamGenerateContent。
// 解析请求体，处理 User-Agent 头部，从路径参数或查询参数中获取模型名称，返回流式响应。
// 建流前错误以 HTTP JSON 返回；建流后若出现协议错误或写入错误，将直接终止流，不伪造 JSON error chunk。
//
//	@Summary      流式生成 Gemini 内容
//	@Description  处理原生 Gemini API 的 streamGenerateContent 请求；建流前错误返回 HTTP JSON，建流后错误终止流（不写入伪造的 JSON error chunk）
//	@Tags         native-gemini
//	@Accept       json
//	@Produce      text/event-stream
//	@Param        model    path      string              true  "模型名称"
//	@Param        request  body      geminiTypes.Request true  "请求体"
//	@Success      200      {string}  string              "流式事件数据"
//	@Failure      400      {object}  geminiTypes.ErrorResponse   "无效的请求体或缺少模型参数"
//	@Router       /multi/native/v1beta/models/{model}:streamGenerateContent [post]
//	@Security     ApiKeyAuth
func (h *Handler) GeminiStreamGenerateContent(c *gin.Context) {
	logCtx := common.NewRequestLogContext(c, "gemini", "native", "stream_generate_content").
		WithExtra(map[string]string{"protocol_mode": "json"})
	logger := logCtx.EnrichLogger(h.logger)

	var req geminiTypes.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("请求参数绑定失败", "error", err)
		common.WriteGeminiJSONError(c, http.StatusBadRequest, fmt.Sprintf("无效的请求体: %v", err), err)
		return
	}

	// 处理并透传 HTTP 头部
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	common.ApplyHTTPHeaders(req.Headers, h.userAgent, h.passthroughHeaders, c)

	if req.Model == "" {
		req.Model = strings.TrimSpace(c.GetString("gemini_model"))
	}
	if req.Model == "" {
		req.Model = strings.TrimSpace(c.Query("model"))
	}
	if req.Model == "" {
		logger.Warn("缺少模型参数", "error", "缺少模型参数")
		common.WriteGeminiJSONError(c, http.StatusBadRequest, "缺少模型查询参数", nil)
		return
	}

	logCtx = logCtx.WithModel(req.Model)

	h.streamGemini(c, &req, logCtx)
}

func (h *Handler) streamGemini(c *gin.Context, req *geminiTypes.Request, logCtx common.RequestLogContext) {
	streamLogCtx := logCtx.WithExtra(map[string]string{"protocol_mode": "sse", "flow": "stream"})
	ctx := streamLogCtx.WithContext(c.Request.Context())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	resultChan := h.gatewayService.GeminiNativeGenerateContentStreamResult(ctx, req)

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
	streamStarted := false
	canWriteErrorChunk := true

	logger := streamLogCtx.EnrichLogger(h.logger)
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
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines, "stream_phase", "panic")
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
			logger.Error("序列化流事件失败，终止流", "error", err, "stream_phase", "streaming")
			writeGeminiStreamError(fmt.Sprintf("序列化流事件失败: %v", err), http.StatusInternalServerError, err)
			return true
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			streamFailed = true
			canWriteErrorChunk = false
			cancel()
			logger.Error("写入流事件失败，终止流", "error", err, "stream_phase", "streaming")
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
