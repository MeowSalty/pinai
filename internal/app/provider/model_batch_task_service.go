package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

type modelBatchAddTaskPayload struct {
	PlatformID uint          `json:"platform_id"`
	Models     []types.Model `json:"models"`
}

type modelBatchUpdateTaskPayload struct {
	PlatformID uint              `json:"platform_id"`
	Models     []ModelUpdateItem `json:"models"`
}

type modelBatchDeleteTaskPayload struct {
	PlatformID uint   `json:"platform_id"`
	ModelIDs   []uint `json:"model_ids"`
}

func (s *service) EnqueueBatchAddModelsTask(ctx context.Context, platformId uint, models []types.Model) (*BatchTaskAcceptedResponse, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("必须至少提供一个模型")
	}

	exists, err := s.modelControlRepo.ExistsPlatform(ctx, platformId)
	if err != nil {
		return nil, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformId, ErrResourceNotFound)
	}

	payloadBytes, err := json.Marshal(modelBatchAddTaskPayload{PlatformID: platformId, Models: models})
	if err != nil {
		return nil, fmt.Errorf("构建批量新增任务失败：%w", err)
	}

	task := &types.ModelBatchTask{
		Type:       types.ModelBatchTaskTypeAdd,
		Status:     types.ModelBatchTaskStatusPending,
		PlatformID: platformId,
		Payload:    string(payloadBytes),
	}
	if err := s.modelBatchTaskRepo.CreateModelBatchTask(ctx, task); err != nil {
		return nil, err
	}

	s.ensureTaskRuntimeInitialized()
	s.cacheTaskSummary(s.loadTaskSummaryFromEntity(task))
	if err := s.enqueueTaskInMemory(task.ID); err != nil {
		s.logger.Error("模型批量新增任务入队失败", slog.Uint64("task_id", uint64(task.ID)), slog.Any("error", err))
		return nil, fmt.Errorf("模型批量新增任务入队失败：%w", err)
	}

	return &BatchTaskAcceptedResponse{TaskID: task.ID, Type: task.Type, Status: task.Status}, nil
}

func (s *service) EnqueueBatchUpdateModelsTask(ctx context.Context, platformId uint, updateItems []ModelUpdateItem) (*BatchTaskAcceptedResponse, error) {
	if len(updateItems) == 0 {
		return nil, fmt.Errorf("必须至少提供一个模型更新项")
	}

	for i, item := range updateItems {
		if item.ID == 0 {
			return nil, fmt.Errorf("模型更新项 %d 缺少必需的 ID 字段", i)
		}
	}

	exists, err := s.modelControlRepo.ExistsPlatform(ctx, platformId)
	if err != nil {
		return nil, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformId, ErrResourceNotFound)
	}

	payloadBytes, err := json.Marshal(modelBatchUpdateTaskPayload{PlatformID: platformId, Models: updateItems})
	if err != nil {
		return nil, fmt.Errorf("构建批量更新任务失败：%w", err)
	}

	task := &types.ModelBatchTask{
		Type:       types.ModelBatchTaskTypeUpdate,
		Status:     types.ModelBatchTaskStatusPending,
		PlatformID: platformId,
		Payload:    string(payloadBytes),
	}
	if err := s.modelBatchTaskRepo.CreateModelBatchTask(ctx, task); err != nil {
		return nil, err
	}

	s.ensureTaskRuntimeInitialized()
	s.cacheTaskSummary(s.loadTaskSummaryFromEntity(task))
	if err := s.enqueueTaskInMemory(task.ID); err != nil {
		s.logger.Error("模型批量更新任务入队失败", slog.Uint64("task_id", uint64(task.ID)), slog.Any("error", err))
		return nil, fmt.Errorf("模型批量更新任务入队失败：%w", err)
	}

	return &BatchTaskAcceptedResponse{TaskID: task.ID, Type: task.Type, Status: task.Status}, nil
}

