package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// AddModelToPlatform 实现为指定平台添加新模型
func (s *service) AddModelToPlatform(ctx context.Context, platformId uint, model types.Model) (*types.Model, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(platformId)))
	logger.Debug("开始为平台添加模型")

	// 验证平台是否存在
	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台不存在", slog.Any("error", err))
		return nil, err
	}

	// 验证并获取有效的 API 密钥
	validKeys, err := s.validateAndGetAPIKeys(ctx, platformId, model.APIKeys, logger)
	if err != nil {
		return nil, err
	}

	// 保存密钥引用以便后续关联
	model.APIKeys = make([]types.APIKey, len(validKeys))
	for i, key := range validKeys {
		model.APIKeys[i] = *key
	}

	// 设置模型的平台 ID
	model.PlatformID = platformId

	// 创建模型（GORM 会自动处理多对多关系）
	model.ID = 0
	if err := query.Q.Model.WithContext(ctx).Create(&model); err != nil {
		logger.Error("创建模型失败", slog.Any("error", err))
		return nil, fmt.Errorf("创建模型失败：%w", err)
	}

	logger.Info("成功为平台添加模型",
		slog.String("model_name", model.Name),
		slog.Uint64("model_id", uint64(model.ID)),
		slog.Int("api_key_count", len(model.APIKeys)))
	return &model, nil
}

// BatchAddModelsToPlatform 实现批量为指定平台添加模型（原子性操作）
func (s *service) BatchAddModelsToPlatform(ctx context.Context, platformId uint, models []types.Model) ([]*types.Model, error) {
	logger := s.logger.With(
		slog.Uint64("platform_id", uint64(platformId)),
		slog.Int("model_count", len(models)),
	)
	logger.Debug("开始批量为平台添加模型")

	// 基本参数验证
	if len(models) == 0 {
		logger.Warn("未提供任何模型")
		return nil, fmt.Errorf("必须至少提供一个模型")
	}

	// 验证平台是否存在
	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台验证失败", slog.Any("error", err))
		return nil, err
	}

	// 批量验证所有 API 密钥是否存在且属于该平台
	if err := s.batchValidateAPIKeys(ctx, platformId, models, logger); err != nil {
		logger.Error("API 密钥验证失败", slog.Any("error", err))
		return nil, err
	}

	// 在事务中批量创建模型
	var createdModels []*types.Model
	err := query.Q.Transaction(func(tx *query.Query) error {
		for i := range models {
			model := &models[i]
			model.PlatformID = platformId
			model.ID = 0

			// 创建模型（GORM 会自动处理多对多关系）
			if err := tx.Model.WithContext(ctx).Create(model); err != nil {
				logger.Error("创建模型失败",
					slog.String("model_name", model.Name),
					slog.Any("error", err))
				return fmt.Errorf("创建模型 '%s' 失败：%w", model.Name, err)
			}

			createdModels = append(createdModels, model)
		}
		return nil
	})

	if err != nil {
		logger.Error("批量创建模型事务失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功批量为平台添加模型",
		slog.Int("created_count", len(createdModels)))
	return createdModels, nil
}

// GetModelsByPlatform 实现获取指定平台的所有模型列表
func (s *service) GetModelsByPlatform(ctx context.Context, providerId uint) ([]*types.Model, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(providerId)))
	logger.Debug("开始获取平台的模型列表")

	// 验证平台是否存在
	if err := s.validatePlatformExists(ctx, providerId); err != nil {
		logger.Warn("平台不存在", slog.Any("error", err))
		return nil, err
	}

	// 获取模型列表（预加载关联的 API 密钥）
	models, err := query.Q.Model.WithContext(ctx).
		Preload(query.Q.Model.APIKeys).
		Where(query.Q.Model.PlatformID.Eq(providerId)).
		Find()
	if err != nil {
		logger.Error("获取模型列表失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取平台 ID 为 %d 的模型失败：%w", providerId, err)
	}

	logger.Info("成功获取平台的模型列表", slog.Int("count", len(models)))
	return models, nil
}

