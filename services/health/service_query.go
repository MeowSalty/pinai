package health

import (
	"context"
	"fmt"
	"sort"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

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
	logger := s.logger.With("operation", "get_summary")
	logger.Debug("开始获取健康状态统计")

	q := query.Q

	// 获取各资源类型的总数
	platformTotal, err := q.Platform.WithContext(ctx).Count()
	if err != nil {
		logger.Error("获取平台总数失败", "error", err)
		return nil, fmt.Errorf("获取平台总数失败：%w", err)
	}

	apiKeyTotal, err := q.APIKey.WithContext(ctx).Count()
	if err != nil {
		logger.Error("获取密钥总数失败", "error", err)
		return nil, fmt.Errorf("获取密钥总数失败：%w", err)
	}

	modelTotal, err := q.Model.WithContext(ctx).Count()
	if err != nil {
		logger.Error("获取模型总数失败", "error", err)
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

	logger.Debug("成功获取健康状态统计",
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
	logger := s.logger.With(
		"operation", "list_platforms",
		"page", page,
		"page_size", pageSize,
	)

	logger.Debug("开始获取平台健康列表",
		"page", page,
		"page_size", pageSize)

	// 从存储中获取所有平台的健康状态
	platformHealths := s.storage.GetByResourceType(types.ResourceTypePlatform)

	// 如果没有健康记录，返回空列表
	if len(platformHealths) == 0 {
		logger.Debug("没有找到任何平台健康记录")
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
	for _, health := range pagedHealths {
		platformIDs = append(platformIDs, health.ResourceID)
	}

	// 只查询当前页需要的平台信息
	q := query.Q
	platforms, err := q.Platform.WithContext(ctx).
		Select(q.Platform.ID, q.Platform.Name).
		Where(q.Platform.ID.In(platformIDs...)).
		Find()
	if err != nil {
		logger.Error("查询平台信息失败", "error", err)
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
				PlatformID:              platform.ID,
				PlatformName:            platform.Name,
				Status:                  health.Status,
				RetryCount:              health.RetryCount,
				NextAvailableAt:         health.NextAvailableAt,
				BackoffDuration:         health.BackoffDuration,
				LastError:               health.LastError,
				LastErrorCode:           health.LastErrorCode,
				LastErrorMessage:        health.LastErrorMessage,
				LastStructuredErrorCode: health.LastStructuredErrorCode,
				LastHTTPStatus:          health.LastHTTPStatus,
				LastErrorFrom:           health.LastErrorFrom,
				LastCauseMessage:        health.LastCauseMessage,
				LastCheckAt:             health.LastCheckAt,
				LastSuccessAt:           health.LastSuccessAt,
				SuccessCount:            health.SuccessCount,
				ErrorCount:              health.ErrorCount,
			})
		}
	}

	logger.Debug("成功获取平台健康列表",
		"total", total,
		"page", page,
		"page_size", pageSize,
		"item_count", len(items))

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
	logger := s.logger.With(
		"operation", "list_keys",
		"page", page,
		"page_size", pageSize,
	)

	logger.Debug("开始获取密钥健康列表",
		"page", page,
		"page_size", pageSize)

	// 从存储中获取所有密钥的健康状态
	keyHealths := s.storage.GetByResourceType(types.ResourceTypeAPIKey)

	// 如果没有健康记录，返回空列表
	if len(keyHealths) == 0 {
		logger.Debug("没有找到任何密钥健康记录")
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
	for _, health := range pagedHealths {
		keyIDs = append(keyIDs, health.ResourceID)
	}

	// 只查询当前页需要的密钥信息
	q := query.Q
	keys, err := q.APIKey.WithContext(ctx).
		Select(q.APIKey.ID, q.APIKey.Value).
		Where(q.APIKey.ID.In(keyIDs...)).
		Find()
	if err != nil {
		logger.Error("查询密钥信息失败", "error", err)
		return nil, fmt.Errorf("查询密钥信息失败：%w", err)
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
				KeyID:                   key.ID,
				KeyValue:                key.Value,
				Status:                  health.Status,
				RetryCount:              health.RetryCount,
				NextAvailableAt:         health.NextAvailableAt,
				BackoffDuration:         health.BackoffDuration,
				LastError:               health.LastError,
				LastErrorCode:           health.LastErrorCode,
				LastErrorMessage:        health.LastErrorMessage,
				LastStructuredErrorCode: health.LastStructuredErrorCode,
				LastHTTPStatus:          health.LastHTTPStatus,
				LastErrorFrom:           health.LastErrorFrom,
				LastCauseMessage:        health.LastCauseMessage,
				LastCheckAt:             health.LastCheckAt,
				LastSuccessAt:           health.LastSuccessAt,
				SuccessCount:            health.SuccessCount,
				ErrorCount:              health.ErrorCount,
			})
		}
	}

	logger.Debug("成功获取密钥健康列表",
		"total", total,
		"page", page,
		"page_size", pageSize,
		"item_count", len(items))

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
	logger := s.logger.With(
		"operation", "list_models",
		"page", page,
		"page_size", pageSize,
	)

	logger.Debug("开始获取模型健康列表",
		"page", page,
		"page_size", pageSize)

	// 从存储中获取所有模型的健康状态
	modelHealths := s.storage.GetByResourceType(types.ResourceTypeModel)

	// 如果没有健康记录，返回空列表
	if len(modelHealths) == 0 {
		logger.Debug("没有找到任何模型健康记录")
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
	for _, health := range pagedHealths {
		modelIDs = append(modelIDs, health.ResourceID)
	}

	// 只查询当前页需要的模型信息
	q := query.Q
	models, err := q.Model.WithContext(ctx).
		Select(q.Model.ID, q.Model.Name, q.Model.Alias_).
		Where(q.Model.ID.In(modelIDs...)).
		Find()
	if err != nil {
		logger.Error("查询模型信息失败", "error", err)
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
				ModelID:                 model.ID,
				ModelName:               model.Name,
				ModelAlias:              model.Alias,
				Status:                  health.Status,
				RetryCount:              health.RetryCount,
				NextAvailableAt:         health.NextAvailableAt,
				BackoffDuration:         health.BackoffDuration,
				LastError:               health.LastError,
				LastErrorCode:           health.LastErrorCode,
				LastErrorMessage:        health.LastErrorMessage,
				LastStructuredErrorCode: health.LastStructuredErrorCode,
				LastHTTPStatus:          health.LastHTTPStatus,
				LastErrorFrom:           health.LastErrorFrom,
				LastCauseMessage:        health.LastCauseMessage,
				LastCheckAt:             health.LastCheckAt,
				LastSuccessAt:           health.LastSuccessAt,
				SuccessCount:            health.SuccessCount,
				ErrorCount:              health.ErrorCount,
			})
		}
	}

	logger.Debug("成功获取模型健康列表",
		"total", total,
		"page", page,
		"page_size", pageSize,
		"item_count", len(items))

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
	logger := s.logger.With("operation", "list_issues")
	logger.Debug("开始获取异常资源列表")

	// 从存储中获取所有 Unavailable 状态的健康记录
	unavailableHealths := s.storage.GetByStatus(types.HealthStatusUnavailable)

	// 如果没有异常记录，返回空列表
	if len(unavailableHealths) == 0 {
		logger.Debug("没有找到任何异常资源")
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
	for _, health := range unavailableHealths {
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

	// 查询密钥信息及其 PlatformID
	keyMap := make(map[uint]string)
	keyPlatformMap := make(map[uint]uint) // 存储密钥 ID -> 平台 ID 的映射
	if len(keyIDs) > 0 {
		keys, err := q.APIKey.WithContext(ctx).
			Select(q.APIKey.ID, q.APIKey.Value, q.APIKey.PlatformID).
			Where(q.APIKey.ID.In(keyIDs...)).
			Find()
		if err != nil {
			logger.Error("查询密钥信息失败", "error", err)
			return nil, fmt.Errorf("查询密钥信息失败：%w", err)
		}
		for _, key := range keys {
			keyMap[key.ID] = key.Value
			keyPlatformMap[key.ID] = key.PlatformID
		}
	}

	// 查询模型信息及其 PlatformID
	modelMap := make(map[uint]string)
	modelPlatformMap := make(map[uint]uint) // 存储模型 ID -> 平台 ID 的映射
	if len(modelIDs) > 0 {
		models, err := q.Model.WithContext(ctx).
			Select(q.Model.ID, q.Model.Name, q.Model.PlatformID).
			Where(q.Model.ID.In(modelIDs...)).
			Find()
		if err != nil {
			logger.Error("查询模型信息失败", "error", err)
			return nil, fmt.Errorf("查询模型信息失败：%w", err)
		}
		for _, model := range models {
			modelMap[model.ID] = model.Name
			modelPlatformMap[model.ID] = model.PlatformID
		}
	}

	// 收集所有需要查询的平台 ID
	allPlatformIDs := make(map[uint]bool)
	for _, pid := range platformIDs {
		allPlatformIDs[pid] = true
	}
	for _, pid := range keyPlatformMap {
		allPlatformIDs[pid] = true
	}
	for _, pid := range modelPlatformMap {
		allPlatformIDs[pid] = true
	}

	// 转换为切片用于查询
	needQueryPlatformIDs := make([]uint, 0, len(allPlatformIDs))
	for pid := range allPlatformIDs {
		needQueryPlatformIDs = append(needQueryPlatformIDs, pid)
	}

	// 查询平台信息
	platformMap := make(map[uint]string)
	if len(needQueryPlatformIDs) > 0 {
		platforms, err := q.Platform.WithContext(ctx).
			Select(q.Platform.ID, q.Platform.Name).
			Where(q.Platform.ID.In(needQueryPlatformIDs...)).
			Find()
		if err != nil {
			logger.Error("查询平台信息失败", "error", err)
			return nil, fmt.Errorf("查询平台信息失败：%w", err)
		}
		for _, platform := range platforms {
			platformMap[platform.ID] = platform.Name
		}
	}

	// 组装响应数据
	items := make([]IssueItem, 0, len(unavailableHealths))
	for _, health := range unavailableHealths {
		var resourceName string
		var platformName string // 平台名称变量

		switch health.ResourceType {
		case types.ResourceTypePlatform:
			resourceName = platformMap[health.ResourceID]
		case types.ResourceTypeAPIKey:
			resourceName = keyMap[health.ResourceID]
			// 获取密钥所属的平台名称
			if platformID, exists := keyPlatformMap[health.ResourceID]; exists {
				platformName = platformMap[platformID]
			}
		case types.ResourceTypeModel:
			resourceName = modelMap[health.ResourceID]
			// 获取模型所属的平台名称
			if platformID, exists := modelPlatformMap[health.ResourceID]; exists {
				platformName = platformMap[platformID]
			}
		}

		// 只添加能找到资源名称的项
		if resourceName != "" {
			items = append(items, IssueItem{
				ResourceType: health.ResourceType,
				ResourceID:   health.ResourceID,
				ResourceName: resourceName,
				PlatformName: &platformName, // 填充平台名称
				Status:       health.Status,
				LastCheckAt:  health.LastCheckAt,
				LastError:    health.LastError,
			})
		}
	}

	logger.Debug("成功获取异常资源列表", "item_count", len(items))

	return &IssuesListResponse{
		Items: items,
	}, nil
}
