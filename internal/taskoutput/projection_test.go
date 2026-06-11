package taskoutput

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
)

// toPtrString is a test helper that returns a pointer to the given string.
func toPtrString(s string) *string {
	return &s
}

func TestToTaskProjection_BasicFields(t *testing.T) {
	now := time.Now()
	task := model.Task{
		ID:            "task-1",
		Title:         "Test task",
		Status:        model.StatusTodo,
		Source:        model.SourceLocal,
		DueDate:       &now,
		Priority:      model.PriorityHigh,
		Quadrant:      model.QuadrantUrgentImportant,
		ListName:      "My List",
		ListID:        "list-1",
		Tags:          []string{"tag1", "tag2"},
		Progress:      50,
		PriorityScore: 80,
	}

	p := ToTaskProjection(task)

	if p.ID != "task-1" {
		t.Errorf("ID = %q, want %q", p.ID, "task-1")
	}
	if p.Title != "Test task" {
		t.Errorf("Title = %q, want %q", p.Title, "Test task")
	}
	if p.Status != "todo" {
		t.Errorf("Status = %q, want %q", p.Status, "todo")
	}
	if p.Source != "local" {
		t.Errorf("Source = %q, want %q", p.Source, "local")
	}
	if p.DueDate != now.Format("2006-01-02") {
		t.Errorf("DueDate = %q, want %q", p.DueDate, now.Format("2006-01-02"))
	}
	if p.Priority != "P1-高" {
		t.Errorf("Priority = %q, want %q", p.Priority, "P1-高")
	}
	if p.Quadrant != "Q1-紧急重要" {
		t.Errorf("Quadrant = %q, want %q", p.Quadrant, "Q1-紧急重要")
	}
	if p.ListName != "My List" {
		t.Errorf("ListName = %q, want %q", p.ListName, "My List")
	}
	if len(p.Tags) != 2 || p.Tags[0] != "tag1" {
		t.Errorf("Tags = %v, want [tag1 tag2]", p.Tags)
	}
	if p.Progress != 50 {
		t.Errorf("Progress = %d, want 50", p.Progress)
	}
	if p.PriorityScore != 80 {
		t.Errorf("PriorityScore = %d, want 80", p.PriorityScore)
	}
}

func TestToTaskProjection_NilFields(t *testing.T) {
	task := model.Task{
		ID:     "task-2",
		Title:  "Minimal task",
		Status: model.StatusTodo,
		Source: model.SourceLocal,
	}

	p := ToTaskProjection(task)

	if p.DueDate != "" {
		t.Errorf("DueDate = %q, want empty", p.DueDate)
	}
	if p.ParentID != "" {
		t.Errorf("ParentID = %q, want empty", p.ParentID)
	}
	if p.Quadrant != "" {
		t.Errorf("Quadrant = %q, want empty for zero-value quadrant", p.Quadrant)
	}
}

func TestToTaskProjection_ParentID(t *testing.T) {
	task := model.Task{
		ID:       "task-3",
		Title:    "Subtask",
		Status:   model.StatusTodo,
		Source:   model.SourceLocal,
		ParentID: toPtrString("parent-1"),
	}

	p := ToTaskProjection(task)

	if p.ParentID != "parent-1" {
		t.Errorf("ParentID = %q, want %q", p.ParentID, "parent-1")
	}
}

func TestTaskJSONRows_EmptyFields(t *testing.T) {
	tasks := []model.Task{
		{ID: "1", Title: "A", Status: model.StatusTodo, Source: model.SourceLocal},
		{ID: "2", Title: "B", Status: model.StatusCompleted, Source: model.SourceLocal},
	}

	result := TaskJSONRows(tasks, nil)

	slice, ok := result.([]TaskProjection)
	if !ok {
		t.Fatalf("expected []TaskProjection, got %T", result)
	}
	if len(slice) != 2 {
		t.Fatalf("len = %d, want 2", len(slice))
	}
	if slice[0].ID != "1" {
		t.Errorf("slice[0].ID = %q, want %q", slice[0].ID, "1")
	}
}

