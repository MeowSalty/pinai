package provider

import (
	"context"

	"github.com/MeowSalty/pinai/database/types"
)

// New 创建一个新的 Service 实例
func New() Service {
	return &service{}
}

// Service 定义了 LLM 供应商管理的服务接口
type Service interface {
	// CreateProvider 创建一个新的供应方，包括平台、模型和密钥
	CreateProvider(ctx context.Context, req CreateRequest) (*types.Platform, error)

	// CreatePlatform 创建一个新的平台
	CreatePlatform(ctx context.Context, platform types.Platform) (*types.Platform, error)

	// GetPlatforms 获取所有平台列表
	GetPlatforms(ctx context.Context) ([]*types.Platform, error)

	// GetPlatform 获取指定平台详情
	GetPlatform(ctx context.Context, id uint) (*types.Platform, error)

	// UpdatePlatform 更新指定平台信息
	UpdatePlatform(ctx context.Context, id uint, platform types.Platform) (*types.Platform, error)

	// DeleteProvider 删除指定供应方 (将级联删除模型和密钥)
	DeleteProvider(ctx context.Context, id uint) error

	// AddModelToPlatform 为指定平台添加新模型
	AddModelToPlatform(ctx context.Context, platformId uint, model types.Model) (*types.Model, error)

	// GetModelsByPlatform 获取指定平台的所有模型列表
	GetModelsByPlatform(ctx context.Context, platformId uint) ([]*types.Model, error)

	// UpdateModel 更新指定模型信息
	UpdateModel(ctx context.Context, modelId uint, model types.Model) (*types.Model, error)

	// DeleteModel 删除指定模型
	DeleteModel(ctx context.Context, modelId uint) error

	// AddKeyToPlatform 为指定平台添加新密钥
	AddKeyToPlatform(ctx context.Context, platformId uint, key types.APIKey) (*types.APIKey, error)

	// GetKeysByPlatform 获取指定平台的所有密钥列表
	GetKeysByPlatform(ctx context.Context, platformId uint) ([]*types.APIKey, error)

	// UpdateKey 更新指定密钥
	UpdateKey(ctx context.Context, keyId uint, key types.APIKey) (*types.APIKey, error)

	// DeleteKey 删除指定密钥
	DeleteKey(ctx context.Context, keyId uint) error
}
