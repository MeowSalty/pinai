package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// CreatePlatform 实现创建平台
func (s *service) CreatePlatform(ctx context.Context, platform types.Platform) (*types.Platform, error) {
	logger := s.logger.With(
		slog.String("operation", "create_platform"),
		slog.String("platform_name", platform.Name),
	)
	logger.Debug("开始创建平台")

	if s.platformControlRepo == nil {
		return nil, fmt.Errorf("创建平台失败：平台控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("创建平台失败：事务执行器未初始化")
	}

	platform.ID = 0
	err := s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		if innerErr := s.platformControlRepo.CreatePlatform(txCtx, &platform); innerErr != nil {
			return innerErr
		}
		return nil
	})
	if err != nil {
		logger.Error("创建平台失败", slog.Any("error", err))
		_ = s.logPlatformControlAudit(ctx, platform.ID, "platform.create", "failed", fmt.Sprintf("创建平台失败：%v", err))
		return nil, fmt.Errorf("创建平台失败：%w", err)
	}

	logger.Info("成功创建平台", slog.Uint64("platform_id", uint64(platform.ID)))
	_ = s.logPlatformControlAudit(ctx, platform.ID, "platform.create", "success", "创建平台成功")
	return &platform, nil
}

// GetPlatforms 实现获取平台列表
func (s *service) GetPlatforms(ctx context.Context) ([]*types.Platform, error) {
	s.logger.Debug("开始获取平台列表")

	platforms, err := query.Q.Platform.WithContext(ctx).
		Preload(query.Q.Platform.Endpoints).
		Find()
	if err != nil {
		s.logger.Error("获取平台列表失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取平台列表失败：%w", err)
	}

	s.logger.Info("成功获取平台列表", slog.Int("count", len(platforms)))
	return platforms, nil
}

// GetPlatformResourceCounts 批量获取各平台的密钥数和模型数
func (s *service) GetPlatformResourceCounts(ctx context.Context) (keyCounts, modelCounts map[uint]int64, err error) {
	s.logger.Debug("开始获取平台资源统计")

	// 按 platform_id 分组统计密钥数
	type countResult struct {
		PlatformID uint  `gorm:"column:platform_id"`
		Count      int64 `gorm:"column:count"`
	}

	var keyResults []countResult
	err = query.Q.APIKey.WithContext(ctx).UnderlyingDB().
		Table("api_keys").
		Select("platform_id, COUNT(*) as count").
		Group("platform_id").
		Scan(&keyResults).Error
	if err != nil {
		s.logger.Error("统计平台密钥数失败", slog.Any("error", err))
		return nil, nil, fmt.Errorf("统计平台密钥数失败：%w", err)
	}

	// 按 platform_id 分组统计模型数
	var modelResults []countResult
	err = query.Q.Model.WithContext(ctx).UnderlyingDB().
		Table("models").
		Select("platform_id, COUNT(*) as count").
		Group("platform_id").
		Scan(&modelResults).Error
	if err != nil {
		s.logger.Error("统计平台模型数失败", slog.Any("error", err))
		return nil, nil, fmt.Errorf("统计平台模型数失败：%w", err)
	}

	keyCounts = make(map[uint]int64, len(keyResults))
	for _, r := range keyResults {
		keyCounts[r.PlatformID] = r.Count
	}

	modelCounts = make(map[uint]int64, len(modelResults))
	for _, r := range modelResults {
		modelCounts[r.PlatformID] = r.Count
	}

	s.logger.Debug("成功获取平台资源统计",
		slog.Int("key_groups", len(keyCounts)),
		slog.Int("model_groups", len(modelCounts)))
	return keyCounts, modelCounts, nil
}

// GetPlatform 实现获取指定平台详情
func (s *service) GetPlatform(ctx context.Context, id uint) (*types.Platform, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(id)))
	logger.Debug("开始获取平台详情")

	platform, err := s.getPlatformByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			logger.Warn("平台不存在")
		} else {
			logger.Error("获取平台详情失败", slog.Any("error", err))
		}
		return nil, err
	}

	logger.Info("成功获取平台详情", slog.String("platform_name", platform.Name))
	return platform, nil
}