func TestTaskJSONRows_WithFields(t *testing.T) {
	tasks := []model.Task{
		{ID: "1", Title: "A", Status: model.StatusTodo, Source: model.SourceLocal},
	}

	result := TaskJSONRows(tasks, []string{"id", "title"})

	maps, ok := result.([]map[string]interface{})
	if !ok {
		t.Fatalf("expected []map[string]interface{}, got %T", result)
	}
	if len(maps) != 1 {
		t.Fatalf("len = %d, want 1", len(maps))
	}
	if maps[0]["id"] != "1" {
		t.Errorf("id = %v, want %q", maps[0]["id"], "1")
	}
	if maps[0]["title"] != "A" {
		t.Errorf("title = %v, want %q", maps[0]["title"], "A")
	}
}

func TestPrintTasksJSON_Parseable(t *testing.T) {
	tasks := []model.Task{
		{ID: "1", Title: "JSON test", Status: model.StatusTodo, Source: model.SourceLocal},
	}

	var buf strings.Builder
	err := PrintTasksJSON(&buf, tasks, nil)
	if err != nil {
		t.Fatalf("PrintTasksJSON error: %v", err)
	}

	output := buf.String()
	// Output should be parseable as JSON array
	if !strings.HasPrefix(strings.TrimSpace(output), "[") {
		t.Errorf("JSON output should start with '[', got: %s", output[:min(len(output), 40)])
	}
}

func TestPrintTasksCompact_OutputNotEmpty(t *testing.T) {
	tasks := []model.Task{
		{ID: "1", Title: "Compact test", Status: model.StatusTodo, Source: model.SourceLocal},
	}

	var buf strings.Builder
	PrintTasksCompact(&buf, tasks, nil)

	output := buf.String()
	if !strings.Contains(output, "Compact test") {
		t.Errorf("compact output should contain task title, got: %s", output)
	}
	if !strings.Contains(output, "|") {
		t.Errorf("compact output should contain pipe delimiters")
	}
}

func TestPrintTasksTSV_OutputNotEmpty(t *testing.T) {
	tasks := []model.Task{
		{ID: "1", Title: "TSV test", Status: model.StatusTodo, Source: model.SourceLocal},
	}

	var buf strings.Builder
	PrintTasksTSV(&buf, tasks, nil)

	output := buf.String()
	if !strings.Contains(output, "TSV test") {
		t.Errorf("TSV output should contain task title, got: %s", output)
	}
	if !strings.Contains(output, "\t") {
		t.Errorf("TSV output should contain tab delimiters")
	}
}

func TestPrintTasksMarkdown_OutputNotEmpty(t *testing.T) {
	tasks := []model.Task{
		{ID: "1", Title: "MD test", Status: model.StatusTodo, Source: model.SourceLocal, Quadrant: model.QuadrantUrgentImportant},
	}

	var buf strings.Builder
	PrintTasksMarkdown(&buf, tasks)

	output := buf.String()
	if !strings.Contains(output, "MD test") {
		t.Errorf("markdown output should contain task title, got: %s", output)
	}
	if !strings.Contains(output, "# 📋") {
		t.Errorf("markdown output should contain heading")
	}
}

func TestPrintTasksTableUsesEnglishChrome(t *testing.T) {
	var buf bytes.Buffer
	PrintTasksTable(&buf, []model.Task{{
		ID:       "task-1",
		Title:    "Example",
		Status:   model.StatusTodo,
		Quadrant: model.QuadrantUrgentImportant,
		Priority: model.PriorityUrgent,
	}})
	out := buf.String()
	for _, want := range []string{"Title", "Status", "Quadrant", "Priority", "Due", "Total 1 tasks", "Todo", "Q1 Urgent", "P0 Urgent"} {
		if !strings.Contains(out, want) {
			t.Fatalf("PrintTasksTable missing %q:\n%s", want, out)
		}
	}
	for _, disallowed := range []string{"标题", "状态", "象限", "优先级", "截止日期", "共 1 个任务", "待办", "紧急重要"} {
		if strings.Contains(out, disallowed) {
			t.Fatalf("PrintTasksTable should use English chrome, found %q:\n%s", disallowed, out)
		}
	}
}

