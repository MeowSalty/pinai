package router

import (
	statsService "github.com/MeowSalty/pinai/services/stats"
	"github.com/gin-gonic/gin"
)

// createStatsCollectorMiddleware 创建统计数据采集中间件
//
// 该中间件用于采集业务接口的请求数据和活动连接数
//
// 注意：
//   - 对于非流式响应，在请求完成后自动减少连接数
//   - 对于流式响应（SSE），连接数由流式处理器在流结束时通过 defer collector.DecrementConnection() 减少
func createStatsCollectorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		collector := statsService.GetCollector()

		// 记录请求
		collector.RecordRequest()

		// 增加活动连接数
		collector.IncrementConnection()

		// 对于非流式响应，请求完成后减少活动连接数
		// 流式响应会在流式 handler 中通过 defer collector.DecrementConnection() 处理
		defer func() {
			// 检查是否为流式响应（通过响应头判断）
			contentType := c.Writer.Header().Get("Content-Type")
			if contentType != "text/event-stream" {
				// 非流式响应，在这里减少连接数
				collector.DecrementConnection()
			}
			// 流式响应的连接数将在流结束时由 handler 中的 defer 减少
		}()

		c.Next()
	}
}
