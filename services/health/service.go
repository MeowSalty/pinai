package health

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// ResourceHealthSummary 单个资源类型的健康状态汇总
type ResourceHealthSummary struct {
	Total       int64 `json:"total"`       // 总数
	Available   int64 `json:"available"`   // 可用数量
	Warning     int64 `json:"warning"`     // 警告数量
	Unavailable int64 `json:"unavailable"` // 不可用数量
	Unknown     int64 `json:"unknown"`     // 未知数量
}

// HealthSummaryResponse 健康状态统计响应
type HealthSummaryResponse struct {
	Platform ResourceHealthSummary `json:"platform"` // 平台健康状态统计
	APIKey   ResourceHealthSummary `json:"api_key"`  // 密钥健康状态统计
	Model    ResourceHealthSummary `json:"model"`    // 模型健康状态统计
}

// ModelHealthItem 单个模型健康状态项
type ModelHealthItem struct {
	ModelID       uint               `json:"model_id"`        // 模型 ID
	ModelName     string             `json:"model_name"`      // 模型名称
	ModelAlias    string             `json:"model_alias"`     // 模型别名
	Status        types.HealthStatus `json:"status"`          // 健康状态
	RetryCount    int                `json:"retry_count"`     // 重试次数
	LastError     string             `json:"last_error"`      // 最后错误信息
	LastCheckAt   time.Time          `json:"last_check_at"`   // 最后检查时间
	LastSuccessAt *time.Time         `json:"last_success_at"` // 最后成功时间
	SuccessCount  int                `json:"success_count"`   // 成功次数
	ErrorCount    int                `json:"error_count"`     // 错误次数
}

// ModelHealthListResponse 模型健康列表响应
type ModelHealthListResponse struct {
	Items    []ModelHealthItem `json:"items"`     // 模型健康列表
	Total    int               `json:"total"`     // 总数
	Page     int               `json:"page"`      // 当前页码
	PageSize int               `json:"page_size"` // 每页大小
}

// Service 定义健康服务接口
type Service interface {
	GetStorage() *Storage
	EnableHealth(resourceType types.ResourceType, resourceID uint) error
	DisableHealth(resourceType types.ResourceType, resourceID uint) error
	GetHealthSummary(ctx context.Context) (*HealthSummaryResponse, error)
	GetModelHealthList(ctx context.Context, page, pageSize int) (*ModelHealthListResponse, error)
}

// service 健康服务实现
type service struct {
	storage *Storage     // 健康状态存储，用于缓存和持久化
	logger  *slog.Logger // 日志记录器
}

// NewService 创建健康服务实例
//
// 该函数会在 health 包内部初始化 Storage，确保存储的初始化逻辑封装在 health 包中
//
// 参数：
//
//	ctx - 上下文，用于初始化存储
//	logger - 日志记录器
//
// 返回值：
//
//	Service - 健康服务实例
//	error - 初始化错误
func NewService(ctx context.Context, logger *slog.Logger) (Service, error) {
	logger.Info("开始初始化健康服务")

	// 在 health 包内部初始化存储
	storage, err := NewStorage(ctx, logger)
	if err != nil {
		logger.Error("初始化健康状态存储失败", "error", err)
		return nil, fmt.Errorf("初始化健康服务失败：%w", err)
	}

	serviceLogger := logger.WithGroup("health_service")
	serviceLogger.Info("健康服务初始化完成")
	return &service{
		storage: storage,
		logger:  serviceLogger,
	}, nil
}

// GetStorage 获取健康状态存储实例
//
// 该方法用于导出内部的健康状态存储实例，供其他服务（如 Portal Service）使用
//
// 返回值：
//
//	*Storage - 健康状态存储实例
func (s *service) GetStorage() *Storage {
	return s.storage
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
	s.logger.Info("启用资源健康状态",
		"resource_type", resourceType,
		"resource_id", resourceID)

	// 删除健康记录，让系统重新评估
	if err := s.storage.Delete(resourceType, resourceID); err != nil {
		s.logger.Error("启用资源健康状态失败",
			"error", err,
			"resource_type", resourceType,
			"resource_id", resourceID)
		return fmt.Errorf("启用资源健康状态失败：%w", err)
	}

	s.logger.Info("资源健康状态已启用",
		"resource_type", resourceType,
		"resource_id", resourceID)
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
	s.logger.Info("禁用资源健康状态",
		"resource_type", resourceType,
		"resource_id", resourceID)

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
		s.logger.Error("禁用资源健康状态失败",
			"error", err,
			"resource_type", resourceType,
			"resource_id", resourceID)
		return fmt.Errorf("禁用资源健康状态失败：%w", err)
	}

	s.logger.Info("资源健康状态已禁用",
		"resource_type", resourceType,
		"resource_id", resourceID)
	return nil
}

