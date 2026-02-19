package logger

import (
	"log/slog"
	"os"
	"strings"
)

// InitLogger 初始化日志记录器
//
// 返回主日志记录器和按日期分割的日志处理器（用于关闭）
func InitLogger(logLevel string) (*slog.Logger, *dailyRotateHandler) {
	// 解析日志等级
	var level slog.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// 创建 log 目录
	if err := os.MkdirAll("log", 0755); err != nil {
		// 创建临时日志记录器用于输出错误
		tempLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		}))
		tempLogger.Error("创建日志目录失败", "error", err)
	}

	// 创建终端处理器（普通文本格式）
	consoleHandler := newPlainTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	// 创建按日期分割的日志文件处理器（JSON 格式）
	fileHandler, err := newDailyRotateHandler("log", "pinai", level)
	if err != nil {
		// 如果无法创建日志文件，仅使用终端输出
		tempLogger := slog.New(consoleHandler)
		tempLogger.Error("无法创建日志文件，将仅输出到终端", "error", err)
		fileHandler = nil
	}

	// 创建多输出日志记录器
	var logger *slog.Logger
	if fileHandler != nil {
		logger = slog.New(newMultiHandler(consoleHandler, fileHandler))
	} else {
		logger = slog.New(consoleHandler)
	}

	return logger, fileHandler
}
