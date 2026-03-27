package portal

import (
	"context"
	"log/slog"

	"github.com/MeowSalty/pinai/internal/app/gateway"
)

var _ gateway.GatewayPort = (*service)(nil)

// service Portal 服务实现
type service struct {
	runtime          portalRuntime
	modelMappingRule map[string]string
	logger           *slog.Logger
}

func newService(logger *slog.Logger, deps *assembledDependencies) Service {
	return &service{
		runtime:          deps.Runtime,
		modelMappingRule: deps.ModelMappingRule,
		logger:           logger,
	}
}

// New 创建新的 Portal 服务实例
//
// 该函数初始化所有必要的组件，包括数据仓库和网关管理器，并正确配置日志记录器。
//
// 参数：
//   - ctx: 上下文，用于初始化网关管理器
//   - logger: 日志记录器实例，用于记录处理过程中的日志信息
//   - modelMappingStr: 模型映射规则字符串，格式为 "key1:value1,key2:value2"
//   - healthStorage: 健康状态存储实例（最小依赖契约）
//
// 返回值：
//   - Service: 初始化后的 Portal 服务实例
//   - error: 初始化过程中可能出现的错误
func New(ctx context.Context, logger *slog.Logger, modelMappingStr string, healthStorage HealthStorage) (Service, error) {
	logger.Info("开始初始化 Portal 服务", "model_mapping", modelMappingStr)
	_ = ctx

	deps, err := buildServiceDependencies(logger, modelMappingStr, healthStorage)
	if err != nil {
		return nil, err
	}

	logger.Info("Portal 服务初始化完成")
	return newService(logger, deps), nil
}