func (s *service) EnqueueBatchDeleteModelsTask(ctx context.Context, platformId uint, modelIds []uint) (*BatchTaskAcceptedResponse, error) {
	if len(modelIds) == 0 {
		return nil, fmt.Errorf("必须至少提供一个模型 ID")
	}

	exists, err := s.modelControlRepo.ExistsPlatform(ctx, platformId)
	if err != nil {
		return nil, fmt.Errorf("检查平台是否存在失败：%w", err)
	}
	if !exists {
		return nil, fmt.Errorf("未找到 ID 为 %d 的平台：%w", platformId, ErrResourceNotFound)
	}

	payloadBytes, err := json.Marshal(modelBatchDeleteTaskPayload{PlatformID: platformId, ModelIDs: modelIds})
	if err != nil {
		return nil, fmt.Errorf("构建批量删除任务失败：%w", err)
	}

	task := &types.ModelBatchTask{
		Type:       types.ModelBatchTaskTypeDelete,
		Status:     types.ModelBatchTaskStatusPending,
		PlatformID: platformId,
		Payload:    string(payloadBytes),
	}
	if err := s.modelBatchTaskRepo.CreateModelBatchTask(ctx, task); err != nil {
		return nil, err
	}

	s.ensureTaskRuntimeInitialized()
	s.cacheTaskSummary(s.loadTaskSummaryFromEntity(task))
	if err := s.enqueueTaskInMemory(task.ID); err != nil {
		s.logger.Error("模型批量删除任务入队失败", slog.Uint64("task_id", uint64(task.ID)), slog.Any("error", err))
		return nil, fmt.Errorf("模型批量删除任务入队失败：%w", err)
	}

	return &BatchTaskAcceptedResponse{TaskID: task.ID, Type: task.Type, Status: task.Status}, nil
}

func (s *service) GetModelBatchTask(ctx context.Context, taskID uint) (*ModelBatchTaskSummary, error) {
	if cached, ok := s.getCachedTaskSummary(taskID); ok {
		return cached, nil
	}

	task, err := s.modelBatchTaskRepo.GetModelBatchTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	summary := s.loadTaskSummaryFromEntity(task)
	if summary == nil {
		return nil, fmt.Errorf("读取模型批量任务失败：任务为空")
	}
	s.cacheTaskSummary(summary)
	return summary, nil
}

func (s *service) StartModelBatchTaskWorker(ctx context.Context) error {
	s.workerMu.Lock()
	if s.workerRunning {
		s.workerMu.Unlock()
		return nil
	}

	if s.modelBatchTaskRepo == nil {
		s.workerMu.Unlock()
		return fmt.Errorf("启动模型批量任务 worker 失败：任务仓储未初始化")
	}
	s.ensureTaskRuntimeInitializedLocked()

	if ctx == nil {
		ctx = context.Background()
	}

	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.workerCancel = cancel
	s.workerDone = done
	s.workerRunning = true
	s.workerMu.Unlock()

	logger := s.logger.With(
		slog.String("operation", "model_batch_task_worker"),
		slog.Int("queue_size", s.taskQueueSize),
	)

	if err := s.recoverModelBatchTasks(workerCtx); err != nil {
		cancel()
		s.workerMu.Lock()
		s.workerCancel = nil
		s.workerDone = nil
		s.workerRunning = false
		s.workerMu.Unlock()
		return fmt.Errorf("启动模型批量任务 worker 失败：恢复任务失败：%w", err)
	}

	go func() {
		defer close(done)
		for {
			if err := s.processOneModelBatchTask(workerCtx); err != nil {
				if err == context.Canceled {
					logger.Info("模型批量任务 worker 已停止")
					return
				}
				logger.Error("处理模型批量任务失败", slog.Any("error", err))
			}
		}
	}()

	logger.Info("模型批量任务 worker 已启动")
	return nil
}

func (s *service) StopModelBatchTaskWorker(ctx context.Context) error {
	s.workerMu.Lock()
	if !s.workerRunning {
		s.workerMu.Unlock()
		return nil
	}

	cancel := s.workerCancel
	done := s.workerDone
	s.workerCancel = nil
	s.workerDone = nil
	s.workerRunning = false
	s.workerMu.Unlock()

	if cancel != nil {
		cancel()
	}

	if done == nil {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("停止模型批量任务 worker 超时：%w", ctx.Err())
	}
}

