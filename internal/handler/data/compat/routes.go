package multi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	"github.com/MeowSalty/pinai/internal/app/stats"
	"github.com/MeowSalty/pinai/internal/handler/data/auth"
	"github.com/MeowSalty/pinai/internal/handler/data/common"
	"github.com/MeowSalty/pinai/internal/handler/data/native"
	"github.com/gin-gonic/gin"
)

// SetupMultiRoutes 注册 multi 兼容路由。
func SetupMultiRoutes(
	rootRouter *gin.RouterGroup,
	gatewayService gateway.Service,
	collector *stats.Collector,
	userAgent string,
	passthroughHeaders bool,
	logger *slog.Logger,
	apiToken string,
) {
	// 配置子路由
	nativeRouter := rootRouter.Group("/native")
	v1Router := rootRouter.Group("/v1")
	v1betaRouter := rootRouter.Group("/v1beta")

	// 创建认证策略注册表
	authRegistry := auth.NewRegistry(apiToken)

	rootRouter.Use(auth.NewProviderMiddleware(authRegistry, apiToken))
	nativeRouter.Use(auth.NewProviderMiddleware(authRegistry, apiToken))

	// 创建 Handler 实例，传入 userAgent 与 headers 透传配置
	handler := New(gatewayService, collector, userAgent, passthroughHeaders, logger)

	// 注册 OpenAI 兼容路由
	v1Router.POST("/chat/completions", handler.ChatCompletions)
	v1Router.POST("/responses", handler.Responses)

	// 注册 Anthropic 兼容路由
	v1Router.POST("/messages", handler.Messages)

	// 注册 Gemini 兼容路由
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
	v1Router.GET("/models", handler.SelectModels())
	v1betaRouter.GET("/models", handler.SelectGeminiModels())

	// 原生请求
	native.SetupNativeRoutes(nativeRouter, gatewayService, collector, userAgent, passthroughHeaders, logger)
}
