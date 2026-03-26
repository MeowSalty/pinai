package multi

import (
	"log/slog"

	"github.com/MeowSalty/pinai/services/gateway"
)

// Handler 统一的多供应商处理器，处理 OpenAI 和 Anthropic 兼容 API 的请求
//
// 该结构体封装了处理多供应商 AI API 请求所需的服务和日志记录器
type Handler struct {
	// gatewayService 网关应用服务实例，作为数据面主业务边界
	gatewayService gateway.Service
	// userAgent User-Agent 配置，用于控制请求的 User-Agent 头部
	userAgent string
	// passthroughHeaders 控制是否透传 HTTP 请求头（过滤后）
	passthroughHeaders bool
	logger             *slog.Logger
}

// New 创建并初始化一个新的多供应商处理器实例
//
// 该函数使用依赖注入的方式创建 Handler 实例
//
// 参数：
//   - gatewayService: 网关应用服务实例，承接数据面应用边界
//   - userAgent: User-Agent 配置，空则透传客户端 UA，"default" 使用 Go net/http 默认值，其他字符串则复写
//   - passthroughHeaders: 是否透传 HTTP 请求头（过滤后）
//   - logger: 日志记录器实例
//
// 返回值：
//   - *Handler: 初始化后的多供应商处理器实例
func New(gatewayService gateway.Service, userAgent string, passthroughHeaders bool, logger *slog.Logger) *Handler {
	return &Handler{
		gatewayService:     gatewayService,
		userAgent:          userAgent,
		passthroughHeaders: passthroughHeaders,
		logger:             logger,
	}
}
