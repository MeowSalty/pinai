package health

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/MeowSalty/pinai/database/query"
	"github.com/MeowSalty/pinai/database/types"
)

// Storage 健康状态存储实现
//
// 使用内存缓存 + 数据库持久化的混合存储策略：
//   - 读操作优先从内存缓存获取，提高性能
//   - 写操作同时更新缓存和数据库，保证数据一致性
//   - 使用 sync.Map 保证线程安全
type Storage struct {
	cache  sync.Map     // 内存缓存，key 格式："resourceType:resourceID"
	logger *slog.Logger // 日志记录器
}

// NewStorage 创建新的健康状态存储实例
//
// 参数：
//   - ctx: 上下文，用于初始化时从数据库加载数据
//   - logger: 日志记录器
//
// 返回值：
//   - *Storage: 存储实例
//   - error: 初始化错误
func NewStorage(ctx context.Context, logger *slog.Logger) (*Storage, error) {
	storageLogger := logger.WithGroup("health_storage")
	storageLogger.Info("初始化健康状态存储")

	storage := &Storage{
		logger: storageLogger,
	}

	// 从数据库加载所有健康状态到缓存
	if err := storage.loadFromDatabase(ctx); err != nil {
		storageLogger.Error("从数据库加载健康状态失败", "error", err)
		return nil, fmt.Errorf("初始化健康状态存储失败：%w", err)
	}

	storageLogger.Info("健康状态存储初始化完成")
	return storage, nil
}

// Get 获取指定资源的健康状态
//
// 实现 health.Storage 接口
func (s *Storage) Get(resourceType types.ResourceType, resourceID uint) (*types.Health, error) {
	key := s.makeKey(resourceType, resourceID)

	s.logger.Debug("获取健康状态",
		"resource_type", resourceType,
		"resource_id", resourceID,
		"key", key)

	if value, ok := s.cache.Load(key); ok {
		h := value.(*types.Health)
		s.logger.Debug("从缓存获取健康状态成功",
			"resource_type", resourceType,
			"resource_id", resourceID,
			"status", h.Status)
		return h, nil
	}

	s.logger.Debug("健康状态不存在",
		"resource_type", resourceType,
		"resource_id", resourceID)
	return nil, nil
}

// Set 设置指定资源的健康状态
//
// 实现 health.Storage 接口
// 同时更新内存缓存和数据库
func (s *Storage) Set(status *types.Health) error {
	key := s.makeKey(status.ResourceType, status.ResourceID)

	s.logger.Debug("设置健康状态",
		"resource_type", status.ResourceType,
		"resource_id", status.ResourceID,
		"status", status.Status,
		"key", key)

	// 更新内存缓存
	s.cache.Store(key, status)

	// 持久化到数据库
	if err := s.saveToDatabase(context.Background(), status); err != nil {
		s.logger.Error("保存健康状态到数据库失败",
			"error", err,
			"resource_type", status.ResourceType,
			"resource_id", status.ResourceID)
		return fmt.Errorf("保存健康状态失败：%w", err)
	}

	s.logger.Debug("健康状态设置成功",
		"resource_type", status.ResourceType,
		"resource_id", status.ResourceID)
	return nil
}

// Delete 删除指定资源的健康状态
//
// 实现 health.Storage 接口
// 同时删除内存缓存和数据库记录
func (s *Storage) Delete(resourceType types.ResourceType, resourceID uint) error {
	key := s.makeKey(resourceType, resourceID)

	s.logger.Info("删除健康状态",
		"resource_type", resourceType,
		"resource_id", resourceID,
		"key", key)

	// 从内存缓存删除
	s.cache.Delete(key)

	// 从数据库删除
	if err := s.deleteFromDatabase(context.Background(), resourceType, resourceID); err != nil {
		s.logger.Error("从数据库删除健康状态失败",
			"error", err,
			"resource_type", resourceType,
			"resource_id", resourceID)
		return fmt.Errorf("删除健康状态失败：%w", err)
	}

	s.logger.Info("健康状态删除成功",
		"resource_type", resourceType,
		"resource_id", resourceID)
	return nil
}

// makeKey 生成缓存键
//
// 格式："resourceType:resourceID"
func (s *Storage) makeKey(resourceType types.ResourceType, resourceID uint) string {
	return fmt.Sprintf("%d:%d", resourceType, resourceID)
}

