package services

import (
	"context"
	"fmt"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// ProviderCreateRequest 定义了创建供应方的请求体
type ProviderCreateRequest struct {
	Platform types.Platform `json:"platform"`
	Models   []types.Model  `json:"models"`
	APIKey   types.APIKey   `json:"apiKey"`
}

// ProviderService 定义了 LLM 供应商管理的服务接口
type ProviderService interface {
	// CreateProvider 创建一个新的供应方，包括平台、模型和密钥
	CreateProvider(ctx context.Context, req ProviderCreateRequest) (*types.Platform, error)

	// GetProviders 获取所有供应方列表
	GetProviders(ctx context.Context) ([]*types.Platform, error)

	// GetProvider 获取指定供应方详情
	GetProvider(ctx context.Context, id uint) (*types.Platform, error)

	// UpdateProvider 更新指定供应方信息
	UpdateProvider(ctx context.Context, id uint, platform types.Platform) (*types.Platform, error)

	// DeleteProvider 删除指定供应方 (将级联删除模型和密钥)
	DeleteProvider(ctx context.Context, id uint) error

	// AddModelToProvider 为指定供应方添加新模型
	AddModelToProvider(ctx context.Context, providerId uint, model types.Model) (*types.Model, error)

	// GetModelsByProvider 获取指定供应方的所有模型列表
	GetModelsByProvider(ctx context.Context, providerId uint) ([]*types.Model, error)

	// UpdateModel 更新指定模型信息
	UpdateModel(ctx context.Context, providerId uint, modelId uint, model types.Model) (*types.Model, error)

	// DeleteModel 删除指定模型
	DeleteModel(ctx context.Context, providerId uint, modelId uint) error

	// AddKeyToProvider 为指定供应方添加新密钥
	AddKeyToProvider(ctx context.Context, providerId uint, key types.APIKey) (*types.APIKey, error)

	// GetKeysByProvider 获取指定供应方的所有密钥列表
	GetKeysByProvider(ctx context.Context, providerId uint) ([]*types.APIKey, error)

	// UpdateKey 更新指定密钥
	UpdateKey(ctx context.Context, providerId uint, keyId uint, key types.APIKey) (*types.APIKey, error)

	// DeleteKey 删除指定密钥
	DeleteKey(ctx context.Context, providerId uint, keyId uint) error
}

// providerService 是 ProviderService 接口的具体实现
type providerService struct {
	// healthManager *health.Manager
}

// NewProviderService 创建一个新的 ProviderService 实例
func NewProviderService() ProviderService {
	return &providerService{
		// healthManager: healthManager,
	}
}

// CreateProvider 实现创建供应方的业务逻辑
func (s *providerService) CreateProvider(ctx context.Context, req ProviderCreateRequest) (*types.Platform, error) {
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
	if err := tx.Platform.WithContext(ctx).Create(&platform); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建平台失败：%w", err)
	}

	// 2. 创建密钥
	apiKey := req.APIKey
	apiKey.PlatformID = platform.ID
	if err := tx.APIKey.WithContext(ctx).Create(&apiKey); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建 API 密钥失败：%w", err)
	}

	// 3. 创建模型
	modelsToCreate := make([]*types.Model, len(req.Models))
	for i := range req.Models {
		req.Models[i].PlatformID = platform.ID
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

// GetProviders 实现获取供应方列表
func (s *providerService) GetProviders(ctx context.Context) ([]*types.Platform, error) {
	platforms, err := query.Q.Platform.WithContext(ctx).Find()
	if err != nil {
		return nil, fmt.Errorf("获取平台列表失败：%w", err)
	}
	return platforms, nil
}

// GetProvider 实现获取指定供应方详情
func (s *providerService) GetProvider(ctx context.Context, id uint) (*types.Platform, error) {
	platform, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).First()
	if err != nil {
		return nil, fmt.Errorf("获取 ID 为 %d 的平台失败：%w", id, err)
	}
	return platform, nil
}

