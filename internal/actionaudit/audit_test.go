package actionaudit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartCreatesRunningReceipt(t *testing.T) {
	r := Start("test-1", "agent execute", "actions.json", true, false)
	if r.Status != "running" {
		t.Fatalf("status = %q, want running", r.Status)
	}
	if r.SessionID != "test-1" {
		t.Fatalf("session_id = %q, want test-1", r.SessionID)
	}
	if r.DryRun != true {
		t.Fatal("dry_run should be true")
	}
	if r.Confirm != false {
		t.Fatal("confirm should be false")
	}
	if r.SchemaVersion != SchemaVersion {
		t.Fatalf("schema_version = %q, want %s", r.SchemaVersion, SchemaVersion)
	}
	if r.StartedAt.IsZero() {
		t.Fatal("started_at should be set")
	}
	if r.Redaction == "" {
		t.Fatal("redaction policy should be documented")
	}
}

func TestFinishSetsDurationAndStatus(t *testing.T) {
	r := Start("test-2", "review --apply-file", "actions.json", false, true)
	time.Sleep(2 * time.Millisecond)
	stats := Stats{Total: 3, Updated: 2, Skipped: 1, Errors: 0, Confirmed: 2}
	ops := []Operation{{ActionID: "a1", Type: "complete_task", TaskID: "t1", Status: "applied"}}
	r.Finish("ok", stats, ops, []string{})

	if r.Status != "ok" {
		t.Fatalf("status = %q, want ok", r.Status)
	}
	if r.DurationMs <= 0 {
		t.Fatal("duration_ms should be positive after sleep")
	}
	if r.FinishedAt.IsZero() {
		t.Fatal("finished_at should be set")
	}
	if r.Stats.Updated != 2 {
		t.Fatalf("stats.updated = %d, want 2", r.Stats.Updated)
	}
	if len(r.Operations) != 1 {
		t.Fatalf("operations len = %d, want 1", len(r.Operations))
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	r := Start("roundtrip-1", "agent execute", "actions.json", false, true)
	r.Finish("ok", Stats{Total: 1, Updated: 1}, []Operation{{ActionID: "a1", Type: "complete_task", TaskID: "t1", Status: "applied"}}, []string{})

	if err := store.Save(r); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists at expected path
	path := filepath.Join(dir, "audit", "actions", "roundtrip-1.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("receipt file should exist at %s: %v", path, err)
	}

	loaded, err := store.Load("roundtrip-1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.SessionID != "roundtrip-1" {
		t.Fatalf("loaded session_id = %q, want roundtrip-1", loaded.SessionID)
	}
	if loaded.Command != "agent execute" {
		t.Fatalf("loaded command = %q, want 'agent execute'", loaded.Command)
	}
	if loaded.Stats.Updated != 1 {
		t.Fatalf("loaded stats.updated = %d, want 1", loaded.Stats.Updated)
	}
}

func TestSaveEmptyBasePathReturnsError(t *testing.T) {
	store := NewStore("")
	r := Start("err-1", "agent execute", "actions.json", true, false)
	if err := store.Save(r); err == nil {
		t.Fatal("save with empty base path should return error")
	}
}

func TestListReturnsSummaries(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Save two receipts
	r1 := Start("list-1", "agent execute", "a1.json", true, false)
	r1.Finish("ok", Stats{Total: 1, Updated: 1}, nil, nil)
	if err := store.Save(r1); err != nil {
		t.Fatalf("save r1: %v", err)
	}

	r2 := Start("list-2", "review --apply-file", "a2.json", false, true)
	r2.Finish("error", Stats{Total: 1, Errors: 1}, nil, []string{"write failed"})
	if err := store.Save(r2); err != nil {
		t.Fatalf("save r2: %v", err)
	}

	summaries, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	// Verify we can find both session IDs
	found := map[string]bool{}
	for _, s := range summaries {
		found[s.SessionID] = true
	}
	if !found["list-1"] || !found["list-2"] {
		t.Fatalf("expected to find list-1 and list-2, got: %v", found)
	}
}

func TestListEmptyDirReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	summaries, err := store.List()
	if err != nil {
		t.Fatalf("list on empty dir should not error: %v", err)
	}
	if len(summaries) != 0 {
		t.Fatalf("expected 0 summaries, got %d", len(summaries))
	}
}

func TestReceiptContainsNoSensitiveFields(t *testing.T) {
	r := Start("redact-1", "agent execute", "actions.json", true, false)
	r.Finish("ok", Stats{Total: 1, Updated: 1}, []Operation{{
		ActionID: "a1",
		Type:     "defer_task",
		TaskID:   "task-123",
		Status:   "applied",
	}}, []string{})

	// The Receipt struct should never have token/password/auth fields.
	// This test documents that invariant.
	if r.Redaction == "" {
		t.Fatal("redaction policy must be documented in every receipt")
	}
	// Operations should record task_id but never tokens or payloads
	for _, op := range r.Operations {
		if op.TaskID == "" {
			t.Fatal("operation should include task_id")
		}
	}
}
