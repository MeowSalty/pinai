package gateway

import (
	"context"
	"log/slog"
)

// logCtxAttrsFromContext 从 context.Context 提取日志上下文属性。
//
// 由 common 包在 init 阶段通过 RegisterLogCtxAttrsFromContext 注入实际实现，
// 以打破 gateway 与 handler/data/common 之间的循环依赖。
// 未注册时返回 nil，不影响现有日志行为。
var logCtxAttrsFromContext = func(ctx context.Context) []any { return nil }

// RegisterLogCtxAttrsFromContext 注册从 context.Context 提取日志上下文属性的函数。
//
// 由 common 包在 init 阶段调用，将 common.FromContext + SlogAttrs 桥接到 gateway 侧，
// 使 gateway 的流执行器能够读取 Handler 透传的统一日志字段（request_id、client_ip 等）。
func RegisterLogCtxAttrsFromContext(fn func(context.Context) []any) {
	if fn != nil {
		logCtxAttrsFromContext = fn
	}
}

// streamLogContext 承载流式请求的日志上下文。
//
// logger 已附加 WithGroup(loggerGroup) 以及从 context.Context 透传的统一日志字段；
// attrs 保留 gateway 侧的 request_name 和 model 字段，在每条日志中显式传递。
type streamLogContext struct {
	logger *slog.Logger
	attrs  []any
}

// newStreamLogContext 创建流式请求的日志上下文。
//
// 从 ctx 中读取 Handler 透传的统一日志上下文字段（request_id、client_ip、user_agent、
// path、method、provider、api_style 等），将其附加到 logger 上；
// 同时保留 gateway 侧的 request_name 和 model 字段，避免上下文字段退化。
// request_name 和 model 从上下文字段中过滤，防止与 gateway 侧 attrs 重复。
func newStreamLogContext(ctx context.Context, baseLogger *slog.Logger, loggerGroup, requestName, modelName string) streamLogContext {
	logger := baseLogger.WithGroup(loggerGroup)

	// 从 context.Context 读取 Handler 透传的统一日志上下文
	ctxAttrs := logCtxAttrsFromContext(ctx)
	if len(ctxAttrs) > 0 {
		// 过滤掉 gateway 侧已通过 attrs 管理的 request_name 和 model，防止重复
		filtered := make([]any, 0, len(ctxAttrs))
		for i := 0; i+1 < len(ctxAttrs); i += 2 {
			if key, isStr := ctxAttrs[i].(string); isStr {
				if key == "request_name" || key == "model" {
					continue
				}
				filtered = append(filtered, ctxAttrs[i], ctxAttrs[i+1])
			}
		}
		if len(filtered) > 0 {
			logger = logger.With(filtered...)
		}
	}

	return streamLogContext{
		logger: logger,
		attrs:  []any{"request_name", requestName, "model", modelName},
	}
}

// startStream 启动流式请求并返回事件通道。
//
// 将原先"开始执行流式请求"与"流式请求已启动"两条 INFO 合并为单条高价值日志，
// 避免流开始阶段产生重复日志。
func startStream[T any](streamCtx streamLogContext, invoker func() <-chan T) <-chan T {
	streamCtx.logger.Info("开始执行流式请求", streamCtx.attrs...)
	stream := invoker()
	return stream
}

// logStreamComplete 记录流式请求正常完成的单条 INFO 日志。
//
// 替代原先"结束条件满足"与"上游通道关闭"双记的模式，
// 通过 reason 区分完成原因（"done" 表示收到终止事件，"channel_closed" 表示上游通道关闭），
// 避免正常结束时产生两条 INFO 日志。
func logStreamComplete(streamCtx streamLogContext, reason string, extraAttrs ...any) {
	attrs := make([]any, 0, len(streamCtx.attrs)+2+len(extraAttrs))
	attrs = append(attrs, streamCtx.attrs...)
	attrs = append(attrs, "reason", reason)
	attrs = append(attrs, extraAttrs...)
	streamCtx.logger.Info("流式请求完成", attrs...)
}
