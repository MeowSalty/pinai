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
