package syncaudit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

func TestDiffSavesAndLoadsAuditSession(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	taskStore, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	if err := taskStore.SaveTask(ctx, &model.Task{
		ID:          "task-1",
		Title:       "同步任务",
		Status:      model.StatusTodo,
		Source:      model.SourceLocal,
		SourceRawID: "remote-1",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	store := Store{BasePath: dir}
	session, err := store.Diff(ctx, taskStore, "local", "todoist")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if session.Schema != SessionSchema || !session.DryRun || len(session.Operations) != 1 {
		t.Fatalf("unexpected session: %+v", session)
	}
	if err := store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	loaded, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if loaded.ID != session.ID || loaded.Operations[0].LocalID != "task-1" {
		t.Fatalf("unexpected loaded session: %+v", loaded)
	}
}

func TestDiffClassifiesCreateUpdateSkipDeleteAndConflict(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	taskStore, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Now()
	saveDiffTask(t, ctx, taskStore, model.Task{ID: "src-create", Title: "Create Me", Status: model.StatusTodo, Source: model.SourceMicrosoft, CreatedAt: now, UpdatedAt: now})
	saveDiffTask(t, ctx, taskStore, model.Task{ID: "src-update", Title: "Update Me", Status: model.StatusTodo, Source: model.SourceMicrosoft, Priority: model.PriorityHigh, CreatedAt: now, UpdatedAt: now})
	saveDiffTask(t, ctx, taskStore, model.Task{ID: "src-skip", Title: "Skip Me", Status: model.StatusTodo, Source: model.SourceMicrosoft, CreatedAt: now, UpdatedAt: now})
	saveDiffTask(t, ctx, taskStore, model.Task{ID: "src-conflict", Title: "Duplicate", Status: model.StatusTodo, Source: model.SourceMicrosoft, CreatedAt: now, UpdatedAt: now})

	saveDiffTask(t, ctx, taskStore, model.Task{ID: "tgt-update", Title: "Update Me", Status: model.StatusDeferred, Source: model.SourceTodoist, Priority: model.PriorityLow, CreatedAt: now, UpdatedAt: now})
	saveDiffTask(t, ctx, taskStore, model.Task{ID: "tgt-skip", Title: "Skip Me", Status: model.StatusTodo, Source: model.SourceTodoist, CreatedAt: now, UpdatedAt: now})
	saveDiffTask(t, ctx, taskStore, model.Task{ID: "tgt-delete", Title: "Delete Me", Status: model.StatusTodo, Source: model.SourceTodoist, CreatedAt: now, UpdatedAt: now})
	saveDiffTask(t, ctx, taskStore, model.Task{ID: "tgt-conflict-a", Title: "Duplicate", Status: model.StatusTodo, Source: model.SourceTodoist, CreatedAt: now, UpdatedAt: now})
	saveDiffTask(t, ctx, taskStore, model.Task{ID: "tgt-conflict-b", Title: "Duplicate", Status: model.StatusTodo, Source: model.SourceTodoist, CreatedAt: now, UpdatedAt: now})

	session, err := Store{BasePath: dir}.Diff(ctx, taskStore, "microsoft", "todoist")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if session.Stats.Created != 1 || session.Stats.Updated != 1 || session.Stats.Deleted != 1 || session.Stats.Skipped != 1 || session.Stats.Conflicts != 1 {
		t.Fatalf("unexpected stats: %+v operations=%+v", session.Stats, session.Operations)
	}
	if op := findOperation(t, session.Operations, "update", "src-update"); len(op.Fields) == 0 {
		t.Fatalf("expected update fields, got %+v", op)
	}
	if op := findOperation(t, session.Operations, "delete", "tgt-delete"); !op.RequiresConfirmation {
		t.Fatalf("delete must require confirmation, got %+v", op)
	}
}

func TestDiffRejectsSameSourceAndTarget(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	taskStore, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	if _, err := (Store{BasePath: dir}).Diff(ctx, taskStore, "microsoft", "microsoft"); err == nil {
		t.Fatalf("expected same source/target to fail")
	}
}

func TestNewIDIsUniqueWithinSameSecond(t *testing.T) {
	first := newID("diff")
	second := newID("diff")
	if first == second {
		t.Fatalf("expected unique ids, got %s", first)
	}
}

func TestBackupCreateAndRestore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tasks.json"), []byte(`{"schema":"x","data":{"tasks":[]}}`), 0o644); err != nil {
		t.Fatalf("write tasks: %v", err)
	}
	store := Store{BasePath: dir}
	backup, err := store.CreateBackup()
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	backupID := backup["id"].(string)
	if err := os.WriteFile(filepath.Join(dir, "tasks.json"), []byte(`changed`), 0o644); err != nil {
		t.Fatalf("mutate tasks: %v", err)
	}
	if _, err := store.RestoreBackup(backupID); err != nil {
		t.Fatalf("RestoreBackup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "tasks.json"))
	if err != nil {
		t.Fatalf("read restored tasks: %v", err)
	}
	if string(data) == "changed" {
		t.Fatalf("expected backup restore to replace mutated data")
	}
}

func saveDiffTask(t *testing.T, ctx context.Context, store *filestore.FileStorage, task model.Task) {
	t.Helper()
	if err := store.SaveTask(ctx, &task); err != nil {
		t.Fatalf("SaveTask %s: %v", task.ID, err)
	}
}

func findOperation(t *testing.T, ops []Operation, opType, id string) Operation {
	t.Helper()
	for _, op := range ops {
		if op.Type == opType && op.LocalID == id {
			return op
		}
	}
	t.Fatalf("operation %s %s not found in %+v", opType, id, ops)
	return Operation{}
}
