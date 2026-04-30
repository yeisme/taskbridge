package actionfile

import (
	"context"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

func TestExecuteDryRunDoesNotMutateTask(t *testing.T) {
	ctx := context.Background()
	store := newTaskStore(t)
	task := &model.Task{ID: "task-1", Title: "任务", Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.SaveTask(ctx, task); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	result := Executor{TaskStore: store}.Execute(ctx, &File{Schema: Schema, Actions: []Action{{
		ID:                   "act-1",
		Type:                 "complete_task",
		TaskID:               "task-1",
		RequiresConfirmation: true,
	}}}, ExecuteOptions{DryRun: true})
	if result.Status != "ok" || result.Updated != 1 || result.RequiresConfirmation {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	loaded, err := store.GetTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if loaded.Status != model.StatusTodo || loaded.CompletedAt != nil {
		t.Fatalf("dry-run mutated task: %+v", loaded)
	}
}

func TestExecuteRequiresConfirmationForDangerousActions(t *testing.T) {
	ctx := context.Background()
	store := newTaskStore(t)
	if err := store.SaveTask(ctx, &model.Task{ID: "task-1", Title: "任务", Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	result := Executor{TaskStore: store}.Execute(ctx, &File{Schema: Schema, Actions: []Action{{
		ID:     "act-1",
		Type:   "complete_task",
		TaskID: "task-1",
	}}}, ExecuteOptions{})
	if !result.RequiresConfirmation || result.Updated != 0 || result.Skipped != 1 {
		t.Fatalf("expected confirmation gate, got %+v", result)
	}
}

func TestExecuteConfirmMutatesTask(t *testing.T) {
	ctx := context.Background()
	store := newTaskStore(t)
	if err := store.SaveTask(ctx, &model.Task{ID: "task-1", Title: "任务", Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: time.Now(), UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	result := Executor{TaskStore: store}.Execute(ctx, &File{Schema: Schema, Actions: []Action{{
		ID:     "act-1",
		Type:   "complete_task",
		TaskID: "task-1",
	}}}, ExecuteOptions{Confirm: true})
	if result.Status != "ok" || result.Updated != 1 || result.Skipped != 0 {
		t.Fatalf("unexpected execute result: %+v", result)
	}
	loaded, err := store.GetTask(ctx, "task-1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if loaded.Status != model.StatusCompleted || loaded.CompletedAt == nil {
		t.Fatalf("expected completed task, got %+v", loaded)
	}
}

func newTaskStore(t *testing.T) *filestore.FileStorage {
	t.Helper()
	store, err := filestore.New(t.TempDir(), "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	return store
}
