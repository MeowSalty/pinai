package facade

import (
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/internal/infra/portal/healthadapter"
	"github.com/MeowSalty/pinai/internal/infra/portal/logadapter"
	"github.com/MeowSalty/pinai/internal/infra/portal/repository"
	runtimepkg "github.com/MeowSalty/pinai/internal/infra/portal/runtime"
	portalSDK "github.com/MeowSalty/portal"
)

// AssembledDependencies 表示 Portal 服务构造阶段的装配结果。
//
// 该结构仅用于初始化边界，避免请求执行路径感知装配细节。
type AssembledDependencies struct {
	Runtime          runtimepkg.Runtime
	ModelMappingRule map[string]string
}

// BuildServiceDependencies 构建 Portal 服务所需依赖。
func BuildServiceDependencies(
	logger *slog.Logger,
	modelMappingStr string,
	healthStorage healthadapter.HealthStorage,
	parseModelMapping func(string) (map[string]string, error),
) (*AssembledDependencies, error) {
	repo := repository.New(logger)
	health := healthadapter.New(healthStorage)

	runtime, err := newPortalRuntime(logger, repo, health)
	if err != nil {
		return nil, err
	}

	modelMappingRule, err := parseModelMappingRule(logger, modelMappingStr, parseModelMapping)
	if err != nil {
		return nil, err
	}

	return &AssembledDependencies{
		Runtime:          runtime,
		ModelMappingRule: modelMappingRule,
	}, nil
}

func newPortalRuntime(logger *slog.Logger, repo *repository.Repository, adapter *healthadapter.Adapter) (runtimepkg.Runtime, error) {
	logger.Debug("正在创建 Portal 运行时")
	runtime, err := portalSDK.New(portalSDK.Config{
		PlatformRepo:  repo,
		ModelRepo:     repo,
		KeyRepo:       repo,
		HealthStorage: adapter,
		LogRepo:       repo,
		Logger:        logadapter.New(logger),
	})
	if err != nil {
		logger.Error("创建 Portal 运行时失败", "error", err)
		return nil, fmt.Errorf("创建 Portal 运行时失败：%w", err)
	}

	logger.Info("Portal 运行时创建成功")
	return runtime, nil
}

func parseModelMappingRule(logger *slog.Logger, modelMappingStr string, parseModelMapping func(string) (map[string]string, error)) (map[string]string, error) {
	logger.Debug("正在解析模型映射规则")
	rule, err := parseModelMapping(modelMappingStr)
	if err != nil {
		logger.Error("解析模型映射规则失败", "error", err, "mapping_str", modelMappingStr)
		return nil, fmt.Errorf("解析模型映射规则失败：%w", err)
	}

	if len(rule) == 0 {
		logger.Debug("未启用模型映射规则")
	} else {
		logger.Info("使用自定义模型映射规则", "mapping", rule, "count", len(rule))
	}

	return rule, nil
}
