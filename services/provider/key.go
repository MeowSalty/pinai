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
	return s.addKeyToPlatformApp(ctx, providerId, key)
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

// GetKey 实现获取指定密钥详情
func (s *service) GetKey(ctx context.Context, keyId uint) (*types.APIKey, error) {
	logger := s.logger.With(slog.Uint64("key_id", uint64(keyId)))
	logger.Debug("开始获取 API 密钥详情")

	// 查询密钥
	apiKey, err := s.getAPIKeyByID(ctx, keyId)
	if err != nil {
		logger.Warn("API 密钥不存在或查询失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功获取 API 密钥详情")
	return apiKey, nil
}

// UpdateKey 实现更新指定密钥
func (s *service) UpdateKey(ctx context.Context, keyId uint, key types.APIKey) (*types.APIKey, error) {
	return s.updateKeyApp(ctx, keyId, key)
}

// DeleteKey 实现删除指定密钥
func (s *service) DeleteKey(ctx context.Context, keyId uint) error {
	return s.deleteKeyApp(ctx, keyId)
}
