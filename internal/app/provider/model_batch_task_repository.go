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

func (r *modelBatchTaskGormRepository) taskModelDB(ctx context.Context) *gorm.DB {
	db := r.taskDB(ctx).
		Session(&gorm.Session{NewDB: true}).
		WithContext(ctx)

	if db.Statement != nil {
		db.Statement.Table = ""
		db.Statement.TableExpr = nil
		db.Statement.Model = nil
		db.Statement.Schema = nil
		db.Statement.Dest = nil
	}

	return db.Model(&types.ModelBatchTask{})
}

// CreateModelBatchTask 创建模型批量任务。
func (r *modelBatchTaskGormRepository) CreateModelBatchTask(ctx context.Context, task *types.ModelBatchTask) error {
	if task == nil {
		return fmt.Errorf("创建模型批量任务失败：任务参数不能为空")
	}

	if err := r.taskModelDB(ctx).Create(task).Error; err != nil {
		r.logger.Error("创建模型批量任务失败", slog.Any("error", err))
		return fmt.Errorf("创建模型批量任务失败：%w", err)
	}

	return nil
}

// GetModelBatchTaskByID 根据 ID 查询模型批量任务。
func (r *modelBatchTaskGormRepository) GetModelBatchTaskByID(ctx context.Context, taskID uint) (*types.ModelBatchTask, error) {
	var task types.ModelBatchTask
	err := r.taskModelDB(ctx).Where("id = ?", taskID).First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("未找到 ID 为 %d 的任务：%w", taskID, ErrTaskNotFound)
		}
		r.logger.Error("查询模型批量任务失败", slog.Uint64("task_id", uint64(taskID)), slog.Any("error", err))
		return nil, fmt.Errorf("查询模型批量任务失败：%w", err)
	}

	return &task, nil
}

// ListUnfinishedModelBatchTasks 查询未完成任务（pending/running）。
func (r *modelBatchTaskGormRepository) ListUnfinishedModelBatchTasks(ctx context.Context) ([]*types.ModelBatchTask, error) {
	var tasks []*types.ModelBatchTask
	err := r.taskModelDB(ctx).
		Where("status IN ?", []string{types.ModelBatchTaskStatusPending, types.ModelBatchTaskStatusRunning}).
		Order("id ASC").
		Find(&tasks).Error
	if err != nil {
		r.logger.Error("查询未完成模型批量任务失败", slog.Any("error", err))
		return nil, fmt.Errorf("查询未完成模型批量任务失败：%w", err)
	}

	return tasks, nil
}

// MarkModelBatchTaskRunning 将任务标记为运行中。
func (r *modelBatchTaskGormRepository) MarkModelBatchTaskRunning(ctx context.Context, taskID uint, startedAt time.Time) error {
	resultDB := r.taskModelDB(ctx).
		Where("id = ?", taskID).
		Updates(map[string]any{
			"status":        types.ModelBatchTaskStatusRunning,
			"started_at":    startedAt,
			"error_message": "",
			"result":        "",
		})
	if resultDB.Error != nil {
		r.logger.Error("标记模型批量任务为运行中失败", slog.Uint64("task_id", uint64(taskID)), slog.Any("error", resultDB.Error))
		return fmt.Errorf("标记模型批量任务为运行中失败：%w", resultDB.Error)
	}
	if resultDB.RowsAffected == 0 {
		return fmt.Errorf("标记模型批量任务为运行中失败：未找到 ID 为 %d 的任务：%w", taskID, ErrTaskNotFound)
	}

	return nil
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

	resultDB := r.taskModelDB(ctx).
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
