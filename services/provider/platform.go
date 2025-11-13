package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// CreatePlatform 实现创建平台
func (s *service) CreatePlatform(ctx context.Context, platform types.Platform) (*types.Platform, error) {
	s.logger.Debug("开始创建平台", slog.String("platform_name", platform.Name))

	platform.ID = 0
	if err := query.Q.Platform.WithContext(ctx).Create(&platform); err != nil {
		s.logger.Error("创建平台失败", slog.String("platform_name", platform.Name), slog.Any("error", err))
		return nil, fmt.Errorf("创建平台失败：%w", err)
	}

	s.logger.Info("成功创建平台", slog.String("platform_name", platform.Name), slog.Uint64("platform_id", uint64(platform.ID)))
	return &platform, nil
}

// GetPlatforms 实现获取平台列表
func (s *service) GetPlatforms(ctx context.Context) ([]*types.Platform, error) {
	s.logger.Debug("开始获取平台列表")

	platforms, err := query.Q.Platform.WithContext(ctx).Find()
	if err != nil {
		s.logger.Error("获取平台列表失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取平台列表失败：%w", err)
	}

	s.logger.Info("成功获取平台列表", slog.Int("count", len(platforms)))
	return platforms, nil
}

// GetPlatform 实现获取指定平台详情
func (s *service) GetPlatform(ctx context.Context, id uint) (*types.Platform, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(id)))
	logger.Debug("开始获取平台详情")

	platform, err := s.getPlatformByID(ctx, id)
	if err != nil {
		logger.Error("获取平台详情失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功获取平台详情", slog.String("platform_name", platform.Name))
	return platform, nil
}

// UpdatePlatform 实现更新平台信息
func (s *service) UpdatePlatform(ctx context.Context, id uint, platform types.Platform) (*types.Platform, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(id)))
	logger.Debug("开始更新平台")

	// 只更新非零值字段
	result, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).Updates(platform)
	if err != nil {
		logger.Error("更新平台失败", slog.Any("error", err))
		return nil, fmt.Errorf("更新 ID 为 %d 的平台失败：%w", id, err)
	}
	if result.RowsAffected == 0 {
		logger.Warn("平台不存在")
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台", id)
	}

	// 返回更新后的完整对象
	updatedPlatform, err := s.getPlatformByID(ctx, id)
	if err != nil {
		logger.Error("获取更新后的平台失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功更新平台", slog.String("platform_name", updatedPlatform.Name))
	return updatedPlatform, nil
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

	// 检查平台是否存在
	platform, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("平台不存在")
			return fmt.Errorf("未找到 ID 为 %d 的平台", id)
		}
		logger.Error("查询平台失败", slog.Any("error", err))
		return fmt.Errorf("查询平台失败：%w", err)
	}

	logger.Debug("找到平台", slog.String("platform_name", platform.Name))

	// 查询平台下的所有密钥
	apiKeys, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.PlatformID.Eq(id)).Find()
	if err != nil {
		logger.Error("查询平台关联的密钥失败", slog.Any("error", err))
		return fmt.Errorf("查询平台关联的密钥失败：%w", err)
	}
	logger.Debug("查询到关联密钥", slog.Int("apikey_count", len(apiKeys)))

	// 备份密钥与模型的关联关系
	backups := make([]apiKeyModelsBackup, 0, len(apiKeys))
	logger.Debug("开始备份密钥与模型的关联关系")
	for _, key := range apiKeys {
		// 查询该密钥关联的所有模型
		models, err := query.Q.APIKey.Models.Model(key).Find()
		if err != nil {
			logger.Error("查询密钥关联的模型失败",
				slog.Uint64("apikey_id", uint64(key.ID)),
				slog.Any("error", err))
			return fmt.Errorf("查询密钥 ID 为 %d 关联的模型失败：%w", key.ID, err)
		}
		if len(models) > 0 {
			backups = append(backups, apiKeyModelsBackup{
				apiKeyID: key.ID,
				models:   models,
			})
			logger.Debug("备份密钥关联关系",
				slog.Uint64("apikey_id", uint64(key.ID)),
				slog.Int("model_count", len(models)))
		}
	}
	logger.Debug("完成备份关联关系", slog.Int("backup_count", len(backups)))

	// 清理密钥与模型的多对多关联关系
	//
	// TODO：这里由于存在未知错误，导致该操作在事务内无法正常完成，
	// 因此采取暂时将其移动到事务外的临时方案。
	// Issue：https://github.com/go-gorm/gorm/issues/7649
	for _, key := range apiKeys {
		count := query.Q.APIKey.Models.Model(key).Count()
		if count == 0 {
			logger.Debug("密钥没有关联模型，跳过清理", slog.Uint64("apikey_id", uint64(key.ID)))
			continue
		}
		// 清理该密钥与所有模型的关联
		if err := query.Q.APIKey.Models.Model(key).Clear(); err != nil {
			logger.Error("清理密钥与模型的关联关系失败",
				slog.Uint64("apikey_id", uint64(key.ID)),
				slog.Any("error", err))
			return fmt.Errorf("清理密钥 ID 为 %d 与模型的关联关系失败：%w", key.ID, err)
		}
		logger.Debug("成功清理密钥与模型的关联关系", slog.Uint64("apikey_id", uint64(key.ID)), slog.Int64("model_count", count))
	}
	logger.Debug("成功清理所有密钥与模型的关联关系")

	// 在事务中执行删除操作
	err = query.Q.Transaction(func(tx *query.Query) error {
		// 删除所有模型
		result, err := tx.Model.WithContext(ctx).Where(tx.Model.PlatformID.Eq(id)).Delete()
		if err != nil {
			logger.Error("删除平台关联的模型失败", slog.Any("error", err))
			return fmt.Errorf("删除平台关联的模型失败：%w", err)
		}
		logger.Debug("成功删除所有模型", slog.Int64("deleted_count", result.RowsAffected))

		// 删除所有密钥
		result, err = tx.APIKey.WithContext(ctx).Where(tx.APIKey.PlatformID.Eq(id)).Delete()
		if err != nil {
			logger.Error("删除平台关联的密钥失败", slog.Any("error", err))
			return fmt.Errorf("删除平台关联的密钥失败：%w", err)
		}
		logger.Debug("成功删除所有密钥", slog.Int64("deleted_count", result.RowsAffected))

		// 删除平台本身
		result, err = tx.Platform.WithContext(ctx).Where(tx.Platform.ID.Eq(id)).Delete()
		if err != nil {
			logger.Error("删除平台失败", slog.Any("error", err))
			return fmt.Errorf("删除平台失败：%w", err)
		}
		if result.RowsAffected == 0 {
			logger.Warn("平台已被删除")
			return fmt.Errorf("平台 ID 为 %d 已被删除", id)
		}

		return nil
	})

	if err != nil {
		// 事务失败，恢复关联关系备份
		logger.Warn("事务失败，开始恢复关联关系备份", slog.Any("error", err))
		for _, backup := range backups {
			key := &types.APIKey{ID: backup.apiKeyID}
			if restoreErr := query.Q.APIKey.Models.Model(key).Append(backup.models...); restoreErr != nil {
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
		return err
	}

	logger.Info("成功删除平台及其所有关联数据")
	return nil
}
