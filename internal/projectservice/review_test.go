package projectservice

import (
	"context"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

func TestProjectReviewAndLifecycle(t *testing.T) {
	ctx := context.Background()
	taskStore, err := filestore.New(t.TempDir(), "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	projectStore, err := project.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("project.NewFileStore: %v", err)
	}
	projectID := "proj-1"
	if err := projectStore.SaveProject(ctx, &project.Project{ID: projectID, Name: "项目", GoalText: "项目", GoalType: project.GoalTypeGeneric, Status: project.StatusActive}); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}
	if err := taskStore.SaveTask(ctx, &model.Task{
		ID:               "task-large",
		Title:            "大任务",
		Status:           model.StatusTodo,
		Source:           model.SourceLocal,
		EstimatedMinutes: 240,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{"tb_project_id": projectID}},
	}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	service := ExecutionService{TaskStore: taskStore, ProjectStore: projectStore}
	review, err := service.Review(ctx, projectID)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if review.Progress["total_tasks"] != 1 || review.Progress["blocked_tasks"] != 1 {
		t.Fatalf("unexpected progress: %+v", review.Progress)
	}
	if len(review.SuggestedActions) != 1 || review.SuggestedActions[0].Type != "split_task" {
		t.Fatalf("expected split action, got %+v", review.SuggestedActions)
	}

	done, err := service.Done(ctx, projectID)
	if err != nil {
		t.Fatalf("Done: %v", err)
	}
	if done["status"] != project.StatusCompleted {
		t.Fatalf("unexpected done result: %+v", done)
	}

	archived, err := service.Archive(ctx, projectID)
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if archived["status"] != project.StatusArchived {
		t.Fatalf("unexpected archive result: %+v", archived)
	}
}
