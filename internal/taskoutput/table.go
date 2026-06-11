package taskoutput

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/pkg/ui"
)

// PrintTasksTable renders tasks in a terminal table.
func PrintTasksTable(w io.Writer, tasks []model.Task) {
	termWidth := detectTerminalWidth()
	idW := 5
	statusW := 14
	quadrantW := 16
	priorityW := 12
	dueW := 10
	providerW := 10
	gapW := 2
	colCount := 8

	minTitleW := 28
	minListW := 14
	maxTitleW := 160
	maxListW := 60

	fixedW := idW + statusW + quadrantW + priorityW + dueW + providerW + (colCount-1)*gapW
	flexibleW := termWidth - fixedW
	if flexibleW < minTitleW+minListW {
		flexibleW = minTitleW + minListW
	}

	titleW := clampInt((flexibleW*2)/3, minTitleW, maxTitleW)
	listW := flexibleW - titleW
	if listW < minListW {
		deficit := minListW - listW
		titleW -= deficit
	}
	if titleW < minTitleW {
		titleW = minTitleW
	}
	listW = flexibleW - titleW
	if listW > maxListW {
		extra := listW - maxListW
		listW = maxListW
		titleW += extra
	}

	table := ui.NewSimpleTable(
		ui.Column{Header: "ID", Width: idW, AlignLeft: true},
		ui.Column{Header: "Title", Width: titleW, AlignLeft: true},
		ui.Column{Header: "Status", Width: statusW, AlignLeft: true},
		ui.Column{Header: "Quadrant", Width: quadrantW, AlignLeft: true},
		ui.Column{Header: "Priority", Width: priorityW, AlignLeft: true},
		ui.Column{Header: "Due", Width: dueW, AlignLeft: true},
		ui.Column{Header: "Provider", Width: providerW, AlignLeft: true},
		ui.Column{Header: "List", Width: listW, AlignLeft: true},
	)

	fmt.Fprintln(w)
	for _, t := range tasks {
		dueDate := "-"
		if t.DueDate != nil {
			dueDate = t.DueDate.Format("01-02")
			if t.DueDate.Before(time.Now()) && t.Status != model.StatusCompleted {
				dueDate = "!" + dueDate
			}
		}

		title := TruncateDisplay(t.Title, titleW)
		if t.Status == model.StatusCompleted {
			title = "✓ " + title
		}

		listName := "-"
		if t.ListName != "" {
			listName = TruncateDisplay(t.ListName, listW)
		}

		table.AddRow(
			TruncateDisplay(t.ID, idW),
			title,
			TruncateDisplay(StatusShort(t.Status), statusW),
			TruncateDisplay(quadrantShort(t.Quadrant), quadrantW),
			TruncateDisplay(priorityShort(t.Priority), priorityW),
			TruncateDisplay(dueDate, dueW),
			TruncateDisplay(string(t.Source), providerW),
			listName,
		)
	}

	fmt.Fprintln(w, table.Render())
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Total %d tasks\n", len(tasks))
}

// quadrantShort returns a compact English label for the quadrant.
func quadrantShort(q model.Quadrant) string {
	switch q {
	case model.QuadrantUrgentImportant:
		return "Q1 Urgent"
	case model.QuadrantNotUrgentImportant:
		return "Q2 Important"
	case model.QuadrantUrgentNotImportant:
		return "Q3 Delegate"
	case model.QuadrantNotUrgentNotImportant:
		return "Q4 Defer"
	default:
		return "-"
	}
}

// priorityShort returns a compact English label for the priority.
func priorityShort(p model.Priority) string {
	switch p {
	case model.PriorityUrgent:
		return "P0 Urgent"
	case model.PriorityHigh:
		return "P1 High"
	case model.PriorityMedium:
		return "P2 Medium"
	case model.PriorityLow:
		return "P3 Low"
	default:
		return "-"
	}
}

// StatusShort returns a compact English label for the status.
func StatusShort(s model.TaskStatus) string {
	switch s {
	case model.StatusTodo:
		return "Todo"
	case model.StatusInProgress:
		return "In progress"
	case model.StatusCompleted:
		return "Completed"
	case model.StatusCancelled:
		return "Cancelled"
	case model.StatusDeferred:
		return "Deferred"
	default:
		return string(s)
	}
}

// TruncateDisplay truncates a string for display, respecting CJK width.
func TruncateDisplay(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if ui.StringWidth(s) <= maxWidth {
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
