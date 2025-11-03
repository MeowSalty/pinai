package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// CreateProvider 实现创建供应方的业务逻辑
func (s *service) CreateProvider(ctx context.Context, req CreateRequest) (*types.Platform, error) {
	s.logger.Debug("开始创建供应方", slog.String("platform_name", req.Platform.Name))

	// 开启事务以确保原子性
	tx := query.Q.Begin()
	if tx.Error != nil {
		s.logger.Error("开启事务失败", slog.Any("error", tx.Error))
		return nil, fmt.Errorf("开启事务失败：%w", tx.Error)
	}

	// 使用 defer 确保事务被正确处理
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			s.logger.Error("创建供应方时发生 panic，已回滚事务", slog.Any("panic", r))
		}
	}()

	// 1. 创建平台
	platform := req.Platform
	platform.ID = 0
	if err := tx.Platform.WithContext(ctx).Create(&platform); err != nil {
		tx.Rollback()
		s.logger.Error("创建平台失败，已回滚事务", slog.String("platform_name", platform.Name), slog.Any("error", err))
		return nil, fmt.Errorf("创建平台失败：%w", err)
	}
	s.logger.Debug("平台创建成功", slog.Uint64("platform_id", uint64(platform.ID)))

	// 2. 创建密钥
	apiKey := req.APIKey
	apiKey.ID = 0
	apiKey.PlatformID = platform.ID
	if err := tx.APIKey.WithContext(ctx).Create(&apiKey); err != nil {
		tx.Rollback()
		s.logger.Error("创建 API 密钥失败，已回滚事务", slog.Uint64("platform_id", uint64(platform.ID)), slog.Any("error", err))
		return nil, fmt.Errorf("创建 API 密钥失败：%w", err)
	}
	s.logger.Debug("API 密钥创建成功", slog.Uint64("key_id", uint64(apiKey.ID)))

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
			s.logger.Error("批量创建模型失败，已回滚事务", slog.Uint64("platform_id", uint64(platform.ID)), slog.Int("model_count", len(modelsToCreate)), slog.Any("error", err))
			return nil, fmt.Errorf("创建模型失败：%w", err)
		}
		s.logger.Debug("模型批量创建成功", slog.Int("model_count", len(modelsToCreate)))
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		s.logger.Error("提交事务失败", slog.Uint64("platform_id", uint64(platform.ID)), slog.Any("error", err))
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	s.logger.Info("成功创建供应方",
		slog.String("platform_name", platform.Name),
		slog.Uint64("platform_id", uint64(platform.ID)),
		slog.Int("model_count", len(modelsToCreate)))
	return &platform, nil
}

// DeleteProvider 实现删除供应方 (将级联删除模型和密钥)
func (s *service) DeleteProvider(ctx context.Context, id uint) error {
	logger := s.logger.With(slog.Uint64("platform_id", uint64(id)))
	logger.Debug("开始删除供应方")

	// 开启事务以确保原子性
	tx := query.Q.Begin()
	if tx.Error != nil {
		logger.Error("开启事务失败", slog.Any("error", tx.Error))
		return fmt.Errorf("开启事务失败：%w", tx.Error)
	}

	// 使用 defer 确保事务被正确处理
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("删除供应方时发生 panic，已回滚事务", slog.Any("panic", r))
		}
	}()

	// 1. 删除关联的模型
	modelResult, err := tx.Model.WithContext(ctx).Where(tx.Model.PlatformID.Eq(id)).Delete()
	if err != nil {
		tx.Rollback()
		logger.Error("删除关联模型失败，已回滚事务", slog.Any("error", err))
		return fmt.Errorf("删除平台 ID 为 %d 的模型失败：%w", id, err)
	}
	logger.Debug("删除关联模型成功", slog.Int64("deleted_count", modelResult.RowsAffected))

	// 2. 删除关联的密钥
	keyResult, err := tx.APIKey.WithContext(ctx).Where(tx.APIKey.PlatformID.Eq(id)).Delete()
	if err != nil {
		tx.Rollback()
		logger.Error("删除关联 API 密钥失败，已回滚事务", slog.Any("error", err))
		return fmt.Errorf("删除平台 ID 为 %d 的 API 密钥失败：%w", id, err)
	}
	logger.Debug("删除关联 API 密钥成功", slog.Int64("deleted_count", keyResult.RowsAffected))

	// 3. 删除平台本身
	result, err := tx.Platform.WithContext(ctx).Where(tx.Platform.ID.Eq(id)).Delete()
	if err != nil {
		tx.Rollback()
		logger.Error("删除平台失败，已回滚事务", slog.Any("error", err))
		return fmt.Errorf("删除 ID 为 %d 的平台失败：%w", id, err)
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		logger.Warn("平台不存在")
		return fmt.Errorf("未找到 ID 为 %d 的平台", id)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		logger.Error("提交事务失败", slog.Any("error", err))
		return fmt.Errorf("提交事务失败：%w", err)
	}

	logger.Info("成功删除供应方",
		slog.Int64("deleted_models", modelResult.RowsAffected),
		slog.Int64("deleted_keys", keyResult.RowsAffected))
	return nil
}
