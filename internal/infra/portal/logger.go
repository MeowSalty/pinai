package portal

import (
	"log/slog"

	logadapterpkg "github.com/MeowSalty/pinai/internal/infra/portal/logadapter"
	"github.com/MeowSalty/portal/logger"
)

// NewSlogAdapter 创建一个新的 slog 适配器
//
// 参数：
//   - logger: slog.Logger 实例
//
// 返回值：
//   - logger.Logger: 实现了 portal.logger.Logger 接口的适配器
func NewSlogAdapter(logger *slog.Logger) logger.Logger {
	return logadapterpkg.New(logger)
}
