package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"
)

// plainTextHandler 实现普通文本格式的日志处理器
type plainTextHandler struct {
	opts slog.HandlerOptions
	mu   sync.Mutex
	out  io.Writer
}

// newPlainTextHandler 创建普通文本格式的日志处理器
func newPlainTextHandler(out io.Writer, opts *slog.HandlerOptions) *plainTextHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &plainTextHandler{
		opts: *opts,
		out:  out,
	}
}

// Enabled 检查日志级别是否启用
func (h *plainTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// Handle 处理日志记录
func (h *plainTextHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var buf bytes.Buffer

	// 格式：[时间] [级别] [组] 消息
	buf.WriteString(r.Time.Format("2006/01/02 15:04:05.000"))

	// 日志级别
	levelStr := "INFO"
	switch r.Level {
	case slog.LevelDebug:
		levelStr = "DEBUG"
	case slog.LevelInfo:
		levelStr = "INFO"
	case slog.LevelWarn:
		levelStr = "WARN"
	case slog.LevelError:
		levelStr = "ERROR"
	}
	buf.WriteString(" ")
	buf.WriteString(levelStr)
	buf.WriteString(" ")

	// 消息
	buf.WriteString(r.Message)

	// 属性
	r.Attrs(func(a slog.Attr) bool {
		buf.WriteString(" ")
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(a.Value.String())
		return true
	})

	buf.WriteString("\n")

	_, err := h.out.Write(buf.Bytes())
	return err
}

// WithAttrs 返回带有额外属性的处理器
func (h *plainTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &plainTextHandler{
		opts: h.opts,
		out:  h.out,
	}
}

// WithGroup 返回带有组的处理器
func (h *plainTextHandler) WithGroup(name string) slog.Handler {
	return &plainTextHandler{
		opts: h.opts,
		out:  h.out,
	}
}
