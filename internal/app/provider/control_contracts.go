package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"github.com/MeowSalty/pinai/internal/app/health"
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
	ListAPIKeysByPlatform(ctx context.Context, platformID uint) ([]*types.APIKey, error)
	CountModelsByAPIKey(ctx context.Context, apiKeyID uint) (int64, error)
	ListModelsByAPIKey(ctx context.Context, apiKeyID uint) ([]*types.Model, error)
	ClearAPIKeyModelRelations(ctx context.Context, apiKeyID uint) error
	AppendAPIKeyModels(ctx context.Context, apiKeyID uint, models []*types.Model) error
	DeleteModelsByPlatform(ctx context.Context, platformID uint) (int64, error)
	DeleteAPIKeysByPlatform(ctx context.Context, platformID uint) (int64, error)
	DeletePlatform(ctx context.Context, platformID uint) (int64, error)
	EnablePlatformHealth(ctx context.Context, platformID uint) error
	DisablePlatformHealth(ctx context.Context, platformID uint) error
}

// ModelControlRepository 定义模型控制面写路径所需最小仓储能力。
type ModelControlRepository interface {
	ExistsPlatform(ctx context.Context, platformID uint) (bool, error)
	ListModelsByIDs(ctx context.Context, modelIDs []uint) ([]*types.Model, error)
	ListAPIKeysByPlatformAndIDs(ctx context.Context, platformID uint, apiKeyIDs []uint) ([]*types.APIKey, error)
	CreateModel(ctx context.Context, model *types.Model) error
	GetModel(ctx context.Context, modelID uint) (*types.Model, error)
	GetModelWithAPIKeys(ctx context.Context, modelID uint) (*types.Model, error)
	ReplaceModelAPIKeys(ctx context.Context, modelID uint, apiKeys []*types.APIKey) error
	UpdateModelFields(ctx context.Context, modelID uint, updates map[string]interface{}) (int64, error)
	ListAPIKeysByModel(ctx context.Context, modelID uint) ([]*types.APIKey, error)
	ClearModelAPIKeyRelations(ctx context.Context, modelID uint) error
	AppendModelAPIKeys(ctx context.Context, modelID uint, apiKeys []*types.APIKey) error
	DeleteModelByID(ctx context.Context, modelID uint) (int64, error)
	DeleteModelsByIDs(ctx context.Context, modelIDs []uint) (int64, error)
	EnableModelHealth(ctx context.Context, modelID uint) error
	DisableModelHealth(ctx context.Context, modelID uint) error
}

// KeyControlRepository 定义密钥控制面写路径所需最小仓储能力。
type KeyControlRepository interface {
	ExistsPlatform(ctx context.Context, platformID uint) (bool, error)
	CreateAPIKey(ctx context.Context, key *types.APIKey) error
	GetAPIKey(ctx context.Context, keyID uint) (*types.APIKey, error)
	UpdateAPIKey(ctx context.Context, keyID uint, updates types.APIKey) (int64, error)
	ListModelsByAPIKey(ctx context.Context, keyID uint) ([]*types.Model, error)
	ClearAPIKeyModelRelations(ctx context.Context, keyID uint) error
	AppendAPIKeyModels(ctx context.Context, keyID uint, models []*types.Model) error
	DeleteAPIKeyByID(ctx context.Context, keyID uint) (int64, error)
	EnableAPIKeyHealth(ctx context.Context, keyID uint) error
	DisableAPIKeyHealth(ctx context.Context, keyID uint) error
}

// EndpointControlRepository 定义端点控制面写路径所需最小仓储能力。
type EndpointControlRepository interface {
	ExistsPlatform(ctx context.Context, platformID uint) (bool, error)
	CreateEndpoint(ctx context.Context, endpoint *types.Endpoint) error
	GetEndpoint(ctx context.Context, endpointID uint) (*types.Endpoint, error)
	UpdateEndpointFields(ctx context.Context, endpointID uint, updates types.Endpoint, fieldNames []string) (int64, error)
	DeleteEndpointByID(ctx context.Context, endpointID uint) (int64, error)
	GetLatestEndpointByPlatform(ctx context.Context, platformID uint) (*types.Endpoint, error)
	SetEndpointDefault(ctx context.Context, endpointID uint, isDefault bool) (int64, error)
	CountDefaultEndpointsByPlatform(ctx context.Context, platformID uint) (int64, error)
}

type controlTxQueryKey struct{}

func queryFromControlTx(ctx context.Context) *query.Query {
	if ctx == nil {
		return nil
	}

	tx, ok := ctx.Value(controlTxQueryKey{}).(*query.Query)
	if !ok {
		return nil
	}

	return tx
}

// queryControlTx 是基于 gorm query 的事务执行器实现。
type queryControlTx struct{}

// NewQueryControlTx 创建基于数据库事务的执行器。
func NewQueryControlTx() ControlTx {
	return queryControlTx{}
}

// WithinTx 在数据库事务中执行业务函数。
func (queryControlTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("事务执行函数不能为空")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	return query.Q.Transaction(func(tx *query.Query) error {
		txCtx := context.WithValue(ctx, controlTxQueryKey{}, tx)
		return fn(txCtx)
	})
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