// UpdatePlatform 实现更新平台信息
func (s *service) UpdatePlatform(ctx context.Context, id uint, platform types.Platform) (*types.Platform, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(id)))
	logger.Debug("开始更新平台")

	if s.platformControlRepo == nil {
		return nil, fmt.Errorf("更新平台失败：平台控制仓储未初始化")
	}
	if s.controlTx == nil {
		return nil, fmt.Errorf("更新平台失败：事务执行器未初始化")
	}

	var updatedPlatform *types.Platform
	err := s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		rowsAffected, innerErr := s.platformControlRepo.UpdatePlatform(txCtx, id, platform)
		if innerErr != nil {
			return innerErr
		}
		if rowsAffected == 0 {
			return fmt.Errorf("未找到 ID 为 %d 的平台：%w", id, ErrResourceNotFound)
		}

		updatedPlatform, innerErr = s.platformControlRepo.GetPlatform(txCtx, id)
		if innerErr != nil {
			return innerErr
		}

		return nil
	})
	if err != nil {
		logger.Error("更新平台失败", slog.Any("error", err))
		_ = s.logPlatformControlAudit(ctx, id, "platform.update", "failed", fmt.Sprintf("更新平台失败：%v", err))
		return nil, fmt.Errorf("更新 ID 为 %d 的平台失败：%w", id, err)
	}

	logger.Info("成功更新平台", slog.String("platform_name", updatedPlatform.Name))
	_ = s.logPlatformControlAudit(ctx, id, "platform.update", "success", "更新平台成功")
	return updatedPlatform, nil
}

func (s *service) logPlatformControlAudit(ctx context.Context, platformID uint, action, result, detail string) error {
	if s.controlAudit == nil {
		return nil
	}

	return s.controlAudit.Log(ctx, ControlAuditEvent{
		Action:     action,
		Resource:   "platform",
		ResourceID: platformID,
		Result:     result,
		Detail:     detail,
	})
}

// DeletePlatform 实现删除平台（包括其关联的模型、密钥及关联关系）
func (s *service) DeletePlatform(ctx context.Context, id uint) error {
	// apiKeyModelsBackup 关联关系备份结构
	type apiKeyModelsBackup struct {
		apiKeyID uint
		models   []*types.Model
	}

	logger := s.logger.With(slog.Uint64("platform_id", uint64(id)))
	logger.Debug("开始删除平台")
	if s.platformControlRepo == nil {
		return fmt.Errorf("删除平台失败：平台控制仓储未初始化")
	}
	if s.controlTx == nil {
		return fmt.Errorf("删除平台失败：事务执行器未初始化")
	}

	// 检查平台是否存在
	exists, err := s.platformControlRepo.ExistsPlatform(ctx, id)
	if err != nil {
		logger.Error("检查平台是否存在失败", slog.Any("error", err))
		_ = s.logPlatformControlAudit(ctx, id, "platform.delete", "failed", fmt.Sprintf("检查平台是否存在失败：%v", err))
		return fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		logger.Warn("平台不存在")
		return fmt.Errorf("未找到 ID 为 %d 的平台：%w", id, ErrResourceNotFound)
	}

	// 查询平台下的所有密钥
	apiKeys, err := s.platformControlRepo.ListAPIKeysByPlatform(ctx, id)
	if err != nil {
		logger.Error("查询平台关联的密钥失败", slog.Any("error", err))
		_ = s.logPlatformControlAudit(ctx, id, "platform.delete", "failed", fmt.Sprintf("查询平台关联的密钥失败：%v", err))
		return fmt.Errorf("查询平台关联的密钥失败：%w", err)
	}
	logger.Debug("查询到关联密钥", slog.Int("apikey_count", len(apiKeys)))

	// 备份密钥与模型的关联关系
	backups := make([]apiKeyModelsBackup, 0, len(apiKeys))
	logger.Debug("开始备份密钥与模型的关联关系")
	for _, key := range apiKeys {
		// 查询该密钥关联的模型数量
		count, err := s.platformControlRepo.CountModelsByAPIKey(ctx, key.ID)
		if err != nil {
			logger.Error("统计密钥关联的模型数量失败",
				slog.Uint64("apikey_id", uint64(key.ID)),
				slog.Any("error", err))
			_ = s.logPlatformControlAudit(ctx, id, "platform.delete", "failed", fmt.Sprintf("统计密钥 ID 为 %d 关联模型数量失败：%v", key.ID, err))
			return fmt.Errorf("统计密钥 ID 为 %d 关联模型数量失败：%w", key.ID, err)
		}
		if count == 0 {
			continue
		}

		// 查询该密钥关联的所有模型
		models, err := s.platformControlRepo.ListModelsByAPIKey(ctx, key.ID)
		if err != nil {
			logger.Error("查询密钥关联的模型失败",
				slog.Uint64("apikey_id", uint64(key.ID)),
				slog.Any("error", err))
			_ = s.logPlatformControlAudit(ctx, id, "platform.delete", "failed", fmt.Sprintf("查询密钥 ID 为 %d 关联模型失败：%v", key.ID, err))
			return fmt.Errorf("查询密钥 ID 为 %d 关联的模型失败：%w", key.ID, err)
		}

		backups = append(backups, apiKeyModelsBackup{
			apiKeyID: key.ID,
			models:   models,
		})
		logger.Debug("备份密钥关联关系",
			slog.Uint64("apikey_id", uint64(key.ID)),
			slog.Int("model_count", len(models)))
	}
	logger.Debug("完成备份关联关系", slog.Int("backup_count", len(backups)))

	// 清理密钥与模型的多对多关联关系
	//
	// TODO：这里由于存在未知错误，导致该操作在事务内无法正常完成，
	// 因此采取暂时将其移动到事务外的临时方案。
	// Issue：https://github.com/go-gorm/gorm/issues/7649
	for _, backup := range backups {
		if len(backup.models) == 0 {
			continue
		}
		// 清理该密钥与所有模型的关联
		if err := s.platformControlRepo.ClearAPIKeyModelRelations(ctx, backup.apiKeyID); err != nil {
			logger.Error("清理密钥与模型的关联关系失败",
				slog.Uint64("apikey_id", uint64(backup.apiKeyID)),
				slog.Any("error", err))
			_ = s.logPlatformControlAudit(ctx, id, "platform.delete", "failed", fmt.Sprintf("清理密钥 ID 为 %d 与模型关联关系失败：%v", backup.apiKeyID, err))
			return fmt.Errorf("清理密钥 ID 为 %d 与模型的关联关系失败：%w", backup.apiKeyID, err)
		}
		logger.Debug("成功清理密钥与模型的关联关系", slog.Uint64("apikey_id", uint64(backup.apiKeyID)), slog.Int("model_count", len(backup.models)))
	}
	logger.Debug("成功清理所有密钥与模型的关联关系")

	// 在事务中执行删除操作
	err = s.controlTx.WithinTx(ctx, func(txCtx context.Context) error {
		// 删除所有模型
		deletedModels, innerErr := s.platformControlRepo.DeleteModelsByPlatform(txCtx, id)
		if innerErr != nil {
			return innerErr
		}
		logger.Debug("成功删除所有模型", slog.Int64("deleted_count", deletedModels))

		// 删除所有密钥
		deletedKeys, innerErr := s.platformControlRepo.DeleteAPIKeysByPlatform(txCtx, id)
		if innerErr != nil {
			return innerErr
		}
		logger.Debug("成功删除所有密钥", slog.Int64("deleted_count", deletedKeys))

		// 删除平台本身
		deletedPlatforms, innerErr := s.platformControlRepo.DeletePlatform(txCtx, id)
		if innerErr != nil {
			return innerErr
		}
		if deletedPlatforms == 0 {
			logger.Warn("平台已被删除")
			return fmt.Errorf("平台 ID 为 %d 已被删除", id)
		}

		return nil
	})

	if err != nil {
		// 事务失败，恢复关联关系备份
		logger.Warn("事务失败，开始恢复关联关系备份", slog.Any("error", err))
		for _, backup := range backups {
			if restoreErr := s.platformControlRepo.AppendAPIKeyModels(ctx, backup.apiKeyID, backup.models); restoreErr != nil {
				logger.Error("恢复密钥与模型的关联关系失败",
					slog.Uint64("apikey_id", uint64(backup.apiKeyID)),
					slog.Any("error", restoreErr))
			} else {
				logger.Debug("成功恢复密钥与模型的关联关系",
					slog.Uint64("apikey_id", uint64(backup.apiKeyID)),
					slog.Int("model_count", len(backup.models)))
			}
		}
		logger.Debug("完成关联关系恢复")
		_ = s.logPlatformControlAudit(ctx, id, "platform.delete", "failed", fmt.Sprintf("删除平台失败：%v", err))
		return fmt.Errorf("删除平台失败：%w", err)
	}

	logger.Info("成功删除平台及其所有关联数据")
	_ = s.logPlatformControlAudit(ctx, id, "platform.delete", "success", "删除平台成功")
	return nil
}

