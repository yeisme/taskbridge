package taskoutput

import (
	"fmt"
	"io"
	"strings"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/pkg/ui"
)

// Compact and TSV format rendering for task output.

var compactFieldMaxWidths = map[string]int{
	"title":       40,
	"status":      12,
	"priority":    12,
	"quadrant":    12,
	"due_date":    12,
	"source":      12,
	"list_name":   12,
	"id":          12,
	"tags":        20,
	"progress":    8,
	"description": 40,
}

var defaultCompactFields = []string{"title", "status", "priority", "quadrant", "due_date", "source", "list_name"}

func compactStatus(s model.TaskStatus) string {
	switch s {
	case model.StatusTodo:
		return "todo"
	case model.StatusInProgress:
		return "in_progress"
	case model.StatusCompleted:
		return "completed"
	case model.StatusCancelled:
		return "cancelled"
	case model.StatusDeferred:
		return "deferred"
	default:
		return string(s)
	}
}

func compactPriority(p model.Priority) string {
	switch p {
	case model.PriorityUrgent:
		return "P0"
	case model.PriorityHigh:
		return "P1"
	case model.PriorityMedium:
		return "P2"
	case model.PriorityLow:
		return "P3"
	default:
		return "-"
	}
}

func compactQuadrant(q model.Quadrant) string {
	switch q {
	case model.QuadrantUrgentImportant:
		return "Q1"
	case model.QuadrantNotUrgentImportant:
		return "Q2"
	case model.QuadrantUrgentNotImportant:
		return "Q3"
	case model.QuadrantNotUrgentNotImportant:
		return "Q4"
	default:
		return "-"
	}
}

func compactFieldValue(t model.Task, field string) string {
	switch field {
	case "id":
		return t.ID
	case "title":
		return t.Title
	case "status":
		return compactStatus(t.Status)
	case "priority":
		return compactPriority(t.Priority)
	case "quadrant":
		return compactQuadrant(t.Quadrant)
	case "due_date":
		if t.DueDate != nil {
			return t.DueDate.Format("01-02")
		}
		return "-"
	case "source":
		return string(t.Source)
	case "list_name":
		if t.ListName != "" {
			return t.ListName
		}
		return "-"
	case "tags":
		if len(t.Tags) > 0 {
			return strings.Join(t.Tags, ",")
		}
		return "-"
	case "progress":
		return fmt.Sprintf("%d%%", t.Progress)
	case "description":
		if t.Description != "" {
			return t.Description
		}
		return "-"
	default:
		return "-"
	}
}

// PrintTasksCompact renders tasks in compact pipe-delimited format.
func PrintTasksCompact(w io.Writer, tasks []model.Task, fields []string) {
	if len(fields) == 0 {
		fields = defaultCompactFields
	}

	headers := make([]string, len(fields))
	for i, f := range fields {
		headers[i] = truncateCompact(f, compactFieldMaxWidths[f])
	}
	fmt.Fprintln(w, strings.Join(headers, "|"))

	for _, t := range tasks {
		row := make([]string, len(fields))
		for i, f := range fields {
			row[i] = truncateCompact(compactFieldValue(t, f), compactFieldMaxWidths[f])
		}
		fmt.Fprintln(w, strings.Join(row, "|"))
	}
}

func truncateCompact(s string, maxWidth int) string {
	if maxWidth <= 0 || ui.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}
	target := maxWidth - 3
	var b strings.Builder
	cur := 0
	for _, r := range s {
		rw := ui.StringWidth(string(r))
		if cur+rw > target {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	b.WriteString("...")
	return b.String()
}

var tsvDefaultFields = []string{"id", "title", "status", "priority", "quadrant", "due_date", "source", "list_name"}

func escapeTSV(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

func tsvFieldValue(t model.Task, field string) string {
	switch field {
	case "id":
		return t.ID
	case "title":
		return t.Title
	case "status":
		return string(t.Status)
	case "priority":
		return compactPriority(t.Priority)
	case "quadrant":
		return compactQuadrant(t.Quadrant)
	case "due_date":
		if t.DueDate != nil {
			return t.DueDate.Format("2006-01-02")
		}
		return ""
	case "source":
		return string(t.Source)
	case "list_name":
		return t.ListName
	case "tags":
		if len(t.Tags) > 0 {
			return strings.Join(t.Tags, ",")
		}
		return ""
	case "progress":
		return fmt.Sprintf("%d", t.Progress)
	case "description":
		return t.Description
	default:
		return ""
	}
}

// PrintTasksTSV renders tasks as tab-separated values.
func PrintTasksTSV(w io.Writer, tasks []model.Task, fields []string) {
	if len(fields) == 0 {
		fields = tsvDefaultFields
	}
	fmt.Fprintln(w, strings.Join(fields, "\t"))
	for _, t := range tasks {
		row := make([]string, len(fields))
		for i, f := range fields {
			row[i] = escapeTSV(tsvFieldValue(t, f))
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
}
