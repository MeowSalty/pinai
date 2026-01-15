package health

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
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

// PlatformHealthItem 单个平台健康状态项
type PlatformHealthItem struct {
	PlatformID    uint               `json:"platform_id"`     // 平台 ID
	PlatformName  string             `json:"platform_name"`   // 平台名称
	Status        types.HealthStatus `json:"status"`          // 健康状态
	RetryCount    int                `json:"retry_count"`     // 重试次数
	LastError     string             `json:"last_error"`      // 最后错误信息
	LastCheckAt   time.Time          `json:"last_check_at"`   // 最后检查时间
	LastSuccessAt *time.Time         `json:"last_success_at"` // 最后成功时间
	SuccessCount  int                `json:"success_count"`   // 成功次数
	ErrorCount    int                `json:"error_count"`     // 错误次数
}

// PlatformHealthListResponse 平台健康列表响应
type PlatformHealthListResponse struct {
	Items    []PlatformHealthItem `json:"items"`     // 平台健康列表
	Total    int                  `json:"total"`     // 总数
	Page     int                  `json:"page"`      // 当前页码
	PageSize int                  `json:"page_size"` // 每页大小
}

// APIKeyHealthItem 单个密钥健康状态项
type APIKeyHealthItem struct {
	KeyID         uint               `json:"key_id"`          // 密钥 ID
	KeyValue      string             `json:"key_value"`       // 密钥值
	Status        types.HealthStatus `json:"status"`          // 健康状态
	RetryCount    int                `json:"retry_count"`     // 重试次数
	LastError     string             `json:"last_error"`      // 最后错误信息
	LastCheckAt   time.Time          `json:"last_check_at"`   // 最后检查时间
	LastSuccessAt *time.Time         `json:"last_success_at"` // 最后成功时间
	SuccessCount  int                `json:"success_count"`   // 成功次数
	ErrorCount    int                `json:"error_count"`     // 错误次数
}

