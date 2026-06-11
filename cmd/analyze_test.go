package cmd

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
	"github.com/yeisme/taskbridge/pkg/config"
)

func TestRenderPriorityAnalysisStatsPanel(t *testing.T) {
	analysis := PriorityAnalysis{
		Urgent:  PriorityData{Count: 1, Percentage: 25},
		High:    PriorityData{Count: 1, Percentage: 25},
		Medium:  PriorityData{Count: 1, Percentage: 25},
		Low:     PriorityData{Count: 1, Percentage: 25},
		None:    PriorityData{Count: 0, Percentage: 0},
		Summary: SummaryData{Total: 5, Active: 4, Completed: 1},
	}

	out := renderPriorityAnalysis(analysis)
	for _, want := range []string{"Prioritization analysis", "Urgent (P0)", "High (P1)", "Medium (P2)", "Low (P3)", "No priority", "Total: 5 tasks | Active: 4 | Completed: 1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("renderPriorityAnalysis missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatalf("renderPriorityAnalysis should use stats panel box:\n%s", out)
	}
}

func TestRenderPriorityAnalysisZeroStillShowsTotalsFooter(t *testing.T) {
	out := renderPriorityAnalysis(PriorityAnalysis{})
	if !strings.Contains(out, "Total: 0 tasks | Active: 0 | Completed: 0") {
		t.Fatalf("zero priority analysis should show total/active/completed footer:\n%s", out)
	}
}

func TestRenderPriorityAnalysisIncludesAverageScoreInPanelFooter(t *testing.T) {
	analysis := PriorityAnalysis{
		Urgent:  PriorityData{Count: 1, Percentage: 100},
		Summary: SummaryData{Total: 1, Active: 1, AvgPriorityScore: 87.5},
	}

	out := renderPriorityAnalysis(analysis)
	if !strings.Contains(out, "Average priority score: 87.5") {
		t.Fatalf("renderPriorityAnalysis should include average score in panel footer:\n%s", out)
	}
}

func TestAnalyzePriorityDefaultOutputIsHumanPanelNotJSON(t *testing.T) {
	withAnalyzeTestConfig(t, t.TempDir())
	seedAnalyzeTasks(t, []model.Task{
		analyzeTask("urgent", model.PriorityUrgent, model.QuadrantUrgentImportant, 90, model.StatusTodo, nil, nil),
		analyzeTask("done", model.PriorityLow, model.QuadrantNotUrgentNotImportant, 0, model.StatusCompleted, nil, nil),
	})

	stdout := captureStdout(t, func() {
		if err := runAnalyzePriority(nil, nil); err != nil {
			t.Fatalf("runAnalyzePriority: %v", err)
		}
	})

	if strings.HasPrefix(strings.TrimSpace(stdout), "{") {
		t.Fatalf("default analyze priority output should not be JSON: %s", stdout)
	}
	for _, want := range []string{"Prioritization analysis", "╭", "Urgent (P0)", "Average priority score: 90.0"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("default analyze priority output missing %q:\n%s", want, stdout)
		}
	}
}

func TestAnalyzePriorityLegacyJSONIsParseable(t *testing.T) {
	withAnalyzeTestConfig(t, t.TempDir())
	seedAnalyzeTasks(t, []model.Task{
		analyzeTask("high", model.PriorityHigh, model.QuadrantNotUrgentImportant, 42, model.StatusTodo, nil, nil),
	})
	analyzeFormat = "json"

	stdout := captureStdout(t, func() {
		if err := runAnalyzePriority(nil, nil); err != nil {
			t.Fatalf("runAnalyzePriority: %v", err)
		}
	})

	var parsed PriorityAnalysis
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("legacy JSON should be parseable: %v\noutput: %s", err, stdout)
	}
	if parsed.High.Count != 1 || parsed.Summary.AvgPriorityScore != 42 {
		t.Fatalf("unexpected legacy JSON payload: %#v", parsed)
	}
}

