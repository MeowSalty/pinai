package native

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/handlers/multi/common"
	"github.com/MeowSalty/pinai/services/stats"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	"github.com/gin-gonic/gin"
)

// GeminiGenerateContent 处理原生 Gemini generateContent 请求，路径为 POST /multi/native/v1beta/models/:model:generateContent。
// 解析请求体，处理 User-Agent 头部，从路径参数或查询参数中获取模型名称。
// 成功时返回 200 和响应数据，失败时返回 400 或 500 错误。
//
//	@Summary      生成 Gemini 内容
//	@Description  处理原生 Gemini API 的 generateContent 请求，非流式模式
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
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "native")
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

	resp, err := h.gatewayService.GeminiNativeGenerateContent(c.Request.Context(), &req)
	if err != nil {
		mappedErr := h.gatewayService.MapDataPlaneError(err, "请求失败")
		common.WriteGeminiJSONError(c, mappedErr.StatusCode, mappedErr.Message, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GeminiStreamGenerateContent 处理原生 Gemini streamGenerateContent 请求，路径为 POST /multi/native/v1beta/models/:model:streamGenerateContent。
// 解析请求体，处理 User-Agent 头部，从路径参数或查询参数中获取模型名称，返回流式响应。
// 成功时返回 200 和流式事件数据，失败时返回 400 错误。
//
//	@Summary      流式生成 Gemini 内容
//	@Description  处理原生 Gemini API 的 streamGenerateContent 请求，流式模式
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
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "native")
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

	h.streamGemini(c, &req)
}

func (h *Handler) streamGemini(c *gin.Context, req *geminiTypes.Request) {
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	resultChan := h.gatewayService.GeminiNativeGenerateContentStreamResult(ctx, req)

	collector := stats.GetCollector()
	defer collector.DecrementConnection()

	flusher, _ := c.Writer.(http.Flusher)

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "native", "flow", "stream")
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("原生流处理异常", "panic", r, "stack", stackLines)
		}
	}()

	for result := range resultChan {
		if result.Event == nil {
			continue
		}

		if result.ErrorMessage != "" {
			logger.Warn("上游返回 Gemini 流式错误事件", "error_message", result.ErrorMessage)
			errorData, marshalErr := json.Marshal(common.NewGeminiErrorResponse(result.ErrorMessage, http.StatusInternalServerError, fmt.Errorf("%s", result.ErrorMessage)))
			if marshalErr != nil {
				cancel()
				logger.Error("序列化 Gemini 标准错误事件失败", "error", marshalErr)
				break
			}

			if _, writeErr := fmt.Fprintf(c.Writer, "data: %s\n\n", errorData); writeErr != nil {
				cancel()
				logger.Error("写入 Gemini 标准错误事件失败", "error", writeErr)
				break
			}

			flusher.Flush()
			break
		}

		data, err := json.Marshal(result.Event)
		if err != nil {
			cancel()
			logger.Error("序列化流事件失败", "error", err)
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			cancel()
			logger.Error("写入流事件失败", "error", err)
			break
		}

		flusher.Flush()

		if result.Done {
			break
		}
	}
}