// GetHealthSummary 获取健康状态统计
//
// 该方法返回系统中所有资源类型（平台、密钥、模型）的健康状态统计信息，
// 包括总数、可用数量、警告数量、不可用数量和未知数量。
//
// 参数：
//
//	ctx - 上下文
//
// 返回值：
//
//	*HealthSummaryResponse - 健康状态统计响应
//	error - 操作错误
func (s *service) GetHealthSummary(ctx context.Context) (*HealthSummaryResponse, error) {
	s.logger.Debug("开始获取健康状态统计")

	q := query.Q

	// 获取各资源类型的总数
	platformTotal, err := q.Platform.WithContext(ctx).Count()
	if err != nil {
		s.logger.Error("获取平台总数失败", "error", err)
		return nil, fmt.Errorf("获取平台总数失败：%w", err)
	}

	apiKeyTotal, err := q.APIKey.WithContext(ctx).Count()
	if err != nil {
		s.logger.Error("获取密钥总数失败", "error", err)
		return nil, fmt.Errorf("获取密钥总数失败：%w", err)
	}

	modelTotal, err := q.Model.WithContext(ctx).Count()
	if err != nil {
		s.logger.Error("获取模型总数失败", "error", err)
		return nil, fmt.Errorf("获取模型总数失败：%w", err)
	}

	// 从缓存获取各状态数量
	platformCount := s.storage.CountByResourceType(types.ResourceTypePlatform)
	apiKeyCount := s.storage.CountByResourceType(types.ResourceTypeAPIKey)
	modelCount := s.storage.CountByResourceType(types.ResourceTypeModel)

	// 计算 Unknown 数量 = 总数 - 其他三种状态
	response := &HealthSummaryResponse{
		Platform: ResourceHealthSummary{
			Total:       platformTotal,
			Available:   platformCount.Available,
			Warning:     platformCount.Warning,
			Unavailable: platformCount.Unavailable,
			Unknown:     platformTotal - platformCount.Available - platformCount.Warning - platformCount.Unavailable,
		},
		APIKey: ResourceHealthSummary{
			Total:       apiKeyTotal,
			Available:   apiKeyCount.Available,
			Warning:     apiKeyCount.Warning,
			Unavailable: apiKeyCount.Unavailable,
			Unknown:     apiKeyTotal - apiKeyCount.Available - apiKeyCount.Warning - apiKeyCount.Unavailable,
		},
		Model: ResourceHealthSummary{
			Total:       modelTotal,
			Available:   modelCount.Available,
			Warning:     modelCount.Warning,
			Unavailable: modelCount.Unavailable,
			Unknown:     modelTotal - modelCount.Available - modelCount.Warning - modelCount.Unavailable,
		},
	}

	s.logger.Info("成功获取健康状态统计",
		"platform_total", platformTotal,
		"api_key_total", apiKeyTotal,
		"model_total", modelTotal)

	return response, nil
}

// GetModelHealthList 获取模型健康列表
//
// 该方法返回所有模型的健康状态列表，包括模型名称、别名和详细的健康信息。
// 支持分页查询，提升大数据量场景下的性能。
//
// 参数：
//
//	ctx - 上下文
//	page - 页码（从 1 开始）
//	pageSize - 每页大小
//
// 返回值：
//
//	*ModelHealthListResponse - 模型健康列表响应
//	error - 操作错误
func (s *service) GetModelHealthList(ctx context.Context, page, pageSize int) (*ModelHealthListResponse, error) {
	s.logger.Debug("开始获取模型健康列表",
		"page", page,
		"page_size", pageSize)

	// 从存储中获取所有模型的健康状态
	modelHealths := s.storage.GetByResourceType(types.ResourceTypeModel)

	// 如果没有健康记录，返回空列表
	if len(modelHealths) == 0 {
		s.logger.Info("没有找到任何模型健康记录")
		return &ModelHealthListResponse{
			Items:    []ModelHealthItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// 计算总数和分页范围
	total := len(modelHealths)
	start := (page - 1) * pageSize
	end := start + pageSize

	// 边界检查
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// 先分页，只处理当前页的健康记录
	pagedHealths := modelHealths[start:end]

	// 提取当前页的模型 ID
	modelIDs := make([]uint, 0, len(pagedHealths))
	healthMap := make(map[uint]*types.Health, len(pagedHealths))
	for _, health := range pagedHealths {
		modelIDs = append(modelIDs, health.ResourceID)
		healthMap[health.ResourceID] = health
	}

	// 只查询当前页需要的模型信息
	q := query.Q
	models, err := q.Model.WithContext(ctx).
		Select(q.Model.ID, q.Model.Name, q.Model.Alias_).
		Where(q.Model.ID.In(modelIDs...)).
		Find()
	if err != nil {
		s.logger.Error("查询模型信息失败", "error", err)
		return nil, fmt.Errorf("查询模型信息失败：%w", err)
	}

	// 组装响应数据
	items := make([]ModelHealthItem, 0, len(models))
	for _, model := range models {
		health := healthMap[model.ID]
		if health != nil {
			items = append(items, ModelHealthItem{
				ModelID:       model.ID,
				ModelName:     model.Name,
				ModelAlias:    model.Alias,
				Status:        health.Status,
				RetryCount:    health.RetryCount,
				LastError:     health.LastError,
				LastCheckAt:   health.LastCheckAt,
				LastSuccessAt: health.LastSuccessAt,
				SuccessCount:  health.SuccessCount,
				ErrorCount:    health.ErrorCount,
			})
		}
	}

	s.logger.Info("成功获取模型健康列表",
		"total", total,
		"page", page,
		"page_size", pageSize,
		"returned_count", len(items))

	return &ModelHealthListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
