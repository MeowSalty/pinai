package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/MeowSalty/pinai/database/types"
)

func (s *service) ensureTaskRuntimeInitialized() {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	s.ensureTaskRuntimeInitializedLocked()
}

func (s *service) ensureTaskRuntimeInitializedLocked() {
	if s.taskQueueSize <= 0 {
		s.taskQueueSize = 256
	}
	if s.taskQueue == nil {
		s.taskQueue = make(chan uint, s.taskQueueSize)
	}

	s.taskStateMu.Lock()
	defer s.taskStateMu.Unlock()

	if s.taskStateCache == nil {
		s.taskStateCache = make(map[uint]*ModelBatchTaskSummary)
	}
	if s.taskEnqueued == nil {
		s.taskEnqueued = make(map[uint]struct{})
	}
}

func (s *service) cacheTaskSummary(summary *ModelBatchTaskSummary) {
	if summary == nil {
		return
	}

	s.taskStateMu.Lock()
	defer s.taskStateMu.Unlock()

	cloned := *summary
	s.taskStateCache[summary.ID] = &cloned
}

func (s *service) getCachedTaskSummary(taskID uint) (*ModelBatchTaskSummary, bool) {
	s.taskStateMu.RLock()
	defer s.taskStateMu.RUnlock()

	summary, ok := s.taskStateCache[taskID]
	if !ok || summary == nil {
		return nil, false
	}

	cloned := *summary
	return &cloned, true
}

func (s *service) removeQueuedMark(taskID uint) {
	s.taskStateMu.Lock()
	defer s.taskStateMu.Unlock()
	delete(s.taskEnqueued, taskID)
}

func (s *service) enqueueTaskInMemory(taskID uint) error {
	s.taskStateMu.Lock()
	if _, exists := s.taskEnqueued[taskID]; exists {
		s.taskStateMu.Unlock()
		return nil
	}
	s.taskEnqueued[taskID] = struct{}{}
	s.taskStateMu.Unlock()

	select {
	case s.taskQueue <- taskID:
		return nil
	default:
		s.removeQueuedMark(taskID)
		return fmt.Errorf("内存任务队列已满")
	}
}

func (s *service) loadTaskSummaryFromEntity(task *types.ModelBatchTask) *ModelBatchTaskSummary {
	if task == nil {
		return nil
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
		Result:       []byte(task.Result),
		ErrorMessage: task.ErrorMessage,
		StartedAt:    toTime(task.StartedAt),
		FinishedAt:   toTime(task.FinishedAt),
		CreatedAt:    task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    task.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *service) recoverModelBatchTasks(ctx context.Context) error {
	tasks, err := s.modelBatchTaskRepo.ListUnfinishedModelBatchTasks(ctx)
	if err != nil {
		return fmt.Errorf("恢复模型批量任务失败：%w", err)
	}

	for _, task := range tasks {
		summary := s.loadTaskSummaryFromEntity(task)
		if summary == nil {
			continue
		}
		if summary.Status == types.ModelBatchTaskStatusRunning {
			summary.Status = types.ModelBatchTaskStatusPending
			summary.StartedAt = nil
		}
		s.cacheTaskSummary(summary)

		if err := s.enqueueTaskInMemory(task.ID); err != nil {
			return fmt.Errorf("恢复任务 %d 入队失败：%w", task.ID, err)
		}
	}

	return nil
}
