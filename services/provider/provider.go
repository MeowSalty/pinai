package provider

import (
	"context"
	"fmt"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// CreateProvider 实现创建供应方的业务逻辑
func (s *service) CreateProvider(ctx context.Context, req CreateRequest) (*types.Platform, error) {
	// 开启事务以确保原子性
	tx := query.Q.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开启事务失败：%w", tx.Error)
	}

	// 使用 defer 确保事务被正确处理
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 创建平台
	platform := req.Platform
	platform.ID = 0
	if err := tx.Platform.WithContext(ctx).Create(&platform); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建平台失败：%w", err)
	}

	// 2. 创建密钥
	apiKey := req.APIKey
	apiKey.ID = 0
	apiKey.PlatformID = platform.ID
	if err := tx.APIKey.WithContext(ctx).Create(&apiKey); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建 API 密钥失败：%w", err)
	}

	// 3. 批量创建模型
	modelsToCreate := make([]*types.Model, len(req.Models))
	for i := range req.Models {
		req.Models[i].PlatformID = platform.ID
		req.Models[i].ID = 0
		modelsToCreate[i] = &req.Models[i]
	}
	if len(modelsToCreate) > 0 {
		if err := tx.Model.WithContext(ctx).CreateInBatches(modelsToCreate, 100); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建模型失败：%w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	return &platform, nil
}

// DeleteProvider 实现删除供应方 (将级联删除模型和密钥)
func (s *service) DeleteProvider(ctx context.Context, id uint) error {
	// 开启事务以确保原子性
	tx := query.Q.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败：%w", tx.Error)
	}

	// 使用 defer 确保事务被正确处理
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 删除关联的模型
	if _, err := tx.Model.WithContext(ctx).Where(tx.Model.PlatformID.Eq(id)).Delete(); err != nil {
		tx.Rollback()
		return fmt.Errorf("删除平台 ID 为 %d 的模型失败：%w", id, err)
	}

	// 2. 删除关联的密钥
	if _, err := tx.APIKey.WithContext(ctx).Where(tx.APIKey.PlatformID.Eq(id)).Delete(); err != nil {
		tx.Rollback()
		return fmt.Errorf("删除平台 ID 为 %d 的 API 密钥失败：%w", id, err)
	}

	// 3. 删除平台本身
	result, err := tx.Platform.WithContext(ctx).Where(tx.Platform.ID.Eq(id)).Delete()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("删除 ID 为 %d 的平台失败：%w", id, err)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("未找到 ID 为 %d 的平台", id)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败：%w", err)
	}

	return nil
}
