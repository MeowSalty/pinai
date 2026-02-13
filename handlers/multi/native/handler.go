package native

import (
	"log/slog"

	"github.com/MeowSalty/pinai/services/portal"
)

// Handler 处理多平台原生请求
type Handler struct {
	portalService portal.Service
	userAgent     string
	logger        *slog.Logger
}

// New 创建一个新的原生处理器
//
// 参数：
//   - portalService: AI 网关服务实例，用于处理 AI 相关请求
//   - userAgent: User-Agent 配置，空则透传客户端 UA，"default" 使用 fasthttp 默认值，其他字符串则复写
//   - logger: 日志记录器实例
func New(portalService portal.Service, userAgent string, logger *slog.Logger) *Handler {
	return &Handler{
		portalService: portalService,
		userAgent:     userAgent,
		logger:        logger,
	}
}