// GetResourcePlatformMaps 获取密钥和模型的 resource_id -> platform_id 映射
func (s *service) GetResourcePlatformMaps(ctx context.Context) (keyMap, modelMap map[uint]uint, err error) {
	s.logger.Debug("开始获取资源平台映射")

	k := query.Q.APIKey
	// 仅查询 id 和 platform_id 两列，减少数据传输
	apiKeys, err := k.WithContext(ctx).Select(k.ID, k.PlatformID).Find()
	if err != nil {
		s.logger.Error("查询密钥平台映射失败", slog.Any("error", err))
		return nil, nil, fmt.Errorf("查询密钥平台映射失败：%w", err)
	}

	m := query.Q.Model
	models, err := m.WithContext(ctx).Select(m.ID, m.PlatformID).Find()
	if err != nil {
		s.logger.Error("查询模型平台映射失败", slog.Any("error", err))
		return nil, nil, fmt.Errorf("查询模型平台映射失败：%w", err)
	}

	keyMap = make(map[uint]uint, len(apiKeys))
	for _, ak := range apiKeys {
		keyMap[ak.ID] = ak.PlatformID
	}

	modelMap = make(map[uint]uint, len(models))
	for _, md := range models {
		modelMap[md.ID] = md.PlatformID
	}

	s.logger.Debug("成功获取资源平台映射",
		slog.Int("key_count", len(keyMap)),
		slog.Int("model_count", len(modelMap)))
	return keyMap, modelMap, nil
}
