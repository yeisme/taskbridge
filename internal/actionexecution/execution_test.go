package actionexecution

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/actionaudit"
	"github.com/yeisme/taskbridge/internal/actionfile"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

func newTaskStore(t *testing.T) *filestore.FileStorage {
	t.Helper()
	store, err := filestore.New(t.TempDir(), "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	return store
}

func seedTask(t *testing.T, store *filestore.FileStorage, id string) {
	t.Helper()
	now := time.Now()
	if err := store.SaveTask(context.Background(), &model.Task{ID: id, Title: id, Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
}

func okActionFile() *actionfile.File {
	return &actionfile.File{Schema: actionfile.Schema, Actions: []actionfile.Action{
		{ID: "a1", Type: "complete_task", TaskID: "task-1"},
	}}
}

// TestExecuteDefaultSessionIDWhenEmpty verifies the facade generates an
// action_<timestamp> session id when none is supplied.
func TestExecuteDefaultSessionIDWhenEmpty(t *testing.T) {
	store := newTaskStore(t)
	seedTask(t, store, "task-1")
	fixed := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	service := Service{TaskStore: store, Now: func() time.Time { return fixed }}

	res := service.Execute(context.Background(), okActionFile(), Options{Command: "test", Confirm: true})

	if want := "action_20260616_120000"; res.Receipt.SessionID != want {
		t.Fatalf("default session id = %q, want %q", res.Receipt.SessionID, want)
	}
}

// TestExecuteNilAuditStoreSkipsWrite verifies the facade does not attempt to
// persist a receipt (and does not panic) when no audit store is configured.
func TestExecuteNilAuditStoreSkipsWrite(t *testing.T) {
	store := newTaskStore(t)
	seedTask(t, store, "task-1")
	service := Service{TaskStore: store, AuditStore: nil}

	res := service.Execute(context.Background(), okActionFile(), Options{SessionID: "s1", Command: "test", Confirm: true})

	if res.ReceiptSaved {
		t.Fatal("ReceiptSaved should be false when AuditStore is nil")
	}
	if res.AuditWriteErr != nil {
		t.Fatalf("unexpected AuditWriteErr: %v", res.AuditWriteErr)
	}
}

// TestExecuteAuditWriteFailureMarksError verifies a failing audit write is
// surfaced on the result without losing the execution outcome.
func TestExecuteAuditWriteFailureMarksError(t *testing.T) {
	store := newTaskStore(t)
	seedTask(t, store, "task-1")
	// basePath pointing at an existing file forces MkdirAll to fail inside Save.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	service := Service{TaskStore: store, AuditStore: actionaudit.NewStore(blocker)}

	res := service.Execute(context.Background(), okActionFile(), Options{SessionID: "s1", Command: "test", Confirm: true})

	if res.ReceiptSaved {
		t.Fatal("ReceiptSaved should be false when audit write fails")
	}
	if res.AuditWriteErr == nil {
		t.Fatal("expected AuditWriteErr to be set")
	}
	if res.Execution.Status != "error" {
		t.Fatalf("execution status = %q, want error", res.Execution.Status)
	}
	if len(res.Execution.Errors) == 0 {
		t.Fatal("expected execution errors to record the audit write failure")
	}
}

// TestExecuteOperationsAndStats verifies per-action outcome derivation flows
// into receipt operations/stats, including dry-run zeroing Confirmed.
func TestExecuteOperationsAndStats(t *testing.T) {
	file := &actionfile.File{Schema: actionfile.Schema, Actions: []actionfile.Action{
		{ID: "a1", Type: "complete_task", TaskID: "task-1"},
		{ID: "a2", Type: "complete_task", TaskID: "missing"},
	}}

	// Confirmed execution: one applied, one failed.
	confirmedStore := newTaskStore(t)
	seedTask(t, confirmedStore, "task-1")
	confirmed := Service{TaskStore: confirmedStore, AuditStore: actionaudit.NewStore(t.TempDir())}
	res := confirmed.Execute(context.Background(), file, Options{SessionID: "s1", Command: "test", Confirm: true})

	if len(res.Receipt.Operations) != 2 {
		t.Fatalf("operations = %d, want 2", len(res.Receipt.Operations))
	}
	if res.Receipt.Operations[0].Status != "applied" || res.Receipt.Operations[1].Status != "failed" {
		t.Fatalf("operation statuses = %q, %q; want applied, failed", res.Receipt.Operations[0].Status, res.Receipt.Operations[1].Status)
	}
	if res.Receipt.Stats.Confirmed != 1 {
		t.Fatalf("confirmed stats.Confirmed = %d, want 1", res.Receipt.Stats.Confirmed)
	}

	// Dry-run execution: Confirmed must be zero even though outcomes exist.
	dryRunStore := newTaskStore(t)
	seedTask(t, dryRunStore, "task-1")
	dryRun := Service{TaskStore: dryRunStore, AuditStore: actionaudit.NewStore(t.TempDir())}
	dryRes := dryRun.Execute(context.Background(), file, Options{SessionID: "s2", Command: "test", DryRun: true})
	if dryRes.Receipt.Stats.Confirmed != 0 {
		t.Fatalf("dry-run stats.Confirmed = %d, want 0", dryRes.Receipt.Stats.Confirmed)
	}
}
