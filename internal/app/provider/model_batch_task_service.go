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

	return &BatchTaskAcceptedResponse{TaskID: task.ID, Type: task.Type, Status: task.Status}, nil
}

func (s *service) GetModelBatchTask(ctx context.Context, taskID uint) (*ModelBatchTaskSummary, error) {
	task, err := s.modelBatchTaskRepo.GetModelBatchTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	toTime := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		v := t.Format(time.RFC3339)
		return &v
	}

	return &ModelBatchTaskSummary{
		ID:           task.ID,
		Type:         task.Type,
		Status:       task.Status,
		PlatformID:   task.PlatformID,
		Result:       json.RawMessage(task.Result),
		ErrorMessage: task.ErrorMessage,
		StartedAt:    toTime(task.StartedAt),
		FinishedAt:   toTime(task.FinishedAt),
		CreatedAt:    task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    task.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *service) StartModelBatchTaskWorker(ctx context.Context) error {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()

	if s.workerRunning {
		return nil
	}

	if s.modelBatchTaskRepo == nil {
		return fmt.Errorf("启动模型批量任务 worker 失败：任务仓储未初始化")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.workerCancel = cancel
	s.workerDone = done
	s.workerRunning = true

	pollInterval := time.Second
	if s.workerPollSecond > 0 {
		pollInterval = time.Duration(s.workerPollSecond) * time.Second
	}

	logger := s.logger.With(
		slog.String("operation", "model_batch_task_worker"),
		slog.Duration("poll_interval", pollInterval),
	)

	go func() {
		defer close(done)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			if err := s.processOneModelBatchTask(workerCtx); err != nil {
				logger.Error("处理模型批量任务失败", slog.Any("error", err))
			}

			select {
			case <-workerCtx.Done():
				logger.Info("模型批量任务 worker 已停止")
				return
			case <-ticker.C:
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
	task, err := s.modelBatchTaskRepo.ClaimNextPendingModelBatchTask(ctx)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	logger := s.logger.With(
		slog.String("operation", "model_batch_task"),
		slog.Uint64("task_id", uint64(task.ID)),
		slog.String("task_type", task.Type),
		slog.Uint64("platform_id", uint64(task.PlatformID)),
	)

	result, runErr := s.executeModelBatchTask(ctx, task)
	if runErr != nil {
		_ = s.modelBatchTaskRepo.FinishModelBatchTask(context.Background(), task.ID, types.ModelBatchTaskStatusFailed, "", runErr.Error())
		logger.Error("模型批量任务执行失败", slog.Any("error", runErr))
		return nil
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		_ = s.modelBatchTaskRepo.FinishModelBatchTask(context.Background(), task.ID, types.ModelBatchTaskStatusFailed, "", fmt.Sprintf("序列化任务结果失败：%v", err))
		logger.Error("序列化模型批量任务结果失败", slog.Any("error", err))
		return nil
	}

	if err := s.modelBatchTaskRepo.FinishModelBatchTask(context.Background(), task.ID, types.ModelBatchTaskStatusSucceeded, string(resultBytes), ""); err != nil {
		logger.Error("更新模型批量任务成功状态失败", slog.Any("error", err))
	}

	logger.Info("模型批量任务执行成功")
	return nil
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