// UpdateModel 实现更新指定模型信息
func (s *service) UpdateModel(ctx context.Context, modelId uint, model types.Model) (*types.Model, error) {
	logger := s.logger.With(slog.Uint64("model_id", uint64(modelId)))
	logger.Debug("开始更新模型")

	// 查询现有模型
	existingModel, err := s.getModelByID(ctx, modelId)
	if err != nil {
		logger.Warn("查询模型失败", slog.Any("error", err))
		return nil, err
	}

	// 如果提供了 API 密钥列表，则更新关联关系
	if len(model.APIKeys) > 0 {
		validKeys, err := s.validateAndGetAPIKeys(ctx, existingModel.PlatformID, model.APIKeys, logger)
		if err != nil {
			return nil, err
		}

		// 使用 Association 的 Replace 方法更新多对多关系
		apiKeyPtrs := make([]*types.APIKey, len(validKeys))
		copy(apiKeyPtrs, validKeys)
		if err := query.Q.Model.APIKeys.Model(existingModel).Replace(apiKeyPtrs...); err != nil {
			logger.Error("更新模型密钥关联失败", slog.Any("error", err))
			return nil, fmt.Errorf("更新模型密钥关联失败：%w", err)
		}

		logger.Info("成功更新模型密钥关联", slog.Int("api_key_count", len(validKeys)))
	}

	// 更新模型的其他字段（排除 APIKeys 字段，因为已单独处理）
	if model.Name != "" || model.Alias != "" {
		result, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).Updates(model)
		if err != nil {
			logger.Error("更新模型字段失败", slog.Any("error", err))
			return nil, fmt.Errorf("更新 ID 为 %d 的模型失败：%w", modelId, err)
		}
		if result.RowsAffected == 0 {
			logger.Warn("模型更新无影响行")
		}
	}

	// 返回更新后的完整对象（包含关联的 API 密钥）
	updatedModel, err := s.getModelWithAPIKeys(ctx, modelId)
	if err != nil {
		logger.Error("获取更新后的模型失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功更新模型", slog.String("model_name", updatedModel.Name))
	return updatedModel, nil
}

// DeleteModel 实现删除指定模型
func (s *service) DeleteModel(ctx context.Context, modelId uint) error {
	logger := s.logger.With(slog.Uint64("model_id", uint64(modelId)))
	logger.Debug("开始删除模型")

	// 查询模型是否存在
	model, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("模型不存在")
			return fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
		}
		logger.Error("查询模型失败", slog.Any("error", err))
		return fmt.Errorf("查询模型失败：%w", err)
	}

	// 备份模型与密钥的关联关系
	var backupAPIKeys []*types.APIKey
	apiKeys, err := query.Q.Model.APIKeys.Model(model).Find()
	if err != nil {
		logger.Error("查询模型关联的密钥失败", slog.Any("error", err))
		return fmt.Errorf("查询模型关联的密钥失败：%w", err)
	}
	if len(apiKeys) > 0 {
		backupAPIKeys = apiKeys
		logger.Debug("备份模型关联关系", slog.Int("api_key_count", len(apiKeys)))
	}

	// 清理模型与密钥的多对多关联关系
	if len(backupAPIKeys) > 0 {
		if err := query.Q.Model.APIKeys.Model(model).Clear(); err != nil {
			logger.Error("清理模型与密钥的关联关系失败", slog.Any("error", err))
			return fmt.Errorf("清理模型与密钥的关联关系失败：%w", err)
		}
		logger.Debug("成功清理模型与密钥的关联关系", slog.Int("api_key_count", len(backupAPIKeys)))
	}

	// 删除模型
	result, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).Delete()
	if err != nil {
		logger.Error("删除模型失败", slog.Any("error", err))

		// 删除失败，恢复关联关系
		if len(backupAPIKeys) > 0 {
			logger.Warn("删除失败，开始恢复关联关系")
			if restoreErr := query.Q.Model.APIKeys.Model(model).Append(backupAPIKeys...); restoreErr != nil {
				logger.Error("恢复模型与密钥的关联关系失败", slog.Any("error", restoreErr))
			} else {
				logger.Debug("成功恢复模型与密钥的关联关系", slog.Int("api_key_count", len(backupAPIKeys)))
			}
		}

		return fmt.Errorf("删除 ID 为 %d 的模型失败：%w", modelId, err)
	}
	if result.RowsAffected == 0 {
		logger.Warn("模型不存在")

		// 模型不存在，恢复关联关系
		if len(backupAPIKeys) > 0 {
			logger.Warn("模型不存在，开始恢复关联关系")
			if restoreErr := query.Q.Model.APIKeys.Model(model).Append(backupAPIKeys...); restoreErr != nil {
				logger.Error("恢复模型与密钥的关联关系失败", slog.Any("error", restoreErr))
			} else {
				logger.Debug("成功恢复模型与密钥的关联关系", slog.Int("api_key_count", len(backupAPIKeys)))
			}
		}

		return fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
	}

	logger.Info("成功删除模型")
	return nil
}

