package router

import (
	"log/slog"

	"github.com/MeowSalty/pinai/handlers/multi"
	"github.com/MeowSalty/pinai/services"
	"github.com/gin-gonic/gin"
)

// setupDataPlaneRoutes 装配数据面路由与相关中间件。
func setupDataPlaneRoutes(web *gin.Engine, svcs *services.Services, config Config, logger *slog.Logger) {
	multiAPI := web.Group("/multi")

	// 为业务 API 添加统计采集中间件
	multiAPI.Use(createStatsCollectorMiddleware())

	multi.SetupMultiRoutes(multiAPI, svcs.GatewayService, svcs.PortalService, config.UserAgent, config.PassthroughHeaders, logger, config.ApiToken)
}
