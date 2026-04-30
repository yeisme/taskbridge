package projectservice

import (
	"context"
	"fmt"
	"time"

	"github.com/yeisme/taskbridge/internal/actionfile"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/storage"
)

type ProjectReview struct {
	Schema           string                 `json:"schema"`
	ProjectID        string                 `json:"project_id"`
	ProjectName      string                 `json:"project_name"`
	Status           project.ProjectStatus  `json:"status"`
	Progress         map[string]int         `json:"progress"`
	Risk             map[string]interface{} `json:"risk"`
	NextTaskID       string                 `json:"next_task_id,omitempty"`
	SuggestedActions []actionfile.Action    `json:"suggested_actions,omitempty"`
}

type ExecutionService struct {
	TaskStore    storage.Storage
	ProjectStore project.Store
}

func (s *ExecutionService) Review(ctx context.Context, projectID string) (*ProjectReview, error) {
	p, err := s.ProjectStore.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	tasks, err := s.projectTasks(ctx, projectID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	completed, overdue, blocked := 0, 0, 0
	var nextTaskID string
	actions := make([]actionfile.Action, 0)
	for _, task := range tasks {
		if task.Status == model.StatusCompleted {
			completed++
			continue
		}
		if task.DueDate != nil && task.DueDate.Before(now) {
			overdue++
		}
		if task.EstimatedMinutes > 180 {
			blocked++
			actions = append(actions, actionfile.Action{
				ID:                   fmt.Sprintf("project_%s_split_%d", projectID, len(actions)+1),
				Type:                 "split_task",
				TaskID:               task.ID,
				ProjectID:            projectID,
				Reason:               "项目任务预估超过 180 分钟，建议拆分",
				RequiresConfirmation: true,
			})
		}
		if nextTaskID == "" {
			nextTaskID = task.ID
		}
	}
	riskLevel := "low"
	reasons := []string{}
	if overdue > 0 {
		riskLevel = "medium"
		reasons = append(reasons, fmt.Sprintf("%d 个任务逾期", overdue))
	}
	if blocked > 0 {
		riskLevel = "medium"
		reasons = append(reasons, fmt.Sprintf("%d 个任务粒度过大", blocked))
	}
	return &ProjectReview{
		Schema:      "taskbridge.project-review.v1",
		ProjectID:   p.ID,
		ProjectName: p.Name,
		Status:      p.Status,
		Progress: map[string]int{
			"total_tasks":     len(tasks),
			"completed_tasks": completed,
			"overdue_tasks":   overdue,
			"blocked_tasks":   blocked,
		},
		Risk:             map[string]interface{}{"level": riskLevel, "reasons": reasons},
		NextTaskID:       nextTaskID,
		SuggestedActions: actions,
	}, nil
}

func (s *ExecutionService) Next(ctx context.Context, projectID string) (map[string]interface{}, error) {
	review, err := s.Review(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"schema": "taskbridge.project-next.v1", "project_id": review.ProjectID, "next_task_id": review.NextTaskID, "risk": review.Risk}, nil
}

func (s *ExecutionService) Adjust(ctx context.Context, projectID, reason string) (*actionfile.File, error) {
	review, err := s.Review(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &actionfile.File{
		Schema:    actionfile.Schema,
		Source:    "project_adjust",
		CreatedAt: time.Now().Format(time.RFC3339),
		Actions:   review.SuggestedActions,
	}, nil
}

func (s *ExecutionService) Done(ctx context.Context, projectID string) (map[string]interface{}, error) {
	p, err := s.ProjectStore.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	p.Status = project.StatusCompleted
	p.CompletedAt = &now
	p.Summary = "项目已由 taskbridge project done 标记完成"
	if err := s.ProjectStore.SaveProject(ctx, p); err != nil {
		return nil, err
	}
	return map[string]interface{}{"schema": "taskbridge.project-done.v1", "project_id": p.ID, "status": p.Status, "completed_at": p.CompletedAt, "summary": p.Summary}, nil
}

func (s *ExecutionService) Archive(ctx context.Context, projectID string) (map[string]interface{}, error) {
	p, err := s.ProjectStore.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	p.Status = project.StatusArchived
	p.ArchivedAt = &now
	if err := s.ProjectStore.SaveProject(ctx, p); err != nil {
		return nil, err
	}
	return map[string]interface{}{"schema": "taskbridge.project-archive.v1", "project_id": p.ID, "status": p.Status, "archived_at": p.ArchivedAt}, nil
}

func (s *ExecutionService) projectTasks(ctx context.Context, projectID string) ([]model.Task, error) {
	tasks, err := s.TaskStore.QueryTasks(ctx, storage.Query{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress, model.StatusDeferred, model.StatusCompleted}})
	if err != nil {
		return nil, err
	}
	result := make([]model.Task, 0)
	for _, task := range tasks {
		if task.Metadata != nil && task.Metadata.CustomFields != nil {
			if value, ok := task.Metadata.CustomFields["tb_project_id"].(string); ok && value == projectID {
				result = append(result, task)
			}
		}
	}
	return result, nil
}
