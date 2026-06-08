package controlplane

import (
	"testing"
	"time"
)

func TestMockTasks(t *testing.T) {
	now := time.Now()
	tasks := MockTasks(now)

	if len(tasks) != 10 {
		t.Fatalf("expected 10 mock tasks, got %d", len(tasks))
	}

	// Verify IDs are unique
	seen := map[string]bool{}
	for _, task := range tasks {
		if seen[task.ID] {
			t.Fatalf("duplicate task ID: %s", task.ID)
		}
		seen[task.ID] = true
	}

	// Verify we have at least one task of each type we care about
	hasTodo := false
	hasInProgress := false
	hasCompleted := false
	sources := map[string]bool{}

	for _, task := range tasks {
		sources[string(task.Source)] = true
		switch task.Status {
		case "todo":
			hasTodo = true
		case "in_progress":
			hasInProgress = true
		case "completed":
			hasCompleted = true
		}
	}

	if !hasTodo {
		t.Error("expected at least one todo task")
	}
	if !hasInProgress {
		t.Error("expected at least one in_progress task")
	}
	if !hasCompleted {
		t.Error("expected at least one completed task")
	}

	// Verify multiple sources
	expectedSources := []string{"local", "google", "microsoft", "todoist"}
	for _, s := range expectedSources {
		if !sources[s] {
			t.Errorf("expected source %q in mock tasks", s)
		}
	}
}

func TestNewMockService(t *testing.T) {
	now := time.Now()
	svc := NewMockService(now)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.TaskStore == nil {
		t.Fatal("expected non-nil TaskStore")
	}
}

func TestMockServiceToday(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	svc := NewMockService(now)
	ctx := t.Context()

	result, err := svc.Today(ctx, Options{Now: now})
	if err != nil {
		t.Fatalf("Today() error: %v", err)
	}

	if result.Schema != SchemaToday {
		t.Errorf("expected schema %q, got %q", SchemaToday, result.Schema)
	}
	if result.Date != "2026-06-05" {
		t.Errorf("expected date 2026-06-05, got %q", result.Date)
	}
	if result.Status != "ok" {
		t.Errorf("expected status ok, got %q", result.Status)
	}

	// Should have must_do section with tasks (today + overdue)
	mustDo := sectionByID(result.Sections, "must_do")
	if mustDo == nil {
		t.Fatal("expected must_do section")
	}
	if len(mustDo.Tasks) == 0 {
		t.Error("expected at least one must_do task")
	}

	// Should have suggested actions for overdue tasks
	if len(result.SuggestedActions) == 0 {
		t.Error("expected at least one suggested action for overdue task")
	}
}

func TestMockServiceNext(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	svc := NewMockService(now)
	ctx := t.Context()

	result, err := svc.Next(ctx, Options{Now: now, Limit: 3})
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}

	if result.Schema != SchemaNext {
		t.Errorf("expected schema %q, got %q", SchemaNext, result.Schema)
	}
	if result.Count > 3 {
		t.Errorf("expected at most 3 tasks with limit, got %d", result.Count)
	}
}

func TestMockServiceInbox(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	svc := NewMockService(now)
	ctx := t.Context()

	result, err := svc.Inbox(ctx, Options{Now: now})
	if err != nil {
		t.Fatalf("Inbox() error: %v", err)
	}

	if result.Schema != SchemaInbox {
		t.Errorf("expected schema %q, got %q", SchemaInbox, result.Schema)
	}
	if result.Count == 0 {
		t.Error("expected at least one inbox task")
	}
}

func TestMockServiceReview(t *testing.T) {
	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	svc := NewMockService(now)
	ctx := t.Context()

	result, err := svc.Review(ctx, Options{Now: now})
	if err != nil {
		t.Fatalf("Review() error: %v", err)
	}

	if result.Schema != SchemaReview {
		t.Errorf("expected schema %q, got %q", SchemaReview, result.Schema)
	}
	if result.Summary["active"] == 0 {
		t.Error("expected at least one active task in review summary")
	}
	if len(result.SuggestedActions) == 0 {
		t.Error("expected suggested actions in review")
	}
}

func TestMockTasksZeroTime(t *testing.T) {
	// Should not panic with zero time
	tasks := MockTasks(time.Time{})
	if len(tasks) != 10 {
		t.Fatalf("expected 10 tasks with zero time, got %d", len(tasks))
	}
}

func sectionByID(sections []Section, id string) *Section {
	for i := range sections {
		if sections[i].ID == id {
			return &sections[i]
		}
	}
	return nil
}
