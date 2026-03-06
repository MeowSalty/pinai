package database

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// migrateRequestLogFields 迁移 request_logs 表的请求类型字段。
//
// 该迁移用于将旧字段 RequestType 的语义拆分为两个布尔字段：
//   - is_stream: 是否流式
//   - is_native: 是否原生（native）
//
// 迁移策略：
// 1. 如果列不存在则补齐列与索引
// 2. 从 request_type 解析并回填到新列（仅在 is_stream=false 且 is_native=false 的记录上执行）
func migrateRequestLogFields(db *gorm.DB) error {
	logger := slog.With("migration", "request_logs")
	migrator := db.Migrator()

	// 临时结构体，仅用于迁移阶段读取/删除旧列。
	type RequestLog struct {
		ID          uint
		RequestType string
		IsStream    bool
		IsNative    bool
	}

	if !migrator.HasTable(&RequestLog{}) {
		logger.Info("request_logs 表不存在，跳过迁移")
		return nil
	}

	// 2) 检查所有相关列是否都存在
	// 仅当 RequestType、IsStream、IsNative 列都存在时，才进行迁移
	hasRequestType := migrator.HasColumn(&RequestLog{}, "request_type")
	hasIsStream := migrator.HasColumn(&RequestLog{}, "is_stream")
	hasIsNative := migrator.HasColumn(&RequestLog{}, "is_native")

	if !hasRequestType || !hasIsStream || !hasIsNative {
		logger.Info("request_logs 相关列不完整，跳过历史数据回填", "has_request_type", hasRequestType, "has_is_stream", hasIsStream, "has_is_native", hasIsNative)
		return nil
	}

	// 注意：这里使用 SQL 进行批量回填，以减少内存占用并保持速度。
	// 逻辑与需求保持一致：
	//   - is_stream: request_type LIKE '%stream%' 且 request_type NOT LIKE '%non-stream%'
	//   - is_native: request_type LIKE '%-native'
	//   - 仅更新 is_stream=false AND is_native=false 的记录，避免覆盖已迁移数据
	updateSQL := `UPDATE request_logs 
SET 
    is_stream = CASE 
        WHEN request_type LIKE '%stream%' AND request_type NOT LIKE '%non-stream%' THEN true 
        ELSE false 
    END,
    is_native = CASE 
        WHEN request_type LIKE '%-native' THEN true 
        ELSE false 
    END
WHERE is_stream = false AND is_native = false`

	result := db.Exec(updateSQL)
	if result.Error != nil {
		return fmt.Errorf("回填 request_logs.is_stream/is_native 失败：%w", result.Error)
	}

	if hasRequestType {
		if err := migrator.DropColumn(&RequestLog{}, "request_type"); err != nil {
			return fmt.Errorf("删除 request_logs.request_type 列失败：%w", err)
		}
		logger.Info("request_logs.request_type 列已删除")
	}

	logger.Info("request_logs 字段迁移完成", "rows_affected", result.RowsAffected)
	return nil
}
