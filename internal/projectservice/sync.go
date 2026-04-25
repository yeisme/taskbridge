package projectservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
	syncengine "github.com/yeisme/taskbridge/internal/sync"
)

type SyncService struct {
	TaskStore    storage.Storage
	ProjectStore project.Store
}

func (s *SyncService) SyncProject(ctx context.Context, projectID string, p provider.Provider, providerName string) (*syncengine.ProjectSyncResult, error) {
	if _, err := s.ProjectStore.GetProject(ctx, projectID); err != nil {
		return nil, fmt.Errorf("读取项目失败: %w", err)
	}

	localTasks, err := s.TaskStore.ListTasks(ctx, storage.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("读取本地任务失败: %w", err)
	}
	projectTasks := make([]model.Task, 0)
	for _, task := range localTasks {
		if taskBelongsToProject(task, projectID) {
			projectTasks = append(projectTasks, task)
		}
	}

	result := &syncengine.ProjectSyncResult{
		ProjectID: projectID,
		Provider:  providerName,
		Status:    string(project.StatusConfirmed),
		Errors:    []string{},
	}
	if len(projectTasks) == 0 {
		result.Message = "未找到可同步的项目任务"
		return result, nil
	}

	taskLists, err := p.ListTaskLists(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取远程清单失败: %w", err)
	}
	defaultListID := syncengine.FindDefaultTaskListID(taskLists)
	targetSource := model.TaskSource(providerName)
	if targetSource == model.SourceGoogle {
		parentTasks, childTasks := syncengine.SplitTasksByParentRelation(projectTasks)
		syncengine.PushProjectLocalTasks(ctx, s.TaskStore, p, parentTasks, defaultListID, targetSource, false, result)
		if len(childTasks) > 0 {
			if err := syncengine.PullProviderTasksIntoLocal(ctx, s.TaskStore, p, providerName); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("pull-before-children: %v", err))
			}
			syncengine.PushProjectLocalTasks(ctx, s.TaskStore, p, childTasks, defaultListID, targetSource, false, result)
		}
	} else {
		syncengine.PushProjectLocalTasks(ctx, s.TaskStore, p, projectTasks, defaultListID, targetSource, false, result)
	}

	item, err := s.ProjectStore.GetProject(ctx, projectID)
	if err == nil {
		item.Status = project.StatusSynced
		_ = s.ProjectStore.SaveProject(ctx, item)
		result.Status = string(project.StatusSynced)
	}
	result.Message = fmt.Sprintf("项目已同步到 %s", providerName)
	return result, nil
}

func taskBelongsToProject(task model.Task, projectID string) bool {
	if task.Metadata == nil || task.Metadata.CustomFields == nil {
		return false
	}
	raw, ok := task.Metadata.CustomFields["tb_project_id"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case string:
		return v == projectID
	case fmt.Stringer:
		return v.String() == projectID
	default:
		return strings.TrimSpace(fmt.Sprint(v)) == projectID
	}
}
