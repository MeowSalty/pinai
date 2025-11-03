package stats

import (
	"bufio"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Collector 实时数据采集器
//
// 用于在 API 入口处采集请求数据，替代从数据库查询的方式
type Collector struct {
	// 请求计数器 - 使用滑动窗口记录每秒的请求数
	requestCounts []int64
	currentSecond int64
	mu            sync.RWMutex

	// 活动连接计数器
	activeConnections int64

	// 日志记录器
	logger *slog.Logger
}

// globalCollector 全局采集器实例
var globalCollector *Collector
var once sync.Once

// InitCollector 初始化全局采集器
func InitCollector(logger *slog.Logger) {
	globalCollector = &Collector{
		requestCounts: make([]int64, 60), // 保存过去 60 秒的数据
		currentSecond: time.Now().Unix(),
		logger:        logger,
	}

	logger.Info("实时数据采集器初始化完成")

	// 启动后台清理协程
	go globalCollector.cleanup()
}

// GetCollector 获取全局采集器实例
func GetCollector() *Collector {
	once.Do(func() {
		if globalCollector == nil {
			logger := slog.Default()
			logger.Warn("采集器未预初始化，使用默认日志记录器进行延迟初始化")
			InitCollector(logger)
		}
	})
	return globalCollector
}

// RecordRequest 记录一次请求
func (c *Collector) RecordRequest() {
	now := time.Now().Unix()
	c.mu.Lock()
	defer c.mu.Unlock()

	// 计算当前秒在数组中的索引
	index := now % 60

	// 如果进入了新的秒，需要清空该位置的旧数据
	if now != c.currentSecond {
		// 清空从上次更新到现在之间的所有秒数据
		for i := c.currentSecond + 1; i <= now; i++ {
			c.requestCounts[i%60] = 0
		}
		c.currentSecond = now
		c.logger.Debug("更新当前秒时间戳", "current_second", now)
	}

	// 记录当前秒的请求数
	c.requestCounts[index]++
}

// IncrementConnection 增加活动连接数
func (c *Collector) IncrementConnection() {
	newCount := atomic.AddInt64(&c.activeConnections, 1)
	c.logger.Debug("增加活动连接", "active_connections", newCount)
}

// DecrementConnection 减少活动连接数
func (c *Collector) DecrementConnection() {
	newCount := atomic.AddInt64(&c.activeConnections, -1)
	c.logger.Debug("减少活动连接", "active_connections", newCount)
}

// GetRPM 获取过去 1 分钟的 RPM (每分钟请求数)
func (c *Collector) GetRPM() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now().Unix()
	var total int64

	// 统计过去 60 秒的请求总数
	for i := int64(0); i < 60; i++ {
		secondTimestamp := now - i
		index := secondTimestamp % 60
		total += c.requestCounts[index]
	}

	return total
}

// GetActiveConnections 获取当前活动连接数
func (c *Collector) GetActiveConnections() int64 {
	return atomic.LoadInt64(&c.activeConnections)
}

// cleanup 后台清理协程，定期清理过期数据
func (c *Collector) cleanup() {
	c.logger.Info("启动后台清理协程")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now().Unix()

		// 清理超过 60 秒的数据
		if now > c.currentSecond {
			clearedCount := now - c.currentSecond
			for i := c.currentSecond + 1; i <= now; i++ {
				c.requestCounts[i%60] = 0
			}
			c.currentSecond = now

			if clearedCount > 1 {
				c.logger.Debug("清理过期数据", "cleared_seconds", clearedCount)
			}
		}
		c.mu.Unlock()
	}
}

// WithStreamTracking 创建流式响应的包装器
//
// 该方法用于处理流式响应的连接计数，确保在流式传输完成后才减少连接数
//
// 参数：
//   - streamFunc: 流式响应处理函数
//
// 返回值：
//   - func(*bufio.Writer): 包装后的流式响应处理函数
func (c *Collector) WithStreamTracking(streamFunc func(*bufio.Writer) error) func(*bufio.Writer) {
	return func(w *bufio.Writer) {
		// 流式响应结束时减少连接数
		defer c.DecrementConnection()
		// 执行流式响应处理
		streamFunc(w)
	}
}
