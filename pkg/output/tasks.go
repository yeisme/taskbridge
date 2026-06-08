package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/pkg/ui"
)

type TaskRenderOptions struct {
	Format string
	Fields []string
	Writer io.Writer
}

type TaskOutput struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	ParentID      string   `json:"parent_id,omitempty"`
	SectionID     string   `json:"section_id,omitempty"`
	SectionName   string   `json:"section_name,omitempty"`
	SubtaskIDs    []string `json:"subtask_ids,omitempty"`
	Quadrant      string   `json:"quadrant,omitempty"`
	Priority      string   `json:"priority,omitempty"`
	DueDate       string   `json:"due_date,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Source        string   `json:"source"`
	ListID        string   `json:"list_id,omitempty"`
	ListName      string   `json:"list_name,omitempty"`
	Progress      int      `json:"progress,omitempty"`
	PriorityScore int      `json:"priority_score,omitempty"`
}

func RenderTasks(tasks []model.Task, opts TaskRenderOptions) error {
	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}
	switch opts.Format {
	case "json":
		return PrintTasksJSON(w, tasks, opts.Fields)
	case "markdown":
		PrintTasksMarkdown(w, tasks)
	case "compact":
		PrintTasksCompact(w, tasks, opts.Fields)
	case "tsv":
		PrintTasksTSV(w, tasks, opts.Fields)
	default:
		PrintTasksTable(w, tasks)
	}
	return nil
}

func ToTaskOutput(t model.Task) TaskOutput {
	quadrantNames := map[model.Quadrant]string{
		model.QuadrantUrgentImportant:       "Q1-紧急重要",
		model.QuadrantNotUrgentImportant:    "Q2-重要不紧急",
		model.QuadrantUrgentNotImportant:    "Q3-紧急不重要",
		model.QuadrantNotUrgentNotImportant: "Q4-不紧急不重要",
	}

	priorityNames := map[model.Priority]string{
		model.PriorityUrgent: "P0-紧急",
		model.PriorityHigh:   "P1-高",
		model.PriorityMedium: "P2-中",
		model.PriorityLow:    "P3-低",
		model.PriorityNone:   "无",
	}

	var dueDate string
	if t.DueDate != nil {
		dueDate = t.DueDate.Format("2006-01-02")
	}
	parentID := ""
	if t.ParentID != nil {
		parentID = *t.ParentID
	}

	return TaskOutput{
		ID:            t.ID,
		Title:         t.Title,
		Status:        string(t.Status),
		ParentID:      parentID,
		SectionID:     taskCustomFieldString(t, "todoist_section_id"),
		SectionName:   taskCustomFieldString(t, "todoist_section_name"),
		SubtaskIDs:    t.SubtaskIDs,
		Quadrant:      quadrantNames[t.Quadrant],
		Priority:      priorityNames[t.Priority],
		DueDate:       dueDate,
		Tags:          t.Tags,
		Source:        string(t.Source),
		ListID:        t.ListID,
		ListName:      t.ListName,
		Progress:      t.Progress,
		PriorityScore: t.PriorityScore,
	}
}

func TaskJSONRows(tasks []model.Task, fields []string) interface{} {
	if len(fields) == 0 {
		out := make([]TaskOutput, len(tasks))
		for i, t := range tasks {
			out[i] = ToTaskOutput(t)
		}
		return out
	}

	result := make([]map[string]interface{}, len(tasks))
	for i, t := range tasks {
		m := make(map[string]interface{}, len(fields))
		for _, f := range fields {
			m[f] = taskJSONFieldValue(t, f)
		}
		result[i] = m
	}
	return result
}

func taskCustomFieldString(t model.Task, key string) string {
	if t.Metadata == nil || t.Metadata.CustomFields == nil {
		return ""
	}
	raw, ok := t.Metadata.CustomFields[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func PrintTasksJSON(w io.Writer, tasks []model.Task, fields []string) error {
	data, err := json.MarshalIndent(TaskJSONRows(tasks, fields), "", "  ")
	if err != nil {
		return fmt.Errorf("序列化任务失败: %w", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

func taskJSONFieldValue(t model.Task, field string) interface{} {
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
		return nil
	case "source":
		return string(t.Source)
	case "list_name":
		return t.ListName
	case "tags":
		return t.Tags
	case "progress":
		return t.Progress
	case "description":
		return t.Description
	default:
		return nil
	}
}

func PrintTasksTable(w io.Writer, tasks []model.Task) {
	termWidth := detectTerminalWidth()
	idW := 5
	statusW := 8
	quadrantW := 14
	priorityW := 7
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
		ui.Column{Header: "标题", Width: titleW, AlignLeft: true},
		ui.Column{Header: "状态", Width: statusW, AlignLeft: true},
		ui.Column{Header: "象限", Width: quadrantW, AlignLeft: true},
		ui.Column{Header: "优先级", Width: priorityW, AlignLeft: true},
		ui.Column{Header: "截止日期", Width: dueW, AlignLeft: true},
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
	fmt.Fprintf(w, "共 %d 个任务\n", len(tasks))
}

func PrintTasksMarkdown(w io.Writer, tasks []model.Task) {
	fmt.Fprintln(w, "# 📋 任务列表")

	quadrants := map[model.Quadrant][]model.Task{}
	for _, t := range tasks {
		quadrants[t.Quadrant] = append(quadrants[t.Quadrant], t)
	}

	quadrantNames := map[model.Quadrant]string{
		model.QuadrantUrgentImportant:       "🔥 紧急且重要 (Q1)",
		model.QuadrantNotUrgentImportant:    "📋 重要不紧急 (Q2)",
		model.QuadrantUrgentNotImportant:    "⚡ 紧急不重要 (Q3)",
		model.QuadrantNotUrgentNotImportant: "🗑️ 不紧急不重要 (Q4)",
	}

	quadrantOrder := []model.Quadrant{
		model.QuadrantUrgentImportant,
		model.QuadrantNotUrgentImportant,
		model.QuadrantUrgentNotImportant,
		model.QuadrantNotUrgentNotImportant,
	}

	for _, q := range quadrantOrder {
		qtasks := quadrants[q]
		if len(qtasks) > 0 {
			fmt.Fprintf(w, "## %s\n\n", quadrantNames[q])
			for _, t := range qtasks {
				status := " "
				if t.Status == model.StatusCompleted {
					status = "x"
				}
				due := ""
				if t.DueDate != nil {
					due = fmt.Sprintf(" 📅 %s", t.DueDate.Format("2006-01-02"))
				}
				fmt.Fprintf(w, "- [%s] %s%s\n", status, t.Title, due)
			}
			fmt.Fprintln(w)
		}
	}

	fmt.Fprintln(w, "---")
	fmt.Fprintf(w, "**总计**: %d 个任务\n", len(tasks))
}

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

func quadrantShort(q model.Quadrant) string {
	switch q {
	case model.QuadrantUrgentImportant:
		return "Q1-紧急重要"
	case model.QuadrantNotUrgentImportant:
		return "Q2-重要不紧急"
	case model.QuadrantUrgentNotImportant:
		return "Q3-紧急不重要"
	case model.QuadrantNotUrgentNotImportant:
		return "Q4-不紧急不重要"
	default:
		return "-"
	}
}

func priorityShort(p model.Priority) string {
	switch p {
	case model.PriorityUrgent:
		return "P0-紧急"
	case model.PriorityHigh:
		return "P1-高"
	case model.PriorityMedium:
		return "P2-中"
	case model.PriorityLow:
		return "P3-低"
	default:
		return "-"
	}
}

func StatusShort(s model.TaskStatus) string {
	switch s {
	case model.StatusTodo:
		return "待办"
	case model.StatusInProgress:
		return "进行中"
	case model.StatusCompleted:
		return "已完成"
	case model.StatusCancelled:
		return "已取消"
	case model.StatusDeferred:
		return "已延期"
	default:
		return string(s)
	}
}

func detectTerminalWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
		return w
	}
	if v := os.Getenv("COLUMNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 40 {
			return n
		}
	}
	return 140
}

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

func clampInt(v, minValue, maxValue int) int {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}
