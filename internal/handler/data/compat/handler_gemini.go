package multi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MeowSalty/pinai/internal/handler/data/common"
	geminiTypes "github.com/MeowSalty/portal/request/adapter/gemini/types"
	"github.com/gin-gonic/gin"
)

// GeminiGenerateContent 处理 Gemini generateContent 请求，路径为 POST /multi/v1beta/models/{model}:generateContent。
// 解析请求体并从参数或查询字符串中获取模型名称，转换为统一格式后调用 ChatCompletion 服务。
// 成功时返回 200 状态码及生成内容响应，失败时返回 400/500 状态码。
//
// @Summary      生成内容
// @Description  调用 Gemini 模型生成内容，支持通过路径参数或查询参数指定模型
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
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "compat")

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
		common.WriteGeminiJSONError(c, mappedErr.StatusCode, mappedErr.Message, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GeminiStreamGenerateContent 处理 Gemini streamGenerateContent 请求，路径为 POST /multi/v1beta/models/{model}:streamGenerateContent。
// 解析请求体并从参数或查询字符串中获取模型名称，转换为统一格式后返回流式响应。
// 成功时返回流式 SSE 响应，失败时返回 400/500 状态码。
//
// @Summary      流式生成内容
// @Description  调用 Gemini 模型流式生成内容，支持通过路径参数或查询参数指定模型
// @Tags         Gemini
// @Accept       json
// @Produce      text/event-stream
// @Param        model    path      string                           true   "模型名称"
// @Param        request  body      geminiTypes.Request  true  "生成内容请求"
// @Success      200      {object}  geminiTypes.Candidate
// @Failure      400      {object}  geminiTypes.ErrorResponse
// @Failure      401      {object}  geminiTypes.ErrorResponse
// @Failure      500      {object}  geminiTypes.ErrorResponse
// @Router       /multi/v1beta/models/{model}:streamGenerateContent [post]
// @Security     ApiKeyAuth
func (h *Handler) GeminiStreamGenerateContent(c *gin.Context) {
	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "compat")

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
	common.SetBaseSSEHeaders(c)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	resultChan := h.gatewayService.GeminiCompatGenerateContentStreamResult(ctx, req)

	if h.collector != nil {
		defer h.collector.DecrementConnection()
	}

	flusher, _ := c.Writer.(http.Flusher)

	logger := h.logger.With("path", c.Request.URL.Path, "method", c.Request.Method, "provider", "gemini", "api_style", "compat", "flow", "stream")
	defer func() {
		if r := recover(); r != nil {
			cancel()
			stack := debug.Stack()
			stackLines := strings.Split(strings.TrimSpace(string(stack)), "\n")
			logger.Error("流式响应处理发生 panic", "panic", r, "stack", stackLines)
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
				logger.Error("无法序列化 Gemini 标准错误事件", "error", marshalErr)
				break
			}

			if _, writeErr := fmt.Fprintf(c.Writer, "data: %s\n\n", errorData); writeErr != nil {
				cancel()
				logger.Error("写入 Gemini 标准错误事件失败", "error", writeErr)
				break
			}

			if flusher != nil {
				flusher.Flush()
			}

			if result.Done {
				break
			}
			continue
		}

		data, err := json.Marshal(result.Event)
		if err != nil {
			cancel()
			logger.Error("无法序列化事件", "error", err)
			break
		}

		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			cancel()
			logger.Error("写入流式响应失败", "error", err)
			break
		}

		if flusher != nil {
			flusher.Flush()
		}

		if result.Done {
			break
		}
	}

	if flusher != nil {
		flusher.Flush()
	}
}
