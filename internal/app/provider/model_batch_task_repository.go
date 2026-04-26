package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
	"gorm.io/gorm"
)

// modelBatchTaskGormRepository 是基于 GORM 的模型批量任务仓储实现。
type modelBatchTaskGormRepository struct {
	logger *slog.Logger
}

// NewModelBatchTaskGormRepository 创建模型批量任务仓储。
func NewModelBatchTaskGormRepository(logger *slog.Logger) ModelBatchTaskRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &modelBatchTaskGormRepository{logger: logger.WithGroup("model_batch_task_repo")}
}

func (r *modelBatchTaskGormRepository) taskDB(ctx context.Context) *gorm.DB {
	q := queryFromContextOrDefault(ctx)
	return q.Model.WithContext(ctx).UnderlyingDB()
}

// CreateModelBatchTask 创建模型批量任务。
func (r *modelBatchTaskGormRepository) CreateModelBatchTask(ctx context.Context, task *types.ModelBatchTask) error {
	if task == nil {
		return fmt.Errorf("创建模型批量任务失败：任务参数不能为空")
	}

	if err := r.taskDB(ctx).Create(task).Error; err != nil {
		r.logger.Error("创建模型批量任务失败", slog.Any("error", err))
		return fmt.Errorf("创建模型批量任务失败：%w", err)
	}

	return nil
}

// GetModelBatchTaskByID 根据 ID 查询模型批量任务。
func (r *modelBatchTaskGormRepository) GetModelBatchTaskByID(ctx context.Context, taskID uint) (*types.ModelBatchTask, error) {
	var task types.ModelBatchTask
	err := r.taskDB(ctx).Where("id = ?", taskID).First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("未找到 ID 为 %d 的任务：%w", taskID, ErrTaskNotFound)
		}
		r.logger.Error("查询模型批量任务失败", slog.Uint64("task_id", uint64(taskID)), slog.Any("error", err))
		return nil, fmt.Errorf("查询模型批量任务失败：%w", err)
	}

	return &task, nil
}

// ClaimNextPendingModelBatchTask 抢占下一个待执行任务并设置为运行中。
func (r *modelBatchTaskGormRepository) ClaimNextPendingModelBatchTask(ctx context.Context) (*types.ModelBatchTask, error) {
	db := r.taskDB(ctx)

	for range 3 {
		var claimed *types.ModelBatchTask
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var task types.ModelBatchTask
			if err := tx.Where("status = ?", types.ModelBatchTaskStatusPending).
				Order("id ASC").
				First(&task).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}

			now := time.Now()
			result := tx.Model(&types.ModelBatchTask{}).
				Where("id = ? AND status = ?", task.ID, types.ModelBatchTaskStatusPending).
				Updates(map[string]any{
					"status":        types.ModelBatchTaskStatusRunning,
					"started_at":    now,
					"error_message": "",
					"result":        "",
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}

			task.Status = types.ModelBatchTaskStatusRunning
			task.StartedAt = &now
			claimed = &task
			return nil
		})
		if err != nil {
			r.logger.Error("抢占模型批量任务失败", slog.Any("error", err))
			return nil, fmt.Errorf("抢占模型批量任务失败：%w", err)
		}
		if claimed != nil {
			return claimed, nil
		}
	}

	return nil, nil
}

// FinishModelBatchTask 完成模型批量任务。
func (r *modelBatchTaskGormRepository) FinishModelBatchTask(ctx context.Context, taskID uint, status, result, errorMessage string) error {
	if status != types.ModelBatchTaskStatusSucceeded && status != types.ModelBatchTaskStatusFailed {
		return fmt.Errorf("完成模型批量任务失败：不支持的任务状态 %s", status)
	}

	now := time.Now()
	updateMap := map[string]any{
		"status":        status,
		"result":        result,
		"error_message": errorMessage,
		"finished_at":   now,
	}

	resultDB := r.taskDB(ctx).Model(&types.ModelBatchTask{}).
		Where("id = ?", taskID).
		Updates(updateMap)
	if resultDB.Error != nil {
		r.logger.Error("更新模型批量任务状态失败", slog.Uint64("task_id", uint64(taskID)), slog.Any("error", resultDB.Error))
		return fmt.Errorf("更新模型批量任务状态失败：%w", resultDB.Error)
	}
	if resultDB.RowsAffected == 0 {
		return fmt.Errorf("更新模型批量任务状态失败：未找到 ID 为 %d 的任务：%w", taskID, ErrTaskNotFound)
	}

	return nil
}
