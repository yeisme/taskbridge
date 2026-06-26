package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/yeisme/taskbridge/internal/controlplane"
	"github.com/yeisme/taskbridge/pkg/ui"
)

func Today(result *controlplane.TodayResult) string {
	var sb strings.Builder
	fprintf(&sb, "%s\n\n", ui.Header("TaskBridge Today "+result.Date))

	for i, section := range result.Sections {
		if i > 0 {
			sb.WriteString("\n")
		}
		fprintf(&sb, "%s %s\n", ui.Bold(section.Title), ui.Dim(fmt.Sprintf("(%d)", len(section.Tasks))))
		if len(section.Tasks) == 0 {
			fprintf(&sb, "%s\n", ui.Dim("  (none)"))
			continue
		}
		for _, task := range section.Tasks {
			sb.WriteString(TaskLine(task, result.Date))
		}
	}

	if len(result.ProjectNext) > 0 {
		sb.WriteString("\n")
		fprintf(&sb, "%s\n", ui.Bold("Project next steps"))
		for _, item := range result.ProjectNext {
			fprintf(&sb, "  %s  %s\n", ui.Highlight(item.ProjectName), ui.Dim(item.NextTaskID))
		}
	}

	if len(result.SuggestedActions) > 0 {
		sb.WriteString("\n")
		fprintf(&sb, "%s\n\n", ui.Divider())
		sb.WriteString(ActionSection(result.SuggestedActions))
	}

	sb.WriteString("\n")
	fprintf(&sb, "%s\n", ui.Divider())
	parts := []string{}
	if v, ok := result.Summary["work"]; ok {
		parts = append(parts, fmt.Sprintf("work %d", v))
	}
	if v, ok := result.Summary["overdue"]; ok && v > 0 {
		parts = append(parts, fmt.Sprintf("overdue %d", v))
	}
	if v, ok := result.Summary["life"]; ok {
		parts = append(parts, fmt.Sprintf("life %d", v))
	}
	if v, ok := result.Summary["inbox"]; ok {
		parts = append(parts, fmt.Sprintf("inbox %d", v))
	}
	fprintf(&sb, "%s\n", ui.Dim("  "+strings.Join(parts, " · ")))
	return sb.String()
}

func TaskList(title string, result *controlplane.ListResult) string {
	var sb strings.Builder
	fprintf(&sb, "%s %s\n\n", ui.Bold(title), ui.Dim(fmt.Sprintf("(%d)", result.Count)))
	for _, task := range result.Tasks {
		sb.WriteString(TaskLine(task, ""))
	}
	return sb.String()
}

func Review(result *controlplane.ReviewResult) string {
	var sb strings.Builder
	fprintf(&sb, "%s\n\n", ui.Header("Task health review"))
	fprintf(&sb, "%s\n", ui.Bold("Summary"))
	for k, v := range result.Summary {
		fprintf(&sb, "  %s  %d\n", ui.Dim(k+":"), v)
	}
	if len(result.SuggestedActions) == 0 {
		fprintf(&sb, "\n%s\n", ui.Dim("No suggested actions."))
		return sb.String()
	}
	sb.WriteString("\n")
	sb.WriteString(ActionSection(result.SuggestedActions))
	return sb.String()
}

func TaskLine(task controlplane.TaskRef, dateRef string) string {
	var sb strings.Builder
	fprintf(&sb, "  %s %s\n", priorityBullet(task.Priority), task.Title)

	parts := []string{sourceBadge(task.Source), domainBadge(task.Domain)}
	if task.Priority > 0 {
		parts = append(parts, priorityLabel(task.Priority))
	}
	if task.Status == "in_progress" {
		parts = append(parts, ui.Warning("in progress"))
	}
	if task.DueDate != nil {
		parts = append(parts, dueDateTag(task.DueDate, dateRef))
	}
	if task.EstimatedMinutes > 0 {
		parts = append(parts, estimate(task.EstimatedMinutes))
	}
	if task.Reason != "" {
		parts = append(parts, ui.Dim(task.Reason))
	}
	fprintf(&sb, "    %s\n", strings.Join(parts, " · "))
	return sb.String()
}

func priorityBullet(priority int) string {
	switch {
	case priority >= 4:
		return ui.PriorityStyle(0).Render("●")
	case priority == 3:
		return ui.PriorityStyle(1).Render("●")
	case priority == 2:
		return ui.PriorityStyle(2).Render("●")
	case priority == 1:
		return ui.PriorityStyle(3).Render("○")
	default:
		return ui.PriorityStyle(3).Render("○")
	}
}

func priorityLabel(priority int) string {
	labels := map[int]string{0: "-", 1: "P3", 2: "P2", 3: "P1", 4: "P0"}
	styleIdx := map[int]int{0: 3, 1: 3, 2: 2, 3: 1, 4: 0}
	l, ok := labels[priority]
	if !ok {
		return "-"
	}
	si, ok := styleIdx[priority]
	if !ok {
		si = 3
	}
	return ui.PriorityStyle(si).Render(l)
}

func sourceBadge(source string) string {
	return ui.Dim(source)
}

func domainBadge(domain string) string {
	if domain == "" {
		domain = "unknown"
	}
	return ui.Dim(domain)
}

func dueDateTag(due *time.Time, dateRef string) string {
	if due == nil {
		return ""
	}

	now := time.Now()
	if dateRef != "" {
		if parsed, err := time.Parse("2006-01-02", dateRef); err == nil {
			now = parsed
		}
	}

	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if dueDay.Before(nowDay) {
		days := int(nowDay.Sub(dueDay).Hours() / 24)
		return ui.Error(fmt.Sprintf("overdue %d day(s)", days))
	}
	if dueDay.Equal(nowDay) {
		return ui.Dim("due " + due.Format("15:04"))
	}
	return ui.Dim("due " + due.Format("01-02"))
}

func estimate(minutes int) string {
	if minutes <= 0 {
		return ""
	}
	if minutes >= 60 && minutes%60 == 0 {
		return ui.Dim(fmt.Sprintf("%dh", minutes/60))
	}
	return ui.Dim(fmt.Sprintf("%d min", minutes))
}

func ActionSection(actions []controlplane.SuggestedAction) string {
	var sb strings.Builder
	fprintf(&sb, "%s\n", ui.Bold("Suggested actions"))
	for _, action := range actions {
		fprintf(&sb, "  %s %s: %s\n", actionType(action.Type), ui.Dim(action.TaskID), action.Reason)
	}
	return sb.String()
}

func actionType(actionType string) string {
	switch actionType {
	case "defer_task":
		return ui.Warning("defer_task")
	case "split_task":
		return ui.Highlight("split_task")
	case "archive_task":
		return ui.Dim("archive_task")
	default:
		return ui.Dim(actionType)
	}
}

func fprintf(sb *strings.Builder, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(sb, format, args...)
}