func (s *service) processOneModelBatchTask(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return context.Canceled
	case taskID := <-s.taskQueue:
		defer s.removeQueuedMark(taskID)

		task, err := s.modelBatchTaskRepo.GetModelBatchTaskByID(ctx, taskID)
		if err != nil {
			return err
		}

		startedAt := time.Now()
		if err := s.modelBatchTaskRepo.MarkModelBatchTaskRunning(ctx, task.ID, startedAt); err != nil {
			s.logger.Error("标记模型批量任务运行中失败", slog.Uint64("task_id", uint64(task.ID)), slog.Any("error", err))
			return nil
		}

		task.Status = types.ModelBatchTaskStatusRunning
		task.StartedAt = &startedAt
		task.ErrorMessage = ""
		task.Result = ""
		s.cacheTaskSummary(s.loadTaskSummaryFromEntity(task))

		logger := s.logger.With(
			slog.String("operation", "model_batch_task"),
			slog.Uint64("task_id", uint64(task.ID)),
			slog.String("task_type", task.Type),
			slog.Uint64("platform_id", uint64(task.PlatformID)),
		)

		result, runErr := s.executeModelBatchTask(ctx, task)
		if runErr != nil {
			finishedAt := time.Now()
			task.Status = types.ModelBatchTaskStatusFailed
			task.FinishedAt = &finishedAt
			task.ErrorMessage = runErr.Error()
			s.cacheTaskSummary(s.loadTaskSummaryFromEntity(task))

			if err := s.modelBatchTaskRepo.FinishModelBatchTask(context.Background(), task.ID, types.ModelBatchTaskStatusFailed, "", runErr.Error()); err != nil {
				logger.Error("更新模型批量任务失败状态失败", slog.Any("error", err))
			}
			logger.Error("模型批量任务执行失败", slog.Any("error", runErr))
			return nil
		}

		resultBytes, err := json.Marshal(result)
		if err != nil {
			finishedAt := time.Now()
			task.Status = types.ModelBatchTaskStatusFailed
			task.FinishedAt = &finishedAt
			task.ErrorMessage = fmt.Sprintf("序列化任务结果失败：%v", err)
			s.cacheTaskSummary(s.loadTaskSummaryFromEntity(task))

			if finishErr := s.modelBatchTaskRepo.FinishModelBatchTask(context.Background(), task.ID, types.ModelBatchTaskStatusFailed, "", task.ErrorMessage); finishErr != nil {
				logger.Error("更新模型批量任务失败状态失败", slog.Any("error", finishErr))
			}
			logger.Error("序列化模型批量任务结果失败", slog.Any("error", err))
			return nil
		}

		finishedAt := time.Now()
		task.Status = types.ModelBatchTaskStatusSucceeded
		task.FinishedAt = &finishedAt
		task.ErrorMessage = ""
		task.Result = string(resultBytes)
		s.cacheTaskSummary(s.loadTaskSummaryFromEntity(task))

		if err := s.modelBatchTaskRepo.FinishModelBatchTask(context.Background(), task.ID, types.ModelBatchTaskStatusSucceeded, string(resultBytes), ""); err != nil {
			logger.Error("更新模型批量任务成功状态失败", slog.Any("error", err))
			return nil
		}

		logger.Info("模型批量任务执行成功")
		return nil
	}
}

func (s *service) executeModelBatchTask(ctx context.Context, task *types.ModelBatchTask) (*BatchTaskResult, error) {
	if task == nil {
		return nil, fmt.Errorf("执行模型批量任务失败：任务为空")
	}

	taskCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	switch task.Type {
	case types.ModelBatchTaskTypeAdd:
		var payload modelBatchAddTaskPayload
		if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
			return nil, fmt.Errorf("解析批量新增任务载荷失败：%w", err)
		}
		created, err := s.batchAddModelsToPlatformApp(taskCtx, payload.PlatformID, payload.Models)
		if err != nil {
			return nil, err
		}
		return &BatchTaskResult{TotalCount: len(payload.Models), CreatedCount: len(created)}, nil

	case types.ModelBatchTaskTypeUpdate:
		var payload modelBatchUpdateTaskPayload
		if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
			return nil, fmt.Errorf("解析批量更新任务载荷失败：%w", err)
		}
		updated, err := s.batchUpdateModelsApp(taskCtx, payload.PlatformID, payload.Models)
		if err != nil {
			return nil, err
		}
		return &BatchTaskResult{TotalCount: len(payload.Models), UpdatedCount: len(updated)}, nil

	case types.ModelBatchTaskTypeDelete:
		var payload modelBatchDeleteTaskPayload
		if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
			return nil, fmt.Errorf("解析批量删除任务载荷失败：%w", err)
		}
		deleted, err := s.batchDeleteModelsApp(taskCtx, payload.PlatformID, payload.ModelIDs)
		if err != nil {
			return nil, err
		}
		return &BatchTaskResult{TotalCount: len(payload.ModelIDs), DeletedCount: deleted}, nil

	default:
		return nil, fmt.Errorf("不支持的模型批量任务类型：%s", task.Type)
	}
}
