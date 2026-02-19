package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

// dailyRotateHandler 实现按日期分割的日志文件处理器
type dailyRotateHandler struct {
	logDir      string
	baseName    string
	level       slog.Level
	mu          sync.Mutex
	currentDate string
	file        *os.File
	handler     slog.Handler
}

// newDailyRotateHandler 创建按日期分割的日志文件处理器
func newDailyRotateHandler(logDir, baseName string, level slog.Level) (*dailyRotateHandler, error) {
	h := &dailyRotateHandler{
		logDir:   logDir,
		baseName: baseName,
		level:    level,
	}
	if err := h.rotate(); err != nil {
		return nil, err
	}
	return h, nil
}

// rotate 检查并轮转日志文件
func (h *dailyRotateHandler) rotate() error {
	currentDate := time.Now().Format("2006-01-02")

	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果日期未变化，无需轮转
	if h.currentDate == currentDate && h.file != nil {
		return nil
	}

	// 关闭旧文件
	if h.file != nil {
		h.file.Close()
	}

	// 创建新文件
	fileName := h.baseName + "-" + currentDate + ".log"
	filePath := h.logDir + "/" + fileName
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	h.currentDate = currentDate
	h.file = file
	h.handler = slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: h.level,
	})

	return nil
}

// Enabled 检查日志级别是否启用
func (h *dailyRotateHandler) Enabled(ctx context.Context, level slog.Level) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler == nil {
		return false
	}
	return h.handler.Enabled(ctx, level)
}

// Handle 处理日志记录
func (h *dailyRotateHandler) Handle(ctx context.Context, r slog.Record) error {
	// 检查是否需要轮转
	if err := h.rotate(); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler == nil {
		return nil
	}
	return h.handler.Handle(ctx, r)
}

// WithAttrs 返回带有额外属性的处理器
func (h *dailyRotateHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler == nil {
		return h
	}
	return &dailyRotateHandler{
		logDir:      h.logDir,
		baseName:    h.baseName,
		level:       h.level,
		currentDate: h.currentDate,
		file:        h.file,
		handler:     h.handler.WithAttrs(attrs),
	}
}

// WithGroup 返回带有组的处理器
func (h *dailyRotateHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.handler == nil {
		return h
	}
	return &dailyRotateHandler{
		logDir:      h.logDir,
		baseName:    h.baseName,
		level:       h.level,
		currentDate: h.currentDate,
		file:        h.file,
		handler:     h.handler.WithGroup(name),
	}
}

// Close 关闭日志文件
func (h *dailyRotateHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.file != nil {
		return h.file.Close()
	}
	return nil
}
