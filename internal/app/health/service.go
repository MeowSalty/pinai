package health

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

// Service 定义健康服务接口
type Service interface {
	EnableHealth(resourceType types.ResourceType, resourceID uint) error
	DisableHealth(resourceType types.ResourceType, resourceID uint) error
	GetHealthSummary(ctx context.Context) (*HealthSummaryResponse, error)
	GetPlatformHealthList(ctx context.Context, page, pageSize int) (*PlatformHealthListResponse, error)
	GetAPIKeyHealthList(ctx context.Context, page, pageSize int) (*APIKeyHealthListResponse, error)
	GetModelHealthList(ctx context.Context, page, pageSize int) (*ModelHealthListResponse, error)
	GetIssues(ctx context.Context) (*IssuesListResponse, error)
}

// service 健康服务实现
type service struct {
	storage *Storage     // 健康状态存储，用于缓存和持久化
	logger  *slog.Logger // 日志记录器
}

// NewService 创建健康服务实例
//
// 参数：
//
//	storage - 健康状态存储实例，由组装层创建后注入
//	logger - 日志记录器
//
// 返回值：
//
//	Service - 健康服务实例
//	error - 初始化错误
func NewService(storage *Storage, logger *slog.Logger) (Service, error) {
	logger.Debug("开始初始化健康服务")

	if storage == nil {
		logger.Error("初始化健康服务失败", "error", "health storage 不能为空")
		return nil, fmt.Errorf("初始化健康服务失败：health storage 不能为空")
	}

	serviceLogger := logger.WithGroup("health_service").With("component", "health_service")
	serviceLogger.Info("健康服务初始化完成")
	return &service{
		storage: storage,
		logger:  serviceLogger,
	}, nil
}

// EnableHealth 启用/恢复资源健康状态
//
// 该方法通过删除健康记录来重置资源的健康状态，让系统在下次请求时重新评估。
// 删除记录后，资源状态将变为 Unknown，所有退避信息将被清除。
//
// 参数：
//
//	resourceType - 资源类型（平台、密钥、模型）
//	resourceID - 资源 ID
//
// 返回值：
//
//	error - 操作错误
func (s *service) EnableHealth(resourceType types.ResourceType, resourceID uint) error {
	logger := s.logger.With(
		"operation", "enable_health",
		"resource_type", resourceType,
		"resource_id", resourceID,
	)

	logger.Info("启用资源健康状态",
		"resource_type", resourceType,
		"resource_id", resourceID)

	// 删除健康记录，让系统重新评估
	if err := s.storage.Delete(resourceType, resourceID); err != nil {
		logger.Error("启用资源健康状态失败", "error", err)
		return fmt.Errorf("启用资源健康状态失败：%w", err)
	}

	logger.Info("资源健康状态已启用")
	return nil
}

// DisableHealth 禁用资源健康状态
//
// 该方法将资源的健康状态设置为 Unavailable，表示手动禁用该资源。
//
// 参数：
//
//	resourceType - 资源类型（平台、密钥、模型）
//	resourceID - 资源 ID
//
// 返回值：
//
//	error - 操作错误
func (s *service) DisableHealth(resourceType types.ResourceType, resourceID uint) error {
	logger := s.logger.With(
		"operation", "disable_health",
		"resource_type", resourceType,
		"resource_id", resourceID,
	)

	logger.Info("禁用资源健康状态")

	// 创建 Unavailable 状态的健康记录
	now := time.Now()
	health := &types.Health{
		ResourceType:    resourceType,
		ResourceID:      resourceID,
		Status:          types.HealthStatusUnavailable,
		LastError:       "手动禁用",
		LastCheckAt:     now,
		RetryCount:      0,
		BackoffDuration: 0,
	}

	// 保存到存储
	if err := s.storage.Set(health); err != nil {
		logger.Error("禁用资源健康状态失败", "error", err)
		return fmt.Errorf("禁用资源健康状态失败：%w", err)
	}

	logger.Info("资源健康状态已禁用")
	return nil
}
