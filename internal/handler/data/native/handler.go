package native

import (
	"log/slog"

	"github.com/MeowSalty/pinai/internal/app/gateway"
	"github.com/MeowSalty/pinai/internal/app/stats"
)

// Handler 处理多平台原生请求
type Handler struct {
	gatewayService     gateway.Service
	collector          *stats.Collector
	userAgent          string
	passthroughHeaders bool
	logger             *slog.Logger
}

// New 创建一个新的原生处理器
//
// 参数：
//   - gatewayService: 网关应用服务实例，承接数据面应用边界
//   - collector: 统计采集器，用于处理流式连接计数
//   - userAgent: User-Agent 配置，空则透传客户端 UA，"default" 使用 Go net/http 默认值，其他字符串则复写
//   - passthroughHeaders: 是否透传 HTTP 请求头（过滤后）
//   - logger: 日志记录器实例
func New(gatewayService gateway.Service, collector *stats.Collector, userAgent string, passthroughHeaders bool, logger *slog.Logger) *Handler {
	return &Handler{
		gatewayService:     gatewayService,
		collector:          collector,
		userAgent:          userAgent,
		passthroughHeaders: passthroughHeaders,
		logger:             logger,
	}
}