// APIKeyHealthListResponse 密钥健康列表响应
type APIKeyHealthListResponse struct {
	Items    []APIKeyHealthItem `json:"items"`     // 密钥健康列表
	Total    int                `json:"total"`     // 总数
	Page     int                `json:"page"`      // 当前页码
	PageSize int                `json:"page_size"` // 每页大小
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

// IssueItem 单个异常资源项
type IssueItem struct {
	ResourceType types.ResourceType `json:"resource_type"` // 资源类型
	ResourceID   uint               `json:"resource_id"`   // 资源 ID
	ResourceName string             `json:"resource_name"` // 资源名称
	Status       types.HealthStatus `json:"status"`        // 资源状态
	LastCheckAt  time.Time          `json:"last_check_at"` // 最后检查
	LastError    string             `json:"last_error"`    // 最后错误
}

// IssuesListResponse 异常资源列表响应
type IssuesListResponse struct {
	Items []IssueItem `json:"items"` // 异常资源列表
}

// Service 定义健康服务接口
type Service interface {
	GetStorage() *Storage
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

// GetPlatformHealthList 获取平台健康列表
//
// 该方法返回所有平台的健康状态列表，包括平台名称和详细的健康信息。
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
//	*PlatformHealthListResponse - 平台健康列表响应
//	error - 操作错误
func (s *service) GetPlatformHealthList(ctx context.Context, page, pageSize int) (*PlatformHealthListResponse, error) {
	s.logger.Debug("开始获取平台健康列表",
		"page", page,
		"page_size", pageSize)

	// 从存储中获取所有平台的健康状态
	platformHealths := s.storage.GetByResourceType(types.ResourceTypePlatform)

	// 如果没有健康记录，返回空列表
	if len(platformHealths) == 0 {
		s.logger.Info("没有找到任何平台健康记录")
		return &PlatformHealthListResponse{
			Items:    []PlatformHealthItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// 按最后检查时间降序排序
	sort.Slice(platformHealths, func(i, j int) bool {
		return platformHealths[i].LastCheckAt.After(platformHealths[j].LastCheckAt)
	})

	// 计算总数和分页范围
	total := len(platformHealths)
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
	pagedHealths := platformHealths[start:end]

	// 提取当前页的平台 ID
	platformIDs := make([]uint, 0, len(pagedHealths))
	healthMap := make(map[uint]*types.Health, len(pagedHealths))
	for _, health := range pagedHealths {
		platformIDs = append(platformIDs, health.ResourceID)
		healthMap[health.ResourceID] = health
	}

	// 只查询当前页需要的平台信息
	q := query.Q
	platforms, err := q.Platform.WithContext(ctx).
		Select(q.Platform.ID, q.Platform.Name).
		Where(q.Platform.ID.In(platformIDs...)).
		Find()
	if err != nil {
		s.logger.Error("查询平台信息失败", "error", err)
		return nil, fmt.Errorf("查询平台信息失败：%w", err)
	}

	// 构建平台 ID 到平台信息的映射
	platformMap := make(map[uint]*types.Platform, len(platforms))
	for i := range platforms {
		platformMap[platforms[i].ID] = platforms[i]
	}

	// 按照 pagedHealths 的顺序组装响应数据，保持排序
	items := make([]PlatformHealthItem, 0, len(pagedHealths))
	for _, health := range pagedHealths {
		platform := platformMap[health.ResourceID]
		if platform != nil {
			items = append(items, PlatformHealthItem{
				PlatformID:    platform.ID,
				PlatformName:  platform.Name,
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

	s.logger.Info("成功获取平台健康列表",
		"total", total,
		"page", page,
		"page_size", pageSize,
		"returned_count", len(items))

	return &PlatformHealthListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetAPIKeyHealthList 获取密钥健康列表
//
// 该方法返回所有密钥的健康状态列表，包括密钥值和详细的健康信息。
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
//	*APIKeyHealthListResponse - 密钥健康列表响应
//	error - 操作错误
func (s *service) GetAPIKeyHealthList(ctx context.Context, page, pageSize int) (*APIKeyHealthListResponse, error) {
	s.logger.Debug("开始获取密钥健康列表",
		"page", page,
		"page_size", pageSize)

	// 从存储中获取所有密钥的健康状态
	keyHealths := s.storage.GetByResourceType(types.ResourceTypeAPIKey)

	// 如果没有健康记录，返回空列表
	if len(keyHealths) == 0 {
		s.logger.Info("没有找到任何密钥健康记录")
		return &APIKeyHealthListResponse{
			Items:    []APIKeyHealthItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// 按最后检查时间降序排序
	sort.Slice(keyHealths, func(i, j int) bool {
		return keyHealths[i].LastCheckAt.After(keyHealths[j].LastCheckAt)
	})

	// 计算总数和分页范围
	total := len(keyHealths)
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
	pagedHealths := keyHealths[start:end]

	// 提取当前页的密钥 ID
	keyIDs := make([]uint, 0, len(pagedHealths))
	healthMap := make(map[uint]*types.Health, len(pagedHealths))
	for _, health := range pagedHealths {
		keyIDs = append(keyIDs, health.ResourceID)
		healthMap[health.ResourceID] = health
	}

	// 只查询当前页需要的密钥信息
	q := query.Q
	keys, err := q.APIKey.WithContext(ctx).
		Select(q.APIKey.ID, q.APIKey.Value).
		Where(q.APIKey.ID.In(keyIDs...)).
		Find()
	if err != nil {
		s.logger.Error("查询密钥信息失败", "error", err)
		return nil, fmt.Errorf("查询密钥信息失败：%w", err)
	}

	// 提取平台 ID
	platformIDs := make([]uint, 0, len(keys))
	for _, key := range keys {
		platformIDs = append(platformIDs, key.PlatformID)
	}

	// 构建密钥 ID 到密钥信息的映射
	keyMap := make(map[uint]*types.APIKey, len(keys))
	for i := range keys {
		keyMap[keys[i].ID] = keys[i]
	}

	// 按照 pagedHealths 的顺序组装响应数据，保持排序
	items := make([]APIKeyHealthItem, 0, len(pagedHealths))
	for _, health := range pagedHealths {
		key := keyMap[health.ResourceID]
		if key != nil {
			items = append(items, APIKeyHealthItem{
				KeyID:         key.ID,
				KeyValue:      key.Value,
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

	s.logger.Info("成功获取密钥健康列表",
		"total", total,
		"page", page,
		"page_size", pageSize,
		"returned_count", len(items))

	return &APIKeyHealthListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
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

	// 按最后检查时间降序排序
	sort.Slice(modelHealths, func(i, j int) bool {
		return modelHealths[i].LastCheckAt.After(modelHealths[j].LastCheckAt)
	})

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

	// 构建模型 ID 到模型信息的映射
	modelMap := make(map[uint]*types.Model, len(models))
	for i := range models {
		modelMap[models[i].ID] = models[i]
	}

	// 按照 pagedHealths 的顺序组装响应数据，保持排序
	items := make([]ModelHealthItem, 0, len(pagedHealths))
	for _, health := range pagedHealths {
		model := modelMap[health.ResourceID]
		if model != nil {
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

// GetIssues 获取所有异常资源列表（状态为 unavailable）
//
// 该方法返回所有状态为 Unavailable 的资源列表，包括资源类型、ID、名称、状态、最后检查时间和最后错误信息。
//
// 参数：
//
//	ctx - 上下文
//
// 返回值：
//
//	*IssuesListResponse - 异常资源列表响应
//	error - 操作错误
func (s *service) GetIssues(ctx context.Context) (*IssuesListResponse, error) {
	s.logger.Debug("开始获取异常资源列表")

	// 从存储中获取所有 Unavailable 状态的健康记录
	unavailableHealths := s.storage.GetByStatus(types.HealthStatusUnavailable)

	// 如果没有异常记录，返回空列表
	if len(unavailableHealths) == 0 {
		s.logger.Info("没有找到任何异常资源")
		return &IssuesListResponse{
			Items: []IssueItem{},
		}, nil
	}

	// 按最后检查时间降序排序
	sort.Slice(unavailableHealths, func(i, j int) bool {
		return unavailableHealths[i].LastCheckAt.After(unavailableHealths[j].LastCheckAt)
	})

	// 按资源类型分组
	platformIDs := make([]uint, 0)
	keyIDs := make([]uint, 0)
	modelIDs := make([]uint, 0)
	healthMap := make(map[string]*types.Health)

	for _, health := range unavailableHealths {
		key := fmt.Sprintf("%d:%d", health.ResourceType, health.ResourceID)
		healthMap[key] = health

		switch health.ResourceType {
		case types.ResourceTypePlatform:
			platformIDs = append(platformIDs, health.ResourceID)
		case types.ResourceTypeAPIKey:
			keyIDs = append(keyIDs, health.ResourceID)
		case types.ResourceTypeModel:
			modelIDs = append(modelIDs, health.ResourceID)
		}
	}

	q := query.Q

	// 查询平台信息
	platformMap := make(map[uint]string)
	if len(platformIDs) > 0 {
		platforms, err := q.Platform.WithContext(ctx).
			Select(q.Platform.ID, q.Platform.Name).
			Where(q.Platform.ID.In(platformIDs...)).
			Find()
		if err != nil {
			s.logger.Error("查询平台信息失败", "error", err)
			return nil, fmt.Errorf("查询平台信息失败：%w", err)
		}
		for _, platform := range platforms {
			platformMap[platform.ID] = platform.Name
		}
	}

	// 查询密钥信息
	keyMap := make(map[uint]string)
	if len(keyIDs) > 0 {
		keys, err := q.APIKey.WithContext(ctx).
			Select(q.APIKey.ID, q.APIKey.Value).
			Where(q.APIKey.ID.In(keyIDs...)).
			Find()
		if err != nil {
			s.logger.Error("查询密钥信息失败", "error", err)
			return nil, fmt.Errorf("查询密钥信息失败：%w", err)
		}
		for _, key := range keys {
			keyMap[key.ID] = key.Value
		}
	}

	// 查询模型信息
	modelMap := make(map[uint]string)
	if len(modelIDs) > 0 {
		models, err := q.Model.WithContext(ctx).
			Select(q.Model.ID, q.Model.Name).
			Where(q.Model.ID.In(modelIDs...)).
			Find()
		if err != nil {
			s.logger.Error("查询模型信息失败", "error", err)
			return nil, fmt.Errorf("查询模型信息失败：%w", err)
		}
		for _, model := range models {
			modelMap[model.ID] = model.Name
		}
	}

	// 组装响应数据
	items := make([]IssueItem, 0, len(unavailableHealths))
	for _, health := range unavailableHealths {
		var resourceName string
		switch health.ResourceType {
		case types.ResourceTypePlatform:
			resourceName = platformMap[health.ResourceID]
		case types.ResourceTypeAPIKey:
			resourceName = keyMap[health.ResourceID]
		case types.ResourceTypeModel:
			resourceName = modelMap[health.ResourceID]
		}

		// 只添加能找到资源名称的项
		if resourceName != "" {
			items = append(items, IssueItem{
				ResourceType: health.ResourceType,
				ResourceID:   health.ResourceID,
				ResourceName: resourceName,
				Status:       health.Status,
				LastCheckAt:  health.LastCheckAt,
				LastError:    health.LastError,
			})
		}
	}

	s.logger.Info("成功获取异常资源列表", "count", len(items))

	return &IssuesListResponse{
		Items: items,
	}, nil
}
