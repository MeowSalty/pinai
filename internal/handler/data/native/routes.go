package native

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	"github.com/MeowSalty/pinai/internal/app/stats"
	"github.com/MeowSalty/pinai/internal/handler/data/common"
	"github.com/gin-gonic/gin"
)

// SetupNativeRoutes 设置多合一原生路由。
func SetupNativeRoutes(
	rootRouter *gin.RouterGroup,
	gatewayService gateway.Service,
	collector *stats.Collector,
	userAgent string,
	passthroughHeaders bool,
	logger *slog.Logger,
) {
	// 配置子路由
	v1Router := rootRouter.Group("/v1")
	v1betaRouter := rootRouter.Group("/v1beta")

	handler := New(gatewayService, collector, userAgent, passthroughHeaders, logger)

	// 注册 OpenAI 原生路由
	v1Router.POST("/chat/completions", handler.OpenAIChatCompletions)
	v1Router.POST("/responses", handler.OpenAIResponses)

	// 注册 Anthropic 原生路由
	v1Router.POST("/messages", handler.AnthropicMessages)

	// 注册 Gemini 原生路由
	v1betaRouter.POST("/models/*action", func(c *gin.Context) {
		action := c.Param("action")
		parts := strings.SplitN(strings.TrimPrefix(action, "/"), ":", 2)
		if len(parts) != 2 {
			common.WriteGeminiJSONError(c, http.StatusBadRequest, "无效的 Gemini 路由格式", nil)
			return
		}
		c.Set("gemini_model", parts[0])
		switch parts[1] {
		case "generateContent":
			handler.GeminiGenerateContent(c)
		case "streamGenerateContent":
			handler.GeminiStreamGenerateContent(c)
		default:
			common.WriteGeminiJSONError(c, http.StatusNotFound, "未知操作", nil)
		}
	})

	// 模型列表
	v1Router.GET("/models", SelectModels())
	v1betaRouter.GET("/models", SelectGeminiModels())
}
