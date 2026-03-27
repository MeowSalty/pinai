package portal

import (
	"log/slog"

	"github.com/MeowSalty/pinai/internal/infra/portal/facade"
)

// assembledDependencies 表示 Portal 服务构造阶段的装配结果。
//
// 该结构仅用于初始化边界，避免请求执行路径感知装配细节。
type assembledDependencies = facade.AssembledDependencies

func buildServiceDependencies(logger *slog.Logger, modelMappingStr string, healthStorage HealthStorage) (*assembledDependencies, error) {
	return facade.BuildServiceDependencies(logger, modelMappingStr, healthStorage, parseModelMapping)
}
