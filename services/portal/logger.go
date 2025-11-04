package portal

import (
	"context"
	"log/slog"

	"github.com/MeowSalty/portal/logger"
)

// slogAdapter 将 log/slog.Logger 适配到 portal.logger.Logger 接口
//
// 该适配器实现了 portal 包所需的日志接口，使得可以使用标准库的 slog 作为日志记录器。
type slogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter 创建一个新的 slog 适配器
//
// 参数：
//   - logger: slog.Logger 实例
//
// 返回值：
//   - logger.Logger: 实现了 portal.logger.Logger 接口的适配器
func NewSlogAdapter(logger *slog.Logger) logger.Logger {
	return &slogAdapter{logger: logger}
}

// Debug 记录调试级别日志
func (s *slogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

// DebugContext 记录带上下文的调试级别日志
func (s *slogAdapter) DebugContext(ctx context.Context, msg string, args ...any) {
	s.logger.DebugContext(ctx, msg, args...)
}

// Info 记录信息级别日志
func (s *slogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// InfoContext 记录带上下文的信息级别日志
func (s *slogAdapter) InfoContext(ctx context.Context, msg string, args ...any) {
	s.logger.InfoContext(ctx, msg, args...)
}

// Warn 记录警告级别日志
func (s *slogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

// WarnContext 记录带上下文的警告级别日志
func (s *slogAdapter) WarnContext(ctx context.Context, msg string, args ...any) {
	s.logger.WarnContext(ctx, msg, args...)
}

// Error 记录错误级别日志
func (s *slogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

// ErrorContext 记录带上下文的错误级别日志
func (s *slogAdapter) ErrorContext(ctx context.Context, msg string, args ...any) {
	s.logger.ErrorContext(ctx, msg, args...)
}

// With 返回一个新的 Logger，包含指定的属性
//
// args 参数应该是成对的键值对，例如：
//
//	logger.With("key1", "value1", "key2", 123)
func (s *slogAdapter) With(args ...any) logger.Logger {
	return &slogAdapter{logger: s.logger.With(args...)}
}

// WithGroup 返回一个新的 Logger，后续日志将属于指定的组
func (s *slogAdapter) WithGroup(name string) logger.Logger {
	return &slogAdapter{logger: s.logger.WithGroup(name)}
}