// BatchDeleteModels 实现批量删除指定平台的模型（原子性操作）
func (s *service) BatchDeleteModels(ctx context.Context, platformId uint, modelIds []uint) (int, error) {
	// modelAPIKeysBackup 关联关系备份结构
	type modelAPIKeysBackup struct {
		modelID uint
		apiKeys []*types.APIKey
	}

	logger := s.logger.With(
		slog.Uint64("platform_id", uint64(platformId)),
		slog.Int("model_count", len(modelIds)),
	)
	logger.Debug("开始批量删除模型")

	// 基本参数验证
	if len(modelIds) == 0 {
		logger.Warn("未提供任何模型 ID")
		return 0, fmt.Errorf("必须至少提供一个模型 ID")
	}

	// 验证平台是否存在
	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台验证失败", slog.Any("error", err))
		return 0, err
	}

	// 批量验证所有模型是否存在且属于该平台
	models, err := query.Q.Model.WithContext(ctx).
		Where(query.Q.Model.ID.In(modelIds...)).
		Find()
	if err != nil {
		logger.Error("批量查询模型失败", slog.Any("error", err))
		return 0, fmt.Errorf("批量查询模型失败：%w", err)
	}

	// 检查模型数量是否匹配
	if len(models) != len(modelIds) {
		logger.Warn("部分模型不存在",
			slog.Int("requested_count", len(modelIds)),
			slog.Int("found_count", len(models)))

		// 找出哪些模型不存在
		foundIds := make(map[uint]struct{}, len(models))
		for _, model := range models {
			foundIds[model.ID] = struct{}{}
		}

		var missingIds []uint
		for _, id := range modelIds {
			if _, exists := foundIds[id]; !exists {
				missingIds = append(missingIds, id)
			}
		}

		return 0, fmt.Errorf("以下模型不存在：%v", missingIds)
	}

	// 验证所有模型都属于指定平台
	for _, model := range models {
		if model.PlatformID != platformId {
			logger.Warn("模型不属于指定平台",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Uint64("model_platform_id", uint64(model.PlatformID)),
				slog.Uint64("expected_platform_id", uint64(platformId)))
			return 0, fmt.Errorf("模型 ID %d 不属于平台 ID %d", model.ID, platformId)
		}
	}

	// 备份模型与密钥的关联关系
	backups := make([]modelAPIKeysBackup, 0, len(models))
	logger.Debug("开始备份模型与密钥的关联关系")
	for _, model := range models {
		// 查询该模型关联的所有密钥
		apiKeys, err := query.Q.Model.APIKeys.Model(model).Find()
		if err != nil {
			logger.Error("查询模型关联的密钥失败",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Any("error", err))
			return 0, fmt.Errorf("查询模型 ID 为 %d 关联的密钥失败：%w", model.ID, err)
		}
		if len(apiKeys) > 0 {
			backups = append(backups, modelAPIKeysBackup{
				modelID: model.ID,
				apiKeys: apiKeys,
			})
			logger.Debug("备份模型关联关系",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Int("api_key_count", len(apiKeys)))
		}
	}
	logger.Debug("完成备份关联关系", slog.Int("backup_count", len(backups)))

	// 清理模型与密钥的多对多关联关系
	//
	// TODO：这里由于存在未知错误，导致该操作在事务内无法正常完成，
	// 因此采取暂时将其移动到事务外的临时方案。
	// Issue：https://github.com/go-gorm/gorm/issues/7649
	logger.Debug("开始清理模型与密钥的关联关系")
	for _, model := range models {
		count := query.Q.Model.APIKeys.Model(model).Count()
		if count == 0 {
			logger.Debug("模型没有关联密钥，跳过清理", slog.Uint64("model_id", uint64(model.ID)))
			continue
		}
		// 清理该模型与所有密钥的关联
		if err := query.Q.Model.APIKeys.Model(model).Clear(); err != nil {
			logger.Error("清理模型与密钥的关联关系失败",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Any("error", err))
			return 0, fmt.Errorf("清理模型 ID 为 %d 与密钥的关联关系失败：%w", model.ID, err)
		}
		logger.Debug("成功清理模型与密钥的关联关系",
			slog.Uint64("model_id", uint64(model.ID)),
			slog.Int64("api_key_count", count))
	}
	logger.Debug("成功清理所有模型与密钥的关联关系")

	// 在事务中批量删除模型
	var deletedCount int
	err = query.Q.Transaction(func(tx *query.Query) error {
		result, err := tx.Model.WithContext(ctx).
			Where(tx.Model.ID.In(modelIds...)).
			Delete()
		if err != nil {
			logger.Error("批量删除模型失败", slog.Any("error", err))
			return fmt.Errorf("批量删除模型失败：%w", err)
		}

		deletedCount = int(result.RowsAffected)
		return nil
	})

	if err != nil {
		// 事务失败，恢复关联关系备份
		logger.Warn("事务失败，开始恢复关联关系备份", slog.Any("error", err))
		for _, backup := range backups {
			model := &types.Model{ID: backup.modelID}
			if restoreErr := query.Q.Model.APIKeys.Model(model).Append(backup.apiKeys...); restoreErr != nil {
				logger.Error("恢复模型与密钥的关联关系失败",
					slog.Uint64("model_id", uint64(backup.modelID)),
					slog.Any("error", restoreErr))
			} else {
				logger.Debug("成功恢复模型与密钥的关联关系",
					slog.Uint64("model_id", uint64(backup.modelID)),
					slog.Int("api_key_count", len(backup.apiKeys)))
			}
		}
		logger.Debug("完成关联关系恢复")
		return 0, err
	}

	logger.Info("成功批量删除模型", slog.Int("deleted_count", deletedCount))
	return deletedCount, nil
}

