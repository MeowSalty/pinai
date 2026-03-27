package portal

import (
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/internal/infra/portal/healthadapter"
	"github.com/MeowSalty/pinai/internal/infra/portal/logadapter"
	"github.com/MeowSalty/pinai/internal/infra/portal/repository"
	portalSDK "github.com/MeowSalty/portal"
)

// portalFacadeDependencies 表示 Portal facade 构造阶段的装配结果。
//
// 该结构仅用于初始化边界，避免请求执行路径感知装配细节。
type portalFacadeDependencies struct {
	Runtime          gatewayRuntime
	ModelMappingRule map[string]string
}

// assemblePortalFacadeDependencies 负责收口 Portal facade 的依赖装配。
func assemblePortalFacadeDependencies(logger *slog.Logger, modelMappingStr string, healthStorage HealthStorage) (*portalFacadeDependencies, error) {
	repo := repository.New(logger)
	health := healthadapter.New(healthStorage)

	runtime, err := newGatewayRuntime(logger, repo, health)
	if err != nil {
		return nil, err
	}

	modelMappingRule, err := parsePortalModelMapping(logger, modelMappingStr)
	if err != nil {
		return nil, err
	}

	return &portalFacadeDependencies{
		Runtime:          runtime,
		ModelMappingRule: modelMappingRule,
	}, nil
}

func newGatewayRuntime(logger *slog.Logger, repo *repository.Repository, adapter *healthadapter.Adapter) (gatewayRuntime, error) {
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

func parsePortalModelMapping(logger *slog.Logger, modelMappingStr string) (map[string]string, error) {
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
