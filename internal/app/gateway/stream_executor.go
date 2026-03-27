package gateway

import "log/slog"

type streamLogContext struct {
	logger *slog.Logger
	attrs  []any
}

func newStreamLogContext(baseLogger *slog.Logger, loggerGroup, requestName, modelName string) streamLogContext {
	return streamLogContext{
		logger: baseLogger.WithGroup(loggerGroup),
		attrs:  []any{"request_name", requestName, "model", modelName},
	}
}

func startStream[T any](streamCtx streamLogContext, invoker func() <-chan T) <-chan T {
	streamCtx.logger.Info("开始执行流式请求", streamCtx.attrs...)
	stream := invoker()
	streamCtx.logger.Info("流式请求已启动", streamCtx.attrs...)
	return stream
}
