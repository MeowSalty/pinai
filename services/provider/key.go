package provider

import (
	"context"
	"fmt"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// AddKeyToProvider 实现为指定供应方添加新密钥
func (s *service) AddKeyToProvider(ctx context.Context, providerId uint, key types.APIKey) (*types.APIKey, error) {
	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 设置密钥的平台 ID
	key.PlatformID = providerId

	// 创建密钥
	key.ID = 0
	if err := query.Q.APIKey.WithContext(ctx).Create(&key); err != nil {
		return nil, fmt.Errorf("创建 API 密钥失败：%w", err)
	}

	return &key, nil
}

// GetKeysByProvider 实现获取指定供应方的所有密钥列表
func (s *service) GetKeysByProvider(ctx context.Context, providerId uint) ([]*types.APIKey, error) {
	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 获取密钥列表 (包含密钥值)
	keys, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.PlatformID.Eq(providerId)).Find()
	if err != nil {
		return nil, fmt.Errorf("获取平台 ID 为 %d 的 API 密钥失败：%w", providerId, err)
	}
	return keys, nil
}

// UpdateKey 实现更新指定密钥
func (s *service) UpdateKey(ctx context.Context, providerId uint, keyId uint, key types.APIKey) (*types.APIKey, error) {
	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的供应方", providerId)
		}
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 检查密钥是否属于该平台
	_, err = query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId), query.Q.APIKey.PlatformID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("在供应方 %d 中未找到 ID 为 %d 的密钥", providerId, keyId)
		}
		return nil, fmt.Errorf("查询密钥时发生错误：%w", err)
	}

	// 只更新非零值字段
	result, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId)).Updates(key)
	if err != nil {
		return nil, fmt.Errorf("更新 ID 为 %d 的密钥失败：%w", keyId, err)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("未找到 ID 为 %d 的密钥", keyId)
	}

	// 返回更新后的完整对象
	updatedKey, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId)).First()
	if err != nil {
		return nil, fmt.Errorf("获取更新后的 ID 为 %d 的密钥失败：%w", keyId, err)
	}

	return updatedKey, nil
}

// DeleteKey 实现删除指定密钥
func (s *service) DeleteKey(ctx context.Context, providerId uint, keyId uint) error {
	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		return fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 检查密钥是否属于该平台
	_, err = query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId), query.Q.APIKey.PlatformID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("在平台 ID 为 %d 中未找到 ID 为 %d 的 API 密钥", providerId, keyId)
		}
		return fmt.Errorf("查询密钥时发生错误：%w", err)
	}

	// 删除密钥
	result, err := query.Q.APIKey.WithContext(ctx).Where(query.Q.APIKey.ID.Eq(keyId)).Delete()
	if err != nil {
		return fmt.Errorf("删除 ID 为 %d 的 API 密钥失败：%w", keyId, err)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到 ID 为 %d 的 API 密钥", keyId)
	}

	return nil
}