func TestAnalyzePriorityGlobalJSONUsesEnvelope(t *testing.T) {
	withAnalyzeTestConfig(t, t.TempDir())
	oldJSON := outputJSON
	outputJSON = true
	t.Cleanup(func() { outputJSON = oldJSON })
	stdout := captureStdout(t, func() {
		if err := runAnalyzePriority(nil, nil); err != nil {
			t.Fatalf("runAnalyzePriority: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("global --json should output JSON envelope: %v\n%s", err, stdout)
	}
	if parsed["command"] != "analyze.priority" || parsed["mode"] != "json" {
		t.Fatalf("unexpected analyze envelope: %#v", parsed)
	}
}

func TestPrintAnalyzeJSONReturnsMarshalError(t *testing.T) {
	stdout := captureStdout(t, func() {
		err := printAnalyzeJSON(map[string]interface{}{"bad": func() {}})
		if err == nil {
			t.Fatalf("expected marshal error")
		}
	})
	if stdout != "" {
		t.Fatalf("marshal errors should not write partial JSON, got %q", stdout)
	}
}

func TestAnalyzeHelpTextIsEnglish(t *testing.T) {
	for name, cmd := range map[string]interface {
		Help() error
		SetOut(io.Writer)
		SetErr(io.Writer)
	}{
		"analyze":  analyzeCmd,
		"quadrant": analyzeQuadrantCmd,
	} {
		output := helpOutput(t, cmd)
		for _, disallowed := range []string{"分析", "任务", "子命令", "示例", "按照", "紧急"} {
			if strings.Contains(output, disallowed) {
				t.Fatalf("%s help should be English, found %q in:\n%s", name, disallowed, output)
			}
		}
	}
}

func withAnalyzeTestConfig(t *testing.T, storagePath string) {
	t.Helper()
	previousCfg := cfg
	previousAnalyzeFormat := analyzeFormat
	t.Cleanup(func() {
		cfg = previousCfg
		analyzeFormat = previousAnalyzeFormat
	})

	cfg = config.DefaultConfig()
	cfg.Storage.Path = storagePath
	cfg.Storage.File.Format = "json"
	analyzeFormat = "text"
}

func seedAnalyzeTasks(t *testing.T, tasks []model.Task) {
	t.Helper()
	store, err := filestore.New(cfg.Storage.Path, cfg.Storage.File.Format)
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	for i := range tasks {
		if err := store.SaveTask(context.Background(), &tasks[i]); err != nil {
			t.Fatalf("SaveTask: %v", err)
		}
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
}

func analyzeTask(title string, priority model.Priority, quadrant model.Quadrant, score int, status model.TaskStatus, dueDate, completedAt *time.Time) model.Task {
	now := time.Now().UTC()
	return model.Task{
		ID:            title,
		Title:         title,
		Status:        status,
		CreatedAt:     now,
		UpdatedAt:     now,
		CompletedAt:   completedAt,
		DueDate:       dueDate,
		Quadrant:      quadrant,
		Priority:      priority,
		PriorityScore: score,
		Source:        model.SourceLocal,
		SourceRawID:   title,
	}
}

func TestRenderAnalyzeSubcommandsUseSharedPanels(t *testing.T) {
	quadrant := renderQuadrantAnalysis(QuadrantAnalysis{Q1: QuadrantData{Name: "Q1", Count: 1, Percentage: 100}})
	timeOut := renderTimeAnalysis(TimeAnalysis{Overdue: TimeData{Description: "Overdue", Count: 2}})
	trend := renderTrendAnalysis(TrendAnalysis{DailyCompletions: []DayData{{Date: "2026-06-10", Completed: 3}}, WeeklyAverage: 3, TotalCompleted: 3})
	report := renderAnalyzeReport(AnalyzeReport{Total: 1, Active: 1})

	for name, out := range map[string]string{"quadrant": quadrant, "time": timeOut, "trend": trend, "report": report} {
		if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
			t.Fatalf("%s output should use shared panel:\n%s", name, out)
		}
	}
}
