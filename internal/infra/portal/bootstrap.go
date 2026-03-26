package portal

import (
	"fmt"
	"log/slog"

	"github.com/MeowSalty/portal"
)

// assembledDependencies 表示 Portal 服务构造阶段的装配结果。
//
// 该结构仅用于初始化边界，避免请求执行路径感知装配细节。
type assembledDependencies struct {
	runtime          portalRuntime
	modelMappingRule map[string]string
}

func buildServiceDependencies(logger *slog.Logger, modelMappingStr string, healthStorage HealthStorage) (*assembledDependencies, error) {
	repo := newPortalAdapterRepository(logger)
	healthAdapter := newHealthStorageAdapter(healthStorage)

	runtime, err := newPortalRuntime(logger, repo, healthAdapter)
	if err != nil {
		return nil, err
	}

	modelMappingRule, err := parseModelMappingRule(logger, modelMappingStr)
	if err != nil {
		return nil, err
	}

	return &assembledDependencies{
		runtime:          runtime,
		modelMappingRule: modelMappingRule,
	}, nil
}

func newPortalRuntime(logger *slog.Logger, repo *portalAdapterRepository, adapter *healthStorageAdapter) (portalRuntime, error) {
	logger.Debug("正在创建 Portal 运行时")
	runtime, err := portal.New(portal.Config{
		PlatformRepo:  repo,
		ModelRepo:     repo,
		KeyRepo:       repo,
		HealthStorage: adapter,
		LogRepo:       repo,
		Logger:        NewSlogAdapter(logger),
	})
	if err != nil {
		logger.Error("创建 Portal 运行时失败", "error", err)
		return nil, fmt.Errorf("创建 Portal 运行时失败：%w", err)
	}

	logger.Info("Portal 运行时创建成功")
	return runtime, nil
}

func parseModelMappingRule(logger *slog.Logger, modelMappingStr string) (map[string]string, error) {
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
