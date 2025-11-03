package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// AddKeyToPlatform 实现为指定供应方添加新密钥
func (s *service) AddKeyToPlatform(ctx context.Context, providerId uint, key types.APIKey) (*types.APIKey, error) {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(providerId)))
	logger.Debug("开始为平台添加 API 密钥")

	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("平台不存在")
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		logger.Error("查询平台失败", slog.Any("error", err))
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
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
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("平台不存在")
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		logger.Error("查询平台失败", slog.Any("error", err))
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
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

	// 删除密钥
	result, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId)).Delete()
	if err != nil {
		logger.Error("删除 API 密钥失败", slog.Any("error", err))
		return fmt.Errorf("删除 ID 为 %d 的 API 密钥失败：%w", keyId, err)
	}
	if result.RowsAffected == 0 {
		logger.Warn("API 密钥不存在")
		return fmt.Errorf("未找到 ID 为 %d 的 API 密钥", keyId)
	}

	logger.Info("成功删除 API 密钥")
	return nil
}
