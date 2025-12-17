package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// AddKeyToPlatform 实现为指定供应方添加新密钥
func (s *service) AddKeyToPlatform(ctx context.Context, providerId uint, key types.APIKey) (*types.APIKey, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(providerId)))
	logger.Debug("开始为平台添加 API 密钥")

	// 检查平台是否存在
	if err := s.validatePlatformExists(ctx, providerId); err != nil {
		logger.Warn("平台不存在")
		return nil, err
	}

	// 设置密钥的平台 ID
	key.PlatformID = providerId

	// 创建密钥
	key.ID = 0
	if err := query.Q.APIKey.WithContext(ctx).Create(&key); err != nil {
		logger.Error("创建 API 密钥失败", slog.Any("error", err))
		return nil, fmt.Errorf("创建 API 密钥失败：%w", err)
	}

	logger.Info("成功为平台添加 API 密钥", slog.Uint64("key_id", uint64(key.ID)))
	return &key, nil
}

// GetKeysByPlatform 实现获取指定供应方的所有密钥列表
func (s *service) GetKeysByPlatform(ctx context.Context, providerId uint) ([]*types.APIKey, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(providerId)))
	logger.Debug("开始获取平台的 API 密钥列表")

	// 检查平台是否存在
	if err := s.validatePlatformExists(ctx, providerId); err != nil {
		logger.Warn("平台不存在")
		return nil, err
	}

	// 获取密钥列表 (包含密钥值)
	keys, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.PlatformID.Eq(providerId)).Find()
	if err != nil {
		logger.Error("获取 API 密钥列表失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取平台 ID 为 %d 的 API 密钥失败：%w", providerId, err)
	}

	logger.Info("成功获取平台的 API 密钥列表", slog.Int("count", len(keys)))
	return keys, nil
}

// UpdateKey 实现更新指定密钥
func (s *service) UpdateKey(ctx context.Context, keyId uint, key types.APIKey) (*types.APIKey, error) {
	logger := s.logger.With(slog.Uint64("key_id", uint64(keyId)))
	logger.Debug("开始更新 API 密钥")

	// 只更新非零值字段
	result, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId)).Updates(key)
	if err != nil {
		logger.Error("更新 API 密钥失败", slog.Any("error", err))
		return nil, fmt.Errorf("更新 ID 为 %d 的密钥失败：%w", keyId, err)
	}
	if result.RowsAffected == 0 {
		logger.Warn("API 密钥不存在")
		return nil, fmt.Errorf("未找到 ID 为 %d 的密钥", keyId)
	}

	// 返回更新后的完整对象
	updatedKey, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId)).First()
	if err != nil {
		logger.Error("获取更新后的 API 密钥失败", slog.Any("error", err))
		return nil, fmt.Errorf("获取更新后的 ID 为 %d 的密钥失败：%w", keyId, err)
	}

	logger.Info("成功更新 API 密钥")
	return updatedKey, nil
}

// DeleteKey 实现删除指定密钥
func (s *service) DeleteKey(ctx context.Context, keyId uint) error {
	logger := s.logger.With(slog.Uint64("key_id", uint64(keyId)))
	logger.Debug("开始删除 API 密钥")

	// 先查询密钥是否存在
	apiKey, err := s.getAPIKeyByID(ctx, keyId)
	if err != nil {
		logger.Warn("API 密钥不存在或查询失败", slog.Any("error", err))
		return err
	}

	// 备份密钥与模型的关联关系
	var backupModels []*types.Model
	count := query.Q.APIKey.Models.Model(apiKey).Count()
	if count > 0 {
		logger.Debug("开始备份密钥与模型的关联关系")
		backupModels, err = query.Q.APIKey.Models.Model(apiKey).Find()
		if err != nil {
			logger.Error("查询密钥关联的模型失败", slog.Any("error", err))
			return fmt.Errorf("查询密钥 ID 为 %d 关联的模型失败：%w", keyId, err)
		}
		logger.Debug("备份密钥关联关系",
			slog.Uint64("key_id", uint64(keyId)),
			slog.Int("model_count", len(backupModels)))
	}

	// 清理密钥与模型的多对多关联关系
	//
	// TODO：这里由于存在未知错误，导致该操作在事务内无法正常完成，
	// 因此采取暂时将其移动到事务外的临时方案。
	// Issue：https://github.com/go-gorm/gorm/issues/7649
	if count > 0 {
		// 清理该密钥与所有模型的关联
		if err := query.Q.APIKey.Models.Model(apiKey).Clear(); err != nil {
			logger.Error("清理密钥与模型的关联关系失败",
				slog.Uint64("key_id", uint64(keyId)),
				slog.Any("error", err))
			return fmt.Errorf("清理密钥 ID 为 %d 与模型的关联关系失败：%w", keyId, err)
		}
		logger.Debug("成功清理密钥与模型的关联关系", slog.Uint64("key_id", uint64(keyId)), slog.Int64("model_count", count))
	} else {
		logger.Debug("密钥没有关联模型，跳过清理", slog.Uint64("key_id", uint64(keyId)))
	}

	// 在事务中执行删除操作
	err = query.Q.Transaction(func(tx *query.Query) error {
		// 删除密钥
		result, err := tx.APIKey.WithContext(ctx).Where(tx.APIKey.ID.Eq(keyId)).Delete()
		if err != nil {
			logger.Error("删除 API 密钥失败", slog.Any("error", err))
			return fmt.Errorf("删除 ID 为 %d 的 API 密钥失败：%w", keyId, err)
		}
		if result.RowsAffected == 0 {
			logger.Warn("API 密钥不存在")
			return fmt.Errorf("未找到 ID 为 %d 的 API 密钥", keyId)
		}
		logger.Debug("成功删除密钥", slog.Int64("deleted_count", result.RowsAffected))

		return nil
	})

	if err != nil {
		// 事务失败，恢复关联关系备份
		if len(backupModels) > 0 {
			logger.Warn("事务失败，开始恢复关联关系备份", slog.Any("error", err))
			if restoreErr := query.Q.APIKey.Models.Model(apiKey).Append(backupModels...); restoreErr != nil {
				logger.Error("恢复密钥与模型的关联关系失败",
					slog.Uint64("key_id", uint64(keyId)),
					slog.Any("error", restoreErr))
			} else {
				logger.Debug("成功恢复密钥与模型的关联关系",
					slog.Uint64("key_id", uint64(keyId)),
					slog.Int("model_count", len(backupModels)))
			}
			logger.Debug("完成关联关系恢复")
		}
		return err
	}

	logger.Info("成功删除 API 密钥")

	count, err = s.removeOrphanedModels(ctx, apiKey.PlatformID, logger)
	if err != nil {
		logger.Error("删除孤立模型失败", slog.Any("error", err))
		return err
	}
	if count > 0 {
		logger.Info("成功删除孤立模型", slog.Int64("deleted_count", count))
	}
	return nil
}
