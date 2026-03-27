package router

import (
	"log/slog"

	appbootstrap "github.com/MeowSalty/pinai/internal/bootstrap"
	multi "github.com/MeowSalty/pinai/internal/handler/data/compat"
	statsService "github.com/MeowSalty/pinai/services/stats"
	"github.com/gin-gonic/gin"
)

// DataPlaneConfig 定义数据面路由所需最小配置。
type DataPlaneConfig struct {
	ApiToken           string
	UserAgent          string
	PassthroughHeaders bool
}

// SetupDataPlaneRoutes 装配数据面路由与相关中间件。
func SetupDataPlaneRoutes(web *gin.Engine, svcs *appbootstrap.Services, config DataPlaneConfig, logger *slog.Logger) {
	multiAPI := web.Group("/multi")

	// 为业务 API 添加统计采集中间件
	multiAPI.Use(createStatsCollectorMiddleware(svcs.StatsCollector))

	multi.SetupMultiRoutes(multiAPI, svcs.GatewayService, config.UserAgent, config.PassthroughHeaders, logger, config.ApiToken)
}

// createStatsCollectorMiddleware 创建统计数据采集中间件。
func createStatsCollectorMiddleware(collector *statsService.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		activeCollector := collector
		if activeCollector == nil {
			// 兼容旧调用链：未显式注入时回退到全局采集器。
			activeCollector = statsService.GetCollector()
		}

		activeCollector.RecordRequest()
		activeCollector.IncrementConnection()

		defer func() {
			contentType := c.Writer.Header().Get("Content-Type")
			if contentType != "text/event-stream" {
				activeCollector.DecrementConnection()
			}
		}()

		c.Next()
	}
}
