package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gen/field"
)

// AddEndpointToPlatform 实现为指定平台添加新端点
func (s *service) AddEndpointToPlatform(ctx context.Context, platformId uint, endpoint types.Endpoint) (*types.Endpoint, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(platformId)))
	logger.Debug("开始为平台添加端点")

	// 检查平台是否存在
	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台不存在", slog.Any("error", err))
		return nil, err
	}

	endpoint.ID = 0
	endpoint.PlatformID = platformId

	err := query.Q.Transaction(func(tx *query.Query) error {
		if err := tx.Endpoint.WithContext(ctx).Create(&endpoint); err != nil {
			logger.Error("创建端点失败", slog.Any("error", err))
			return fmt.Errorf("创建端点失败：%w", err)
		}
		if endpoint.IsDefault {
			if err := s.validatePlatformDefaultUniqueWithQuery(ctx, tx, platformId); err != nil {
				logger.Warn("默认端点校验失败", slog.Any("error", err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("创建端点事务失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功为平台添加端点", slog.Uint64("endpoint_id", uint64(endpoint.ID)))
	return &endpoint, nil
}

// BatchAddEndpointsToPlatform 实现批量为指定平台添加端点（原子性操作）
func (s *service) BatchAddEndpointsToPlatform(ctx context.Context, platformId uint, endpoints []types.Endpoint) ([]*types.Endpoint, error) {
	logger := s.logger.With(
		slog.Uint64("platform_id", uint64(platformId)),
		slog.Int("endpoint_count", len(endpoints)),
	)
	logger.Debug("开始批量为平台添加端点")

	if len(endpoints) == 0 {
		logger.Warn("未提供任何端点")
		return nil, fmt.Errorf("必须至少提供一个端点")
	}

	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台不存在", slog.Any("error", err))
		return nil, err
	}

	createdEndpoints := make([]*types.Endpoint, 0, len(endpoints))
	containsDefault := false
	for i := range endpoints {
		endpoints[i].ID = 0
		endpoints[i].PlatformID = platformId
		if endpoints[i].IsDefault {
			containsDefault = true
		}
		createdEndpoints = append(createdEndpoints, &endpoints[i])
	}

	batchSize := len(createdEndpoints)
	if batchSize > 100 {
		batchSize = 100
	}

	err := query.Q.Transaction(func(tx *query.Query) error {
		if err := tx.Endpoint.WithContext(ctx).CreateInBatches(createdEndpoints, batchSize); err != nil {
			logger.Error("创建端点失败", slog.Any("error", err))
			return fmt.Errorf("创建端点失败：%w", err)
		}
		if containsDefault {
			if err := s.validatePlatformDefaultUniqueWithQuery(ctx, tx, platformId); err != nil {
				logger.Warn("默认端点校验失败", slog.Any("error", err))
				return err
			}
		}
		return nil
	})

	if err != nil {
		logger.Error("批量创建端点事务失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功批量为平台添加端点", slog.Int("created_count", len(createdEndpoints)))
	return createdEndpoints, nil
}

// GetEndpointsByPlatform 实现获取指定平台的所有端点列表
func (s *service) GetEndpointsByPlatform(ctx context.Context, platformId uint) ([]*types.Endpoint, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(platformId)))
	logger.Debug("开始获取平台端点列表")

	// 检查平台是否存在
	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台不存在", slog.Any("error", err))
		return nil, err
	}

	endpoints, err := query.Q.Endpoint.WithContext(ctx).
		Where(query.Q.Endpoint.PlatformID.Eq(platformId)).
		Find()
	if err != nil {
		logger.Error("获取端点列表失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取平台 ID 为 %d 的端点失败：%w", platformId, err)
	}

	logger.Info("成功获取平台端点列表", slog.Int("count", len(endpoints)))
	return endpoints, nil
}

// GetEndpoint 实现获取指定端点详情
func (s *service) GetEndpoint(ctx context.Context, endpointId uint) (*types.Endpoint, error) {
	logger := s.logger.With(slog.Uint64("endpoint_id", uint64(endpointId)))
	logger.Debug("开始获取端点详情")

	endpoint, err := s.getEndpointByID(ctx, endpointId)
	if err != nil {
		logger.Warn("端点不存在或查询失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功获取端点详情")
	return endpoint, nil
}

// UpdateEndpoint 实现更新指定端点
func (s *service) UpdateEndpoint(ctx context.Context, endpointId uint, endpoint types.Endpoint) (*types.Endpoint, error) {
	logger := s.logger.With(slog.Uint64("endpoint_id", uint64(endpointId)))
	logger.Debug("开始更新端点")

	var updatedEndpoint *types.Endpoint
	err := query.Q.Transaction(func(tx *query.Query) error {
		existing, err := tx.Endpoint.WithContext(ctx).Where(tx.Endpoint.ID.Eq(endpointId)).First()
		if err != nil {
			logger.Warn("端点不存在或查询失败", slog.Any("error", err))
			return err
		}

		updates := make(map[string]interface{})
		selectFields := make([]string, 0, 5)
		payload := types.Endpoint{}
		if endpoint.EndpointType != "" {
			updates["endpoint_type"] = endpoint.EndpointType
			payload.EndpointType = endpoint.EndpointType
			selectFields = append(selectFields, "endpoint_type")
		}
		if endpoint.EndpointVariant != "" {
			updates["endpoint_variant"] = endpoint.EndpointVariant
			payload.EndpointVariant = endpoint.EndpointVariant
			selectFields = append(selectFields, "endpoint_variant")
		}
		if endpoint.Path != "" {
			updates["path"] = endpoint.Path
			payload.Path = endpoint.Path
			selectFields = append(selectFields, "path")
		}
		if endpoint.CustomHeaders != nil {
			updates["custom_headers"] = endpoint.CustomHeaders
			payload.CustomHeaders = endpoint.CustomHeaders
			selectFields = append(selectFields, "custom_headers")
		}
		if endpoint.IsDefault {
			updates["is_default"] = true
			payload.IsDefault = true
			selectFields = append(selectFields, "is_default")
		}

		if len(updates) > 0 {
			selectExprs := endpointSelectFields(selectFields)
			result, err := tx.Endpoint.WithContext(ctx).
				Select(selectExprs...).
				Where(tx.Endpoint.ID.Eq(endpointId)).
				Updates(payload)
			if err != nil {
				logger.Error("更新端点失败", slog.Any("error", err))
				return fmt.Errorf("更新 ID 为 %d 的端点失败：%w", endpointId, err)
			}
			if result.RowsAffected == 0 {
				logger.Warn("端点不存在")
				return fmt.Errorf("未找到 ID 为 %d 的端点", endpointId)
			}
		}

		if endpoint.IsDefault {
			if err := s.validatePlatformDefaultUniqueWithQuery(ctx, tx, existing.PlatformID); err != nil {
				logger.Warn("默认端点校验失败", slog.Any("error", err))
				return err
			}
		}

		updatedEndpoint, err = tx.Endpoint.WithContext(ctx).Where(tx.Endpoint.ID.Eq(endpointId)).First()
		if err != nil {
			logger.Error("获取更新后的端点失败", slog.Any("error", err))
			return err
		}

		return nil
	})
	if err != nil {
		logger.Error("更新端点事务失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功更新端点")
	return updatedEndpoint, nil
}

// BatchUpdateEndpoints 实现批量更新指定平台的端点（原子性操作）
func (s *service) BatchUpdateEndpoints(ctx context.Context, platformId uint, updateItems []EndpointUpdateItem) ([]*types.Endpoint, error) {
	logger := s.logger.With(
		slog.Uint64("platform_id", uint64(platformId)),
		slog.Int("endpoint_count", len(updateItems)),
	)
	logger.Debug("开始批量更新端点")

	if len(updateItems) == 0 {
		logger.Warn("未提供任何更新项")
		return nil, fmt.Errorf("必须至少提供一个端点更新项")
	}

	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台验证失败", slog.Any("error", err))
		return nil, err
	}

	endpointIds := make([]uint, 0, len(updateItems))
	itemByID := make(map[uint]*EndpointUpdateItem, len(updateItems))
	for i := range updateItems {
		item := &updateItems[i]
		if item.ID == 0 {
			logger.Warn("端点更新项缺少 ID")
			return nil, fmt.Errorf("端点更新项缺少 ID")
		}
		if _, exists := itemByID[item.ID]; exists {
			return nil, fmt.Errorf("端点更新项包含重复 ID：%d", item.ID)
		}
		endpointIds = append(endpointIds, item.ID)
		itemByID[item.ID] = item
	}

	existingEndpoints, err := query.Q.Endpoint.WithContext(ctx).
		Where(query.Q.Endpoint.ID.In(endpointIds...), query.Q.Endpoint.PlatformID.Eq(platformId)).
		Find()
	if err != nil {
		logger.Error("批量查询端点失败", slog.Any("error", err))
		return nil, fmt.Errorf("批量查询端点失败：%w", err)
	}
	if len(existingEndpoints) != len(endpointIds) {
		foundIds := make(map[uint]struct{}, len(existingEndpoints))
		for _, endpoint := range existingEndpoints {
			foundIds[endpoint.ID] = struct{}{}
		}
		var missing []uint
		for _, id := range endpointIds {
			if _, ok := foundIds[id]; !ok {
				missing = append(missing, id)
			}
		}
		return nil, fmt.Errorf("以下端点不存在或不属于平台：%v", missing)
	}

	var updatedEndpoints []*types.Endpoint
	groups := make(map[string]endpointUpdateGroup)
	groupIDs := make(map[string][]uint)
	containsDefault := false
	for _, endpoint := range existingEndpoints {
		item := itemByID[endpoint.ID]
		updates := make(map[string]interface{})
		selectFields := make([]string, 0, 5)
		payload := types.Endpoint{}
		if item.EndpointType != nil {
			updates["endpoint_type"] = *item.EndpointType
			payload.EndpointType = *item.EndpointType
			selectFields = append(selectFields, "endpoint_type")
		}
		if item.EndpointVariant != nil {
			updates["endpoint_variant"] = *item.EndpointVariant
			payload.EndpointVariant = *item.EndpointVariant
			selectFields = append(selectFields, "endpoint_variant")
		}
		if item.Path != nil {
			updates["path"] = *item.Path
			payload.Path = *item.Path
			selectFields = append(selectFields, "path")
		}
		if item.CustomHeaders != nil {
			updates["custom_headers"] = *item.CustomHeaders
			payload.CustomHeaders = *item.CustomHeaders
			selectFields = append(selectFields, "custom_headers")
		}
		if item.IsDefault != nil {
			updates["is_default"] = *item.IsDefault
			payload.IsDefault = *item.IsDefault
			selectFields = append(selectFields, "is_default")
			containsDefault = true
		}
		if len(updates) == 0 {
			continue
		}
		signature, err := buildEndpointUpdateSignature(updates)
		if err != nil {
			return nil, err
		}
		if _, exists := groups[signature]; !exists {
			groups[signature] = endpointUpdateGroup{payload: payload, selectFields: selectFields}
		}
		groupIDs[signature] = append(groupIDs[signature], endpoint.ID)
	}

	err = query.Q.Transaction(func(tx *query.Query) error {
		for signature, group := range groups {
			ids := groupIDs[signature]
			if len(ids) == 0 {
				continue
			}
			selectExprs := endpointSelectFields(group.selectFields)
			if _, err := tx.Endpoint.WithContext(ctx).
				Select(selectExprs...).
				Where(tx.Endpoint.ID.In(ids...)).
				Updates(group.payload); err != nil {
				return fmt.Errorf("批量更新端点失败：%w", err)
			}
		}

		if containsDefault {
			if err := s.validatePlatformDefaultUniqueWithQuery(ctx, tx, platformId); err != nil {
				logger.Warn("默认端点校验失败", slog.Any("error", err))
				return err
			}
		}

		updatedEndpoints, err = tx.Endpoint.WithContext(ctx).
			Where(tx.Endpoint.ID.In(endpointIds...)).
			Find()
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("批量更新端点事务失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功批量更新端点", slog.Int("updated_count", len(updatedEndpoints)))
	return updatedEndpoints, nil
}

type endpointUpdateGroup struct {
	payload      types.Endpoint
	selectFields []string
}

func endpointSelectFields(names []string) []field.Expr {
	if len(names) == 0 {
		return nil
	}
	selects := make([]field.Expr, 0, len(names))
	for _, name := range names {
		if expr, ok := query.Q.Endpoint.GetFieldByName(name); ok {
			selects = append(selects, expr)
		}
	}
	return selects
}

func buildEndpointUpdateSignature(updates map[string]interface{}) (string, error) {
	if len(updates) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	buffer := make([]byte, 0, 128)
	for _, key := range keys {
		valueSignature, err := buildEndpointUpdateValueSignature(updates[key])
		if err != nil {
			return "", err
		}
		buffer = append(buffer, key...)
		buffer = append(buffer, '=')
		buffer = append(buffer, valueSignature...)
		buffer = append(buffer, ';')
	}

	return string(buffer), nil
}

func buildEndpointUpdateValueSignature(value interface{}) (string, error) {
	switch typed := value.(type) {
	case map[string]string:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		buffer := make([]byte, 0, 128)
		for _, key := range keys {
			buffer = append(buffer, key...)
			buffer = append(buffer, '=')
			buffer = append(buffer, typed[key]...)
			buffer = append(buffer, ';')
		}
		return string(buffer), nil
	case *map[string]string:
		if typed == nil {
			return "<nil>", nil
		}
		return buildEndpointUpdateValueSignature(*typed)
	case string:
		return typed, nil
	case bool:
		if typed {
			return "true", nil
		}
		return "false", nil
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return "", fmt.Errorf("序列化更新内容失败：%w", err)
		}
		return string(data), nil
	}
}

// DeleteEndpoint 实现删除指定端点
func (s *service) DeleteEndpoint(ctx context.Context, endpointId uint) error {
	logger := s.logger.With(slog.Uint64("endpoint_id", uint64(endpointId)))
	logger.Debug("开始删除端点")

	endpoint, err := s.getEndpointByID(ctx, endpointId)
	if err != nil {
		logger.Warn("端点不存在或查询失败", slog.Any("error", err))
		return err
	}

	result, err := query.Q.Endpoint.WithContext(ctx).Where(query.Q.Endpoint.ID.Eq(endpointId)).Delete()
	if err != nil {
		logger.Error("删除端点失败", slog.Any("error", err))
		return fmt.Errorf("删除 ID 为 %d 的端点失败：%w", endpointId, err)
	}
	if result.RowsAffected == 0 {
		logger.Warn("端点不存在")
		return fmt.Errorf("未找到 ID 为 %d 的端点", endpointId)
	}

	if endpoint.IsDefault {
		if err := s.ensurePlatformDefaultExists(ctx, endpoint.PlatformID); err != nil {
			logger.Warn("修复默认端点失败", slog.Any("error", err))
			return err
		}
	}

	logger.Info("成功删除端点")
	return nil
}
