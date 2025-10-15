package types

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// HealthStatus 健康状态枚举
type HealthStatus DBIntType

const (
	HealthStatusUnknown     HealthStatus = iota // 未知
	HealthStatusAvailable                       // 可用
	HealthStatusWarning                         // 警告（使用退避策略）
	HealthStatusUnavailable                     // 不可用
)

// ResourceType 资源类型枚举
type ResourceType DBIntType

const (
	ResourceTypePlatform ResourceType = iota + 1 // 平台级
	ResourceTypeAPIKey                           // 密钥级
	ResourceTypeModel                            // 模型级
)

// DBIntType 自定义整数类型，用于根据数据库类型动态设置字段类型
type DBIntType int8

// GormDBDataType 实现 gorm.DBDataTypeInterface 接口，根据数据库类型返回相应的字段类型
func (DBIntType) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "postgres":
		return "smallint" // PostgreSQL 使用 smallint
	default:
		return "tinyint" // 其他数据库使用 tinyint
	}
}

// Health 健康状态表 (health_status)
type Health struct {
	ID uint `gorm:"primaryKey"`

	ResourceType ResourceType `gorm:"not null;index:idx_resource"` // 资源类型
	ResourceID   uint         `gorm:"not null;index:idx_resource"` // 资源 ID

	Status HealthStatus `gorm:"not null;index"` // 健康状态

	// 指数退避相关
	RetryCount      int        `gorm:"default:0"` // 重试次数
	NextAvailableAt *time.Time `gorm:"index"`     // 下次可用时间
	BackoffDuration int64      `gorm:"default:0"` // 当前退避时长 (秒)

	// 状态详情
	LastError     string     `gorm:"type:text"` // 最后错误信息
	LastErrorCode int        `gorm:"default:0"` // 最后错误码
	LastCheckAt   time.Time  `gorm:"not null"`  // 最后检查时间
	LastSuccessAt *time.Time // 最后成功时间

	// 统计信息
	SuccessCount int `gorm:"default:0"` // 成功次数
	ErrorCount   int `gorm:"default:0"` // 错误次数

	CreatedAt time.Time
	UpdatedAt time.Time
}