func TestStatusShort(t *testing.T) {
	tests := []struct {
		status model.TaskStatus
		want   string
	}{
		{model.StatusTodo, "Todo"},
		{model.StatusInProgress, "In progress"},
		{model.StatusCompleted, "Completed"},
		{model.StatusCancelled, "Cancelled"},
		{model.StatusDeferred, "Deferred"},
		{model.TaskStatus("unknown"), "unknown"},
	}
	for _, tt := range tests {
		got := StatusShort(tt.status)
		if got != tt.want {
			t.Errorf("StatusShort(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestTruncateDisplay(t *testing.T) {
	tests := []struct {
		s        string
		maxWidth int
		want     string
	}{
		{"short", 10, "short"},
		{"", 5, ""},
		{"abc", 0, ""},
		{"a very long title that should be truncated", 20, "a very long title..."},
		{"中文标题测试截断功能是否正常", 10, "中文标..."},
	}
	for _, tt := range tests {
		got := TruncateDisplay(tt.s, tt.maxWidth)
		if got != tt.want {
			t.Errorf("TruncateDisplay(%q, %d) = %q, want %q", tt.s, tt.maxWidth, got, tt.want)
		}
	}
}

func TestNewTaskBrowseProjectionWrapsPageData(t *testing.T) {
	tasks := []model.Task{{ID: "task-1", Title: "One", Status: model.StatusTodo, Source: model.SourceLocal}}

	projection := NewTaskBrowseProjection("task.list", tasks, nil, 2, 1, 0)

	if projection.Command != "task.list" {
		t.Fatalf("Command = %q, want task.list", projection.Command)
	}
	if projection.Facts["total"] != 2 || projection.Facts["count"] != 1 || projection.Facts["has_more"] != true {
		t.Fatalf("unexpected facts: %+v", projection.Facts)
	}
	page, ok := projection.Data.(TaskBrowsePage)
	if !ok {
		t.Fatalf("Data = %T, want TaskBrowsePage", projection.Data)
	}
	if page.Total != 2 || page.Limit != 1 || !page.HasMore {
		t.Fatalf("unexpected page metadata: %+v", page)
	}
}

func TestNewTaskListsProjectionWrapsSummaries(t *testing.T) {
	summaries := []TaskListSummary{{Provider: "local", ListID: "inbox", ListName: "Inbox", TaskCountLocal: 3}}

	projection := NewTaskListsProjection("task.lists", summaries)

	if projection.Command != "task.lists" {
		t.Fatalf("Command = %q, want task.lists", projection.Command)
	}
	if projection.Facts["total"] != 1 {
		t.Fatalf("unexpected facts: %+v", projection.Facts)
	}
	data, ok := projection.Data.(TaskListsData)
	if !ok {
		t.Fatalf("Data = %T, want TaskListsData", projection.Data)
	}
	if len(data.Lists) != 1 || data.Lists[0].ListID != "inbox" {
		t.Fatalf("unexpected data: %+v", data)
	}
}

func TestEmptyProjectionSummariesUseEnglishHints(t *testing.T) {
	tasks := NewTaskBrowseProjection("task.list", nil, nil, 0, 0, 0)
	lists := NewTaskListsProjection("task.lists", nil)

	for name, projection := range map[string]string{"tasks": tasks.Summary, "lists": lists.Summary} {
		if strings.Contains(projection, "未") || strings.Contains(projection, "📭") {
			t.Fatalf("%s summary should be English text only, got %q", name, projection)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
