package provider

import (
	"context"
	"log/slog"

	"github.com/MeowSalty/pinai/database/types"
)

// New 创建一个新的 Service 实例
func New(logger *slog.Logger) Service {
	return &service{
		logger: logger,
	}
}

// Service 定义了 LLM 供应商管理的服务接口
type Service interface {
	// CreatePlatform 创建一个新的平台
	CreatePlatform(ctx context.Context, platform types.Platform) (*types.Platform, error)

	// GetPlatforms 获取所有平台列表
	GetPlatforms(ctx context.Context) ([]*types.Platform, error)

	// GetPlatform 获取指定平台详情
	GetPlatform(ctx context.Context, id uint) (*types.Platform, error)

	// UpdatePlatform 更新指定平台信息
	UpdatePlatform(ctx context.Context, id uint, platform types.Platform) (*types.Platform, error)

	// DeletePlatform 删除指定平台（包括其关联的模型、密钥及关联关系）
	DeletePlatform(ctx context.Context, id uint) error

	// AddModelToPlatform 为指定平台添加新模型
	AddModelToPlatform(ctx context.Context, platformId uint, model types.Model) (*types.Model, error)

	// BatchAddModelsToPlatform 批量为指定平台添加模型（原子性操作）
	BatchAddModelsToPlatform(ctx context.Context, platformId uint, models []types.Model) ([]*types.Model, error)

	// GetModelsByPlatform 获取指定平台的所有模型列表
	GetModelsByPlatform(ctx context.Context, platformId uint) ([]*types.Model, error)

	// UpdateModel 更新指定模型信息
	UpdateModel(ctx context.Context, modelId uint, model types.Model) (*types.Model, error)

	// BatchUpdateModels 批量更新指定平台的模型（原子性操作）
	BatchUpdateModels(ctx context.Context, platformId uint, updateItems []ModelUpdateItem) ([]*types.Model, error)

	// DeleteModel 删除指定模型
	DeleteModel(ctx context.Context, modelId uint) error

	// BatchDeleteModels 批量删除指定平台的模型（原子性操作）
	BatchDeleteModels(ctx context.Context, platformId uint, modelIds []uint) (int, error)

	// AddKeyToPlatform 为指定平台添加新密钥
	AddKeyToPlatform(ctx context.Context, platformId uint, key types.APIKey) (*types.APIKey, error)

	// GetKeysByPlatform 获取指定平台的所有密钥列表
	GetKeysByPlatform(ctx context.Context, platformId uint) ([]*types.APIKey, error)

	// GetKey 获取指定密钥详情
	GetKey(ctx context.Context, keyId uint) (*types.APIKey, error)

	// UpdateKey 更新指定密钥
	UpdateKey(ctx context.Context, keyId uint, key types.APIKey) (*types.APIKey, error)

	// DeleteKey 删除指定密钥
	DeleteKey(ctx context.Context, keyId uint) error

	// AddEndpointToPlatform 为指定平台添加新端点
	AddEndpointToPlatform(ctx context.Context, platformId uint, endpoint types.Endpoint) (*types.Endpoint, error)

	// BatchAddEndpointsToPlatform 批量为指定平台添加端点（原子性操作）
	BatchAddEndpointsToPlatform(ctx context.Context, platformId uint, endpoints []types.Endpoint) ([]*types.Endpoint, error)

	// GetEndpointsByPlatform 获取指定平台的所有端点列表
	GetEndpointsByPlatform(ctx context.Context, platformId uint) ([]*types.Endpoint, error)

	// GetEndpoint 获取指定端点详情
	GetEndpoint(ctx context.Context, endpointId uint) (*types.Endpoint, error)

	// UpdateEndpoint 更新指定端点
	UpdateEndpoint(ctx context.Context, endpointId uint, endpoint types.Endpoint) (*types.Endpoint, error)

	// BatchUpdateEndpoints 批量更新指定平台的端点（原子性操作）
	BatchUpdateEndpoints(ctx context.Context, platformId uint, updateItems []EndpointUpdateItem) ([]*types.Endpoint, error)

	// DeleteEndpoint 删除指定端点
	DeleteEndpoint(ctx context.Context, endpointId uint) error

	// GetModel 获取指定模型详情
	GetModel(ctx context.Context, modelId uint) (*types.Model, error)
}
