package stats

import (
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
}

// globalCollector 全局采集器实例
var globalCollector *Collector

// InitCollector 初始化全局采集器
func InitCollector() {
	globalCollector = &Collector{
		requestCounts: make([]int64, 60), // 保存过去 60 秒的数据
		currentSecond: time.Now().Unix(),
	}

	// 启动后台清理协程
	go globalCollector.cleanup()
}

// GetCollector 获取全局采集器实例
func GetCollector() *Collector {
	if globalCollector == nil {
		InitCollector()
	}
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
	}

	// 记录当前秒的请求数
	c.requestCounts[index]++
}

// IncrementConnection 增加活动连接数
func (c *Collector) IncrementConnection() {
	atomic.AddInt64(&c.activeConnections, 1)
}

// DecrementConnection 减少活动连接数
func (c *Collector) DecrementConnection() {
	atomic.AddInt64(&c.activeConnections, -1)
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
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now().Unix()

		// 清理超过 60 秒的数据
		if now > c.currentSecond {
			for i := c.currentSecond + 1; i <= now; i++ {
				c.requestCounts[i%60] = 0
			}
			c.currentSecond = now
		}
		c.mu.Unlock()
	}
}