// loadFromDatabase 从数据库加载所有健康状态到缓存
func (s *Storage) loadFromDatabase(ctx context.Context) error {
	s.logger.Debug("开始从数据库加载健康状态")

	q := query.Q

	healths, err := q.WithContext(ctx).Health.Find()
	if err != nil {
		s.logger.Error("查询数据库失败", "error", err)
		return fmt.Errorf("查询健康状态失败：%w", err)
	}

	// 加载到缓存
	for _, health := range healths {
		key := s.makeKey(health.ResourceType, health.ResourceID)
		s.cache.Store(key, health)
	}

	s.logger.Info("从数据库加载健康状态完成", "count", len(healths))
	return nil
}

// saveToDatabase 保存健康状态到数据库
func (s *Storage) saveToDatabase(ctx context.Context, status *types.Health) error {
	s.logger.Debug("保存健康状态到数据库",
		"resource_type", status.ResourceType,
		"resource_id", status.ResourceID)

	q := query.Q

	// 使用 Save 进行 upsert 操作
	if err := q.WithContext(ctx).Health.Save(status); err != nil {
		s.logger.Error("保存到数据库失败",
			"error", err,
			"resource_type", status.ResourceType,
			"resource_id", status.ResourceID)
		return fmt.Errorf("保存到数据库失败：%w", err)
	}

	s.logger.Debug("保存到数据库成功",
		"resource_type", status.ResourceType,
		"resource_id", status.ResourceID)
	return nil
}

// deleteFromDatabase 从数据库删除健康状态
func (s *Storage) deleteFromDatabase(ctx context.Context, resourceType types.ResourceType, resourceID uint) error {
	s.logger.Debug("从数据库删除健康状态",
		"resource_type", resourceType,
		"resource_id", resourceID)

	q := query.Q

	// 删除记录
	_, err := q.WithContext(ctx).Health.Where(
		q.Health.ResourceType.Eq(int8(resourceType)),
		q.Health.ResourceID.Eq(resourceID),
	).Delete()

	if err != nil {
		s.logger.Error("从数据库删除失败",
			"error", err,
			"resource_type", resourceType,
			"resource_id", resourceID)
		return fmt.Errorf("从数据库删除失败：%w", err)
	}

	s.logger.Debug("从数据库删除成功",
		"resource_type", resourceType,
		"resource_id", resourceID)
	return nil
}

// StatusCount 各状态数量统计
type StatusCount struct {
	Available   int64 // 可用数量
	Warning     int64 // 警告数量
	Unavailable int64 // 不可用数量
}

// CountByResourceType 按资源类型统计各状态数量
//
// 从内存缓存中统计指定资源类型的各状态数量
//
// 参数：
//
//	resourceType - 资源类型（平台、密钥、模型）
//
// 返回值：
//
//	StatusCount - 各状态数量统计
func (s *Storage) CountByResourceType(resourceType types.ResourceType) StatusCount {
	var count StatusCount

	s.cache.Range(func(key, value interface{}) bool {
		h := value.(*types.Health)
		if h.ResourceType != resourceType {
			return true
		}

		switch h.Status {
		case types.HealthStatusAvailable:
			count.Available++
		case types.HealthStatusWarning:
			count.Warning++
		case types.HealthStatusUnavailable:
			count.Unavailable++
			// Unknown 状态不会存储在健康表中，所以不需要统计
		}
		return true
	})

	s.logger.Debug("统计资源类型健康状态完成",
		"resource_type", resourceType,
		"available", count.Available,
		"warning", count.Warning,
		"unavailable", count.Unavailable)

	return count
}

// GetByResourceType 获取指定资源类型的所有健康状态
//
// 从内存缓存中获取指定资源类型的所有健康状态记录
//
// 参数：
//
//	resourceType - 资源类型（平台、密钥、模型）
//
// 返回值：
//
//	[]*types.Health - 健康状态列表
func (s *Storage) GetByResourceType(resourceType types.ResourceType) []*types.Health {
	var results []*types.Health

	s.cache.Range(func(key, value interface{}) bool {
		h := value.(*types.Health)
		if h.ResourceType == resourceType {
			results = append(results, h)
		}
		return true
	})

	s.logger.Debug("获取资源类型健康状态列表完成",
		"resource_type", resourceType,
		"count", len(results))

	return results
}

// GetByStatus 获取指定状态的所有健康状态记录
//
// 从内存缓存中获取指定状态的所有健康状态记录
//
// 参数：
//
//	status - 健康状态
//
// 返回值：
//
//	[]*types.Health - 健康状态列表
func (s *Storage) GetByStatus(status types.HealthStatus) []*types.Health {
	var results []*types.Health

	s.cache.Range(func(key, value interface{}) bool {
		h := value.(*types.Health)
		if h.Status == status {
			results = append(results, h)
		}
		return true
	})

	s.logger.Debug("获取指定状态的健康状态列表完成",
		"status", status,
		"count", len(results))

	return results
}
