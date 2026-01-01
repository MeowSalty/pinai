package health

import (
	"context"
	"fmt"
	"log/slog"
)

// Service 定义健康服务接口
type Service interface {
	GetStorage() *Storage
}

// service 健康服务实现
type service struct {
	storage *Storage // 健康状态存储，用于缓存和持久化
}

// NewService 创建健康服务实例
//
// 该函数会在 health 包内部初始化 Storage，确保存储的初始化逻辑封装在 health 包中
//
// 参数：
//
//	ctx - 上下文，用于初始化存储
//	logger - 日志记录器
//
// 返回值：
//
//	Service - 健康服务实例
//	error - 初始化错误
func NewService(ctx context.Context, logger *slog.Logger) (Service, error) {
	logger.Info("开始初始化健康服务")

	// 在 health 包内部初始化存储
	storage, err := NewStorage(ctx, logger)
	if err != nil {
		logger.Error("初始化健康状态存储失败", "error", err)
		return nil, fmt.Errorf("初始化健康服务失败：%w", err)
	}

	logger.Info("健康服务初始化完成")
	return &service{
		storage: storage,
	}, nil
}

// GetStorage 获取健康状态存储实例
//
// 该方法用于导出内部的健康状态存储实例，供其他服务（如 Portal Service）使用
//
// 返回值：
//
//	*Storage - 健康状态存储实例
func (s *service) GetStorage() *Storage {
	return s.storage
}
