package clioutput

import (
	"strings"
	"testing"
)

func TestRenderStatPanelIncludesRowsAndFooter(t *testing.T) {
	out := RenderStatPanel(StatPanel{
		Title: "Prioritization analysis",
		Rows: []StatRow{
			{Icon: "🔴", Label: "Urgent (P0)", Value: "0", Percent: "0.0%"},
			{Icon: "⚪", Label: "No priority", Value: "2", Percent: "50.0%"},
		},
		Footer: "Total: 2 tasks | Active: 2 | Completed: 0",
	})

	for _, want := range []string{"Prioritization analysis", "Urgent (P0)", "No priority", "50.0%", "Total: 2 tasks | Active: 2 | Completed: 0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("RenderStatPanel missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Fatalf("RenderStatPanel should use a light boxed panel:\n%s", out)
	}
}

func TestRenderStatPanelAlignsValueAndPercentColumns(t *testing.T) {
	out := RenderStatPanel(StatPanel{
		Title: "Alignment",
		Rows: []StatRow{
			{Label: "Short", Value: "1", Percent: "5.0%", Hint: "small"},
			{Label: "Much longer label", Value: "20", Percent: "95.0%", Hint: "large"},
		},
	})
	lines := strings.Split(out, "\n")
	var valueEndCols []int
	var percentEndCols []int
	for _, line := range lines {
		if !strings.Contains(line, "Short") && !strings.Contains(line, "Much longer label") {
			continue
		}
		value := "1"
		percent := "5.0%"
		if strings.Contains(line, "Much longer label") {
			value = "20"
			percent = "95.0%"
		}
		valueIndex := strings.Index(line, value)
		valueEndCols = append(valueEndCols, DisplayWidth(line[:valueIndex])+DisplayWidth(value))
		percentIndex := strings.Index(line, percent)
		percentEndCols = append(percentEndCols, DisplayWidth(line[:percentIndex])+DisplayWidth(percent))
	}
	if len(valueEndCols) != 2 || valueEndCols[0] != valueEndCols[1] {
		t.Fatalf("value columns not aligned: cols=%v\n%s", valueEndCols, out)
	}
	if len(percentEndCols) != 2 || percentEndCols[0] != percentEndCols[1] {
		t.Fatalf("percent columns not aligned: cols=%v\n%s", percentEndCols, out)
	}
}

func TestRenderSummaryRendersFactsAsTableWithoutDuplicateHighlight(t *testing.T) {
	p := New("doctor.check")
	p.Summary = "TaskBridge is available."
	p.Facts["checks"] = 3
	out := RenderSummary(p)
	if !strings.Contains(out, "┌") || !strings.Contains(out, "Fact") || !strings.Contains(out, "Value") {
		t.Fatalf("RenderSummary should render facts as a table:\n%s", out)
	}
	if strings.Contains(out, "Highlights") && strings.Count(out, "TaskBridge is available.") > 1 {
		t.Fatalf("RenderSummary should not duplicate the status summary as a highlight:\n%s", out)
	}
}

func TestRenderSummaryUsesEnglishSectionsAndOneHint(t *testing.T) {
	p := New("doctor.check")
	p.Summary = "TaskBridge is available."
	p.Facts["checks"] = 3
	p.Risks = []string{"Provider credentials are missing"}
	p.Actions = []Action{{Name: "today", Command: "taskbridge today"}, {Name: "review", Command: "taskbridge review"}}

	out := RenderSummary(p)
	for _, want := range []string{"Status", "Facts", "Risks", "Recommended next step", "taskbridge today"} {
		if !strings.Contains(out, want) {
			t.Fatalf("RenderSummary missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "taskbridge review") {
		t.Fatalf("RenderSummary should render only one primary next step:\n%s", out)
	}
}

func TestRenderExplainUsesReviewableSections(t *testing.T) {
	p := New("review.check")
	p.Summary = "Need review."
	p.Evidence = []string{"3 overdue tasks"}
	p.Risks = []string{"schedule drift"}
	p.Actions = []Action{{Name: "review", Command: "taskbridge review"}}
	out := RenderExplain(p)
	for _, want := range []string{"Conclusion", "Evidence", "Risks", "Recommended next step", "taskbridge review"} {
		if !strings.Contains(out, want) {
			t.Fatalf("RenderExplain missing %q:\n%s", want, out)
		}
	}
}
