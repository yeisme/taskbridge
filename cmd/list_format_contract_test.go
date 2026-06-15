package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

func TestListFormatContractHumanRenderers(t *testing.T) {
	cases := []struct {
		name   string
		format string
		check  func(t *testing.T, stdout string)
	}{
		{
			name:   "table",
			format: "table",
			check: func(t *testing.T, stdout string) {
				t.Helper()
				for _, want := range []string{"ID", "Title", "Status", "Write format contracts", "Review TSV output", "Total 2 tasks"} {
					if !strings.Contains(stdout, want) {
						t.Fatalf("table output missing %q:\n%s", want, stdout)
					}
				}
				if !strings.Contains(stdout, "────") {
					t.Fatalf("table output should include a human-readable header separator:\n%s", stdout)
				}
			},
		},
		{
			name:   "compact",
			format: "compact",
			check: func(t *testing.T, stdout string) {
				t.Helper()
				if strings.Contains(stdout, "─") || strings.Contains(stdout, "│") || strings.Contains(stdout, "┌") {
					t.Fatalf("compact output should not contain table borders:\n%s", stdout)
				}
				for _, want := range []string{"title|status|priority|quadrant|due_date|source|list_name", "Write format contracts|todo", "Review TSV output|in_progress"} {
					if !strings.Contains(stdout, want) {
						t.Fatalf("compact output missing %q:\n%s", want, stdout)
					}
				}
			},
		},
		{
			name:   "markdown",
			format: "markdown",
			check: func(t *testing.T, stdout string) {
				t.Helper()
				for _, want := range []string{"| ID | Title | Status |", "| --- |", "| task-1 | Write format contracts | todo |", "| task-2 | Review TSV output | in_progress |"} {
					if !strings.Contains(stdout, want) {
						t.Fatalf("markdown output missing %q:\n%s", want, stdout)
					}
				}
			},
		},
		{
			name:   "tsv",
			format: "tsv",
			check: func(t *testing.T, stdout string) {
				t.Helper()
				for _, want := range []string{"id\ttitle\tstatus", "task-1\tWrite format contracts\ttodo", "task-2\tReview TSV output\tin_progress"} {
					if !strings.Contains(stdout, want) {
						t.Fatalf("tsv output missing %q:\n%s", want, stdout)
					}
				}
				if strings.Contains(stdout, "|") || strings.Contains(stdout, "─") {
					t.Fatalf("tsv output should be tab-delimited text, not table/markdown:\n%s", stdout)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout := runListFormatContract(t, tc.format)
			assertNotJSON(t, stdout)
			tc.check(t, stdout)
		})
	}
}

func TestListFormatContractJSONRenderer(t *testing.T) {
	stdout := runListFormatContract(t, "json")
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("json format should emit valid JSON:\n%s", stdout)
	}

	var payload struct {
		Tasks []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"tasks"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("unmarshal json output: %v\n%s", err, stdout)
	}
	if payload.Total != 2 || len(payload.Tasks) != 2 {
		t.Fatalf("unexpected json task payload: %+v\n%s", payload, stdout)
	}
	// Order may differ from insertion; check by finding task-1
	foundTask1 := false
	for _, task := range payload.Tasks {
		if task.ID == "task-1" {
			if task.Title != "Write format contracts" || task.Status != "todo" {
				t.Fatalf("unexpected task-1 fields: %+v", task)
			}
			foundTask1 = true
		}
	}
	if !foundTask1 {
		t.Fatalf("task-1 not found in JSON output: %+v\n%s", payload.Tasks, stdout)
	}
}

func runListFormatContract(t *testing.T, format string) string {
	t.Helper()

	dir := t.TempDir()
	withListTestConfig(t, dir)
	listFormat = format
	seedListFormatContractTasks(t, dir)

	return captureStdout(t, func() {
		if err := runList(testListCommand(), nil); err != nil {
			t.Fatalf("runList(%s): %v", format, err)
		}
	})
}

func seedListFormatContractTasks(t *testing.T, dir string) {
	t.Helper()

	store, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Date(2026, 6, 15, 9, 0, 0, 0, time.UTC)
	due := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	for _, task := range []*model.Task{
		{
			ID:        "task-1",
			Title:     "Write format contracts",
			Status:    model.StatusTodo,
			Priority:  model.PriorityHigh,
			Quadrant:  model.QuadrantUrgentImportant,
			DueDate:   &due,
			ListName:  "Engineering",
			Source:    model.SourceLocal,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "task-2",
			Title:     "Review TSV output",
			Status:    model.StatusInProgress,
			Priority:  model.PriorityMedium,
			Quadrant:  model.QuadrantNotUrgentImportant,
			ListName:  "QA",
			Source:    model.SourceLocal,
			CreatedAt: now.Add(time.Minute),
			UpdatedAt: now.Add(time.Minute),
		},
	} {
		if err := store.SaveTask(context.Background(), task); err != nil {
			t.Fatalf("SaveTask(%s): %v", task.ID, err)
		}
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
}

func assertNotJSON(t *testing.T, stdout string) {
	t.Helper()
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		t.Fatal("expected human output, got empty stdout")
	}
	if json.Valid([]byte(trimmed)) {
		t.Fatalf("human format should not be valid JSON:\n%s", stdout)
	}
}
