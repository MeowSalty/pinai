package router

import (
	"log/slog"

	appbootstrap "github.com/MeowSalty/pinai/internal/bootstrap"
	multi "github.com/MeowSalty/pinai/internal/handler/data/compat"
	"github.com/MeowSalty/pinai/internal/app/stats"
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

	multi.SetupMultiRoutes(multiAPI, svcs.GatewayService, svcs.StatsCollector, config.UserAgent, config.PassthroughHeaders, logger, config.ApiToken)
}

// createStatsCollectorMiddleware 创建统计数据采集中间件。
func createStatsCollectorMiddleware(collector *stats.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		if collector == nil {
			// 主路径要求显式注入采集器；缺失时直接跳过采集，避免回退全局状态。
			c.Next()
			return
		}

		collector.RecordRequest()
		collector.IncrementConnection()

		defer func() {
			contentType := c.Writer.Header().Get("Content-Type")
			if contentType != "text/event-stream" {
				collector.DecrementConnection()
			}
		}()

		c.Next()
	}
}