// BatchUpdateModels 实现批量更新指定平台的模型（原子性操作）
func (s *service) BatchUpdateModels(ctx context.Context, platformId uint, updateItems []ModelUpdateItem) ([]*types.Model, error) {
	logger := s.logger.With(
		slog.Uint64("platform_id", uint64(platformId)),
		slog.Int("model_count", len(updateItems)),
	)
	logger.Debug("开始批量更新模型")

	// 基本参数验证
	if len(updateItems) == 0 {
		logger.Warn("未提供任何更新项")
		return nil, fmt.Errorf("必须至少提供一个模型更新项")
	}

	// 验证平台是否存在
	if err := s.validatePlatformExists(ctx, platformId); err != nil {
		logger.Warn("平台验证失败", slog.Any("error", err))
		return nil, err
	}

	// 批量验证所有模型是否存在且属于该平台
	if err := s.batchValidateModels(ctx, platformId, updateItems, logger); err != nil {
		logger.Error("模型验证失败", slog.Any("error", err))
		return nil, err
	}

	// 在事务中批量更新模型
	var updatedModels []*types.Model
	err := query.Q.Transaction(func(tx *query.Query) error {
		for i := range updateItems {
			item := &updateItems[i]
			itemLogger := logger.With(slog.Uint64("model_id", uint64(item.ID)))

			// 查询现有模型
			existingModel, err := s.getModelByID(ctx, item.ID)
			if err != nil {
				itemLogger.Error("查询模型失败", slog.Any("error", err))
				return err
			}

			// 如果提供了 API 密钥列表，则更新关联关系
			if len(item.APIKeys) > 0 {
				validKeys, err := s.validateAndGetAPIKeys(ctx, existingModel.PlatformID, item.APIKeys, itemLogger)
				if err != nil {
					return err
				}

				// 使用 Association 的 Replace 方法更新多对多关系
				apiKeyPtrs := make([]*types.APIKey, len(validKeys))
				copy(apiKeyPtrs, validKeys)
				if err := tx.Model.APIKeys.Model(existingModel).Replace(apiKeyPtrs...); err != nil {
					itemLogger.Error("更新模型密钥关联失败", slog.Any("error", err))
					return fmt.Errorf("更新模型 ID %d 的密钥关联失败：%w", item.ID, err)
				}

				itemLogger.Debug("成功更新模型密钥关联", slog.Int("api_key_count", len(validKeys)))
			}

			// 更新模型的其他字段（部分更新，只更新非空字段）
			needsUpdate := false
			updates := make(map[string]interface{})

			if item.Name != "" && item.Name != existingModel.Name {
				updates["name"] = item.Name
				needsUpdate = true
			}

			if item.Alias != "" && item.Alias != existingModel.Alias {
				updates["alias"] = item.Alias
				needsUpdate = true
			}

			if needsUpdate {
				result, err := tx.Model.WithContext(ctx).
					Where(tx.Model.ID.Eq(item.ID)).
					Updates(updates)
				if err != nil {
					itemLogger.Error("更新模型字段失败", slog.Any("error", err))
					return fmt.Errorf("更新模型 ID %d 的字段失败：%w", item.ID, err)
				}
				if result.RowsAffected == 0 {
					itemLogger.Warn("模型更新无影响行")
				}
				itemLogger.Debug("成功更新模型字段")
			}

			// 重新加载完整的模型数据（包含关联的 API 密钥）
			updatedModel, err := s.getModelWithAPIKeys(ctx, item.ID)
			if err != nil {
				itemLogger.Error("获取更新后的模型失败", slog.Any("error", err))
				return err
			}

			updatedModels = append(updatedModels, updatedModel)
		}
		return nil
	})

	if err != nil {
		logger.Error("批量更新模型事务失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功批量更新模型", slog.Int("updated_count", len(updatedModels)))
	return updatedModels, nil
}

// batchValidateModels 批量验证模型是否存在且属于指定平台
func (s *service) batchValidateModels(ctx context.Context, platformId uint, updateItems []ModelUpdateItem, logger *slog.Logger) error {
	// 提取所有模型 ID
	modelIds := make([]uint, len(updateItems))
	for i, item := range updateItems {
		modelIds[i] = item.ID
	}

	// 批量查询所有模型
	models, err := query.Q.Model.WithContext(ctx).
		Where(query.Q.Model.ID.In(modelIds...)).
		Find()
	if err != nil {
		logger.Error("批量查询模型失败", slog.Any("error", err))
		return fmt.Errorf("批量查询模型失败：%w", err)
	}

	// 检查模型数量是否匹配
	if len(models) != len(modelIds) {
		logger.Warn("部分模型不存在",
			slog.Int("requested_count", len(modelIds)),
			slog.Int("found_count", len(models)))

		// 找出哪些模型不存在
		foundIds := make(map[uint]struct{}, len(models))
		for _, model := range models {
			foundIds[model.ID] = struct{}{}
		}

		var missingIds []uint
		for _, id := range modelIds {
			if _, exists := foundIds[id]; !exists {
				missingIds = append(missingIds, id)
			}
		}

		return fmt.Errorf("以下模型不存在：%v", missingIds)
	}

	// 验证所有模型都属于指定平台
	for _, model := range models {
		if model.PlatformID != platformId {
			logger.Warn("模型不属于指定平台",
				slog.Uint64("model_id", uint64(model.ID)),
				slog.Uint64("model_platform_id", uint64(model.PlatformID)),
				slog.Uint64("expected_platform_id", uint64(platformId)))
			return fmt.Errorf("模型 ID %d 不属于平台 ID %d", model.ID, platformId)
		}
	}

	logger.Debug("成功验证所有模型", slog.Int("validated_count", len(models)))
	return nil
}
