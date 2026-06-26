package taskoutput

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/model"
)

// TaskRenderOptions controls how task output is rendered.
type TaskRenderOptions struct {
	Format string
	Fields []string
	Writer io.Writer
}

// TaskProjection 是 internal/model.Task 的稳定 DTO 投影。
// 命令层和 renderer 只通过 TaskProjection 访问任务数据，不直接暴露 model.Task。
// 字段取舍：省略 Description、Metadata、CreatedAt、UpdatedAt、CompletedAt 等大/内部字段；
// 日期统一格式化为 "2006-01-02"；象限和优先级转为中文显示名。
type TaskProjection struct {
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
	Domain        string   `json:"domain"`
	ListID        string   `json:"list_id,omitempty"`
	ListName      string   `json:"list_name,omitempty"`
	Progress      int      `json:"progress,omitempty"`
	PriorityScore int      `json:"priority_score,omitempty"`
}

// TaskBrowsePage is the stable machine payload for task browsing commands.
type TaskBrowsePage struct {
	Tasks   interface{} `json:"tasks"`
	Total   int         `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
	HasMore bool        `json:"has_more"`
}

// TaskListSummary is the stable DTO for a provider task list row.
type TaskListSummary struct {
	Provider       string `json:"provider"`
	ListID         string `json:"list_id"`
	ListName       string `json:"list_name"`
	TaskCountLocal int    `json:"task_count_local"`
}

// TaskListsData is the stable machine payload for list browsing commands.
type TaskListsData struct {
	Lists []TaskListSummary `json:"lists"`
	Total int               `json:"total"`
}

// NewTaskBrowseProjection wraps task browse results in the shared CLI output projection.
func NewTaskBrowseProjection(command string, tasks []model.Task, fields []string, total, limit, offset int) clioutput.Projection {
	count := len(tasks)
	hasMore := offset+count < total
	p := clioutput.New(command)
	if total == 0 {
		p.Summary = "No tasks found."
		p.Actions = append(p.Actions, clioutput.Action{Name: "sync_now", Command: "taskbridge list --sync-now"})
	} else if count == total {
		p.Summary = fmt.Sprintf("Loaded %d tasks.", total)
	} else {
		p.Summary = fmt.Sprintf("Loaded %d of %d tasks.", count, total)
	}
	p.Facts["total"] = total
	p.Facts["count"] = count
	p.Facts["limit"] = limit
	p.Facts["offset"] = offset
	p.Facts["has_more"] = hasMore
	p.Data = TaskBrowsePage{
		Tasks:   TaskJSONRows(tasks, fields),
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}
	return p
}

// NewTaskListsProjection wraps task-list browse results in the shared CLI output projection.
func NewTaskListsProjection(command string, lists []TaskListSummary) clioutput.Projection {
	if lists == nil {
		lists = []TaskListSummary{}
	}
	p := clioutput.New(command)
	total := len(lists)
	if total == 0 {
		p.Summary = "No lists found."
		p.Actions = append(p.Actions, clioutput.Action{Name: "sync_now", Command: "taskbridge lists --sync-now"})
	} else {
		p.Summary = fmt.Sprintf("Loaded %d lists.", total)
	}
	p.Facts["total"] = total
	p.Data = TaskListsData{Lists: lists, Total: total}
	return p
}

// RenderTasks dispatches task rendering based on format.
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

// ToTaskProjection converts a model.Task to a TaskProjection DTO.
func ToTaskProjection(t model.Task) TaskProjection {
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

	return TaskProjection{
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
		Domain:        string(model.NormalizeTaskDomain(t.Domain)),
		ListID:        t.ListID,
		ListName:      t.ListName,
		Progress:      t.Progress,
		PriorityScore: t.PriorityScore,
	}
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
