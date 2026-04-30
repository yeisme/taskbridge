package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	projectstore "github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

func TestTodayClassifiesTasksAndProjectNext(t *testing.T) {
	ctx := context.Background()
	taskStore, err := filestore.New(t.TempDir(), "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	projectStore, err := projectstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("project.NewFileStore: %v", err)
	}

	now := time.Date(2026, 4, 30, 10, 0, 0, 0, time.Local)
	projectID := "proj-control"
	if err := projectStore.SaveProject(ctx, &project.Project{
		ID:       projectID,
		Name:     "控制面项目",
		GoalText: "控制面项目",
		GoalType: project.GoalTypeGeneric,
		Status:   project.StatusActive,
	}); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	saveTask(t, taskStore, &model.Task{
		ID:               "today",
		Title:            "今天完成",
		Status:           model.StatusTodo,
		Source:           model.SourceLocal,
		Priority:         model.PriorityHigh,
		DueDate:          ptr(now),
		CreatedAt:        now.Add(-24 * time.Hour),
		UpdatedAt:        now,
		EstimatedMinutes: 60,
		Metadata:         projectMetadata(projectID),
	})
	saveTask(t, taskStore, &model.Task{
		ID:        "overdue",
		Title:     "逾期处理",
		Status:    model.StatusTodo,
		Source:    model.SourceLocal,
		Priority:  model.PriorityMedium,
		DueDate:   ptr(now.AddDate(0, 0, -2)),
		CreatedAt: now.AddDate(0, 0, -3),
		UpdatedAt: now,
	})
	saveTask(t, taskStore, &model.Task{
		ID:        "inbox",
		Title:     "待整理",
		Status:    model.StatusTodo,
		Source:    model.SourceLocal,
		Priority:  model.PriorityLow,
		CreatedAt: now,
		UpdatedAt: now,
	})

	result, err := (&Service{TaskStore: taskStore, ProjectStore: projectStore}).Today(ctx, Options{Now: now})
	if err != nil {
		t.Fatalf("Today: %v", err)
	}
	if result.Schema != SchemaToday || result.Status != "ok" {
		t.Fatalf("unexpected envelope: %+v", result)
	}
	if result.Summary["must_do"] != 2 || result.Summary["overdue"] != 1 || result.Summary["inbox"] != 1 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if len(result.ProjectNext) != 1 || result.ProjectNext[0].ProjectID != projectID {
		t.Fatalf("expected project next for %s, got %+v", projectID, result.ProjectNext)
	}
	if len(result.SuggestedActions) != 1 || result.SuggestedActions[0].TaskID != "overdue" {
		t.Fatalf("expected overdue suggested action, got %+v", result.SuggestedActions)
	}
}

func TestReviewSuggestsSplittingLargeTasks(t *testing.T) {
	ctx := context.Background()
	taskStore, err := filestore.New(t.TempDir(), "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Date(2026, 4, 30, 10, 0, 0, 0, time.Local)
	saveTask(t, taskStore, &model.Task{
		ID:               "large",
		Title:            "过大任务",
		Status:           model.StatusTodo,
		Source:           model.SourceLocal,
		Priority:         model.PriorityHigh,
		CreatedAt:        now,
		UpdatedAt:        now,
		EstimatedMinutes: 240,
	})

	result, err := (&Service{TaskStore: taskStore}).Review(ctx, Options{Now: now})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if result.Summary["large"] != 1 {
		t.Fatalf("expected one large task, got %+v", result.Summary)
	}
	if len(result.SuggestedActions) != 1 || result.SuggestedActions[0].Type != "split_task" {
		t.Fatalf("expected split action, got %+v", result.SuggestedActions)
	}
}

func saveTask(t *testing.T, store *filestore.FileStorage, task *model.Task) {
	t.Helper()
	if err := store.SaveTask(context.Background(), task); err != nil {
		t.Fatalf("SaveTask(%s): %v", task.ID, err)
	}
}

func projectMetadata(projectID string) *model.TaskMetadata {
	return &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{"tb_project_id": projectID}}
}

func ptr(t time.Time) *time.Time {
	return &t
}
