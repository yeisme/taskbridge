package taskoutput

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/yeisme/taskbridge/internal/model"
)

// TaskJSONRows converts tasks to JSON-serializable rows.
// If fields is empty, returns full TaskProjection slice; otherwise returns
// a slice of maps with only the requested fields.
func TaskJSONRows(tasks []model.Task, fields []string) interface{} {
	if len(fields) == 0 {
		out := make([]TaskProjection, len(tasks))
		for i, t := range tasks {
			out[i] = ToTaskProjection(t)
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

// PrintTasksJSON writes tasks as JSON to w.
func PrintTasksJSON(w io.Writer, tasks []model.Task, fields []string) error {
	data, err := json.MarshalIndent(TaskJSONRows(tasks, fields), "", "  ")
	if err != nil {
		return fmt.Errorf("serialize tasks failed: %w", err)
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