// UpdateProvider 实现更新供应方信息
func (s *providerService) UpdateProvider(ctx context.Context, id uint, platform types.Platform) (*types.Platform, error) {
	// 只更新非零值字段
	result, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).Updates(platform)
	if err != nil {
		return nil, fmt.Errorf("更新 ID 为 %d 的平台失败：%w", id, err)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台", id)
	}

	// 返回更新后的完整对象
	updatedPlatform, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(id)).First()
	if err != nil {
		return nil, fmt.Errorf("获取更新后的 ID 为 %d 的平台失败：%w", id, err)
	}
	return updatedPlatform, nil
}

// DeleteProvider 实现删除供应方 (将级联删除模型和密钥)
func (s *providerService) DeleteProvider(ctx context.Context, id uint) error {
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

// AddModelToProvider 实现为指定供应方添加新模型
func (s *providerService) AddModelToProvider(ctx context.Context, providerId uint, model types.Model) (*types.Model, error) {
	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 设置模型的平台 ID
	model.PlatformID = providerId

	// 创建模型
	if err := query.Q.Model.WithContext(ctx).Create(&model); err != nil {
		return nil, fmt.Errorf("创建模型失败：%w", err)
	}

	return &model, nil
}

// GetModelsByProvider 实现获取指定供应方的所有模型列表
func (s *providerService) GetModelsByProvider(ctx context.Context, providerId uint) ([]*types.Model, error) {
	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 获取模型列表
	models, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.PlatformID.Eq(providerId)).Find()
	if err != nil {
		return nil, fmt.Errorf("获取平台 ID 为 %d 的模型失败：%w", providerId, err)
	}
	return models, nil
}

// UpdateModel 实现更新指定模型信息
func (s *providerService) UpdateModel(ctx context.Context, providerId uint, modelId uint, model types.Model) (*types.Model, error) {
	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		return nil, fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 检查模型是否属于该平台
	_, err = query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId), query.Q.Model.PlatformID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("在平台 ID 为 %d 中未找到 ID 为 %d 的模型", providerId, modelId)
		}
		return nil, fmt.Errorf("查询模型时发生错误：%w", err)
	}

	// 只更新非零值字段
	result, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).Updates(model)
	if err != nil {
		return nil, fmt.Errorf("更新 ID 为 %d 的模型失败：%w", modelId, err)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
	}

	// 返回更新后的完整对象
	updatedModel, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).First()
	if err != nil {
		return nil, fmt.Errorf("获取更新后的 ID 为 %d 的模型失败：%w", modelId, err)
	}
	return updatedModel, nil
}

// DeleteModel 实现删除指定模型
func (s *providerService) DeleteModel(ctx context.Context, providerId uint, modelId uint) error {
	// 检查平台是否存在
	_, err := query.Q.Platform.WithContext(ctx).Where(query.Q.Platform.ID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("未找到 ID 为 %d 的平台", providerId)
		}
		return fmt.Errorf("查询平台时发生错误：%w", err)
	}

	// 检查模型是否属于该平台
	_, err = query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId), query.Q.Model.PlatformID.Eq(providerId)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("在平台 ID 为 %d 中未找到 ID 为 %d 的模型", providerId, modelId)
		}
		return fmt.Errorf("查询模型时发生错误：%w", err)
	}

	// 删除模型
	result, err := query.Q.Model.WithContext(ctx).Where(query.Q.Model.ID.Eq(modelId)).Delete()
	if err != nil {
		return fmt.Errorf("删除 ID 为 %d 的模型失败：%w", modelId, err)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到 ID 为 %d 的模型", modelId)
	}

	return nil
}

// AddKeyToProvider 实现为指定供应方添加新密钥
func (s *providerService) AddKeyToProvider(ctx context.Context, providerId uint, key types.APIKey) (*types.APIKey, error) {
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
	if err := query.Q.APIKey.WithContext(ctx).Create(&key); err != nil {
		return nil, fmt.Errorf("创建 API 密钥失败：%w", err)
	}

	return &key, nil
}

// GetKeysByProvider 实现获取指定供应方的所有密钥列表
func (s *providerService) GetKeysByProvider(ctx context.Context, providerId uint) ([]*types.APIKey, error) {
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
func (s *providerService) UpdateKey(ctx context.Context, providerId uint, keyId uint, key types.APIKey) (*types.APIKey, error) {
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
func (s *providerService) DeleteKey(ctx context.Context, providerId uint, keyId uint) error {
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