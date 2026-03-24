package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/services/health"
)

// HealthReader 定义控制面读取健康状态所需的最小能力。
type HealthReader interface {
	Get(resourceType types.ResourceType, resourceID uint) (*types.Health, error)
	CountByPlatform(resourceType types.ResourceType, resourceToPlatform map[uint]uint) map[uint]health.StatusCount
}

// ControlTx 定义控制面应用服务事务边界。
type ControlTx interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ControlAuditEvent 表示控制面审计事件。
type ControlAuditEvent struct {
	Action     string
	Resource   string
	ResourceID uint
	Result     string
	Detail     string
}

// ControlAuditLogger 定义控制面审计记录接口。
type ControlAuditLogger interface {
	Log(ctx context.Context, event ControlAuditEvent) error
}

// PlatformControlRepository 定义平台控制面写路径所需最小仓储能力。
type PlatformControlRepository interface {
	ExistsPlatform(ctx context.Context, platformID uint) (bool, error)
	CreatePlatform(ctx context.Context, platform *types.Platform) error
	UpdatePlatform(ctx context.Context, platformID uint, updates types.Platform) (int64, error)
	GetPlatform(ctx context.Context, platformID uint) (*types.Platform, error)
	EnablePlatformHealth(ctx context.Context, platformID uint) error
	DisablePlatformHealth(ctx context.Context, platformID uint) error
}

// noOpControlTx 为默认事务执行器（无事务实现）。
type noOpControlTx struct{}

// NewNoOpControlTx 创建无事务执行器。
func NewNoOpControlTx() ControlTx {
	return noOpControlTx{}
}

// WithinTx 执行业务函数，不开启实际数据库事务。
func (noOpControlTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("事务执行函数不能为空")
	}
	return fn(ctx)
}

// noOpControlAuditLogger 为默认 no-op 审计实现。
type noOpControlAuditLogger struct {
	logger *slog.Logger
}

// NewNoOpControlAuditLogger 创建 no-op 审计记录器。
func NewNoOpControlAuditLogger(logger *slog.Logger) ControlAuditLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &noOpControlAuditLogger{logger: logger.WithGroup("control_audit")}
}

// Log 记录审计占位日志，不做持久化。
func (a *noOpControlAuditLogger) Log(ctx context.Context, event ControlAuditEvent) error {
	_ = ctx

	a.logger.Debug("控制面审计占位事件",
		actionField(event.Action),
		resourceField(event.Resource),
		slog.Uint64("resource_id", uint64(event.ResourceID)),
		resultField(event.Result),
		detailField(event.Detail),
	)
	return nil
}

func actionField(v string) slog.Attr {
	return slog.String("action", v)
}

func resourceField(v string) slog.Attr {
	return slog.String("resource", v)
}

func resultField(v string) slog.Attr {
	return slog.String("result", v)
}

func detailField(v string) slog.Attr {
	return slog.String("detail", v)
}
