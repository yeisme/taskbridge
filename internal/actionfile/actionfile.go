package actionfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage"
)

const Schema = "taskbridge.actions.v1"

type File struct {
	Schema    string   `json:"schema"`
	Source    string   `json:"source,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	Actions   []Action `json:"actions"`
}

type Action struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	TaskID               string `json:"task_id,omitempty"`
	ProjectID            string `json:"project_id,omitempty"`
	DueDate              string `json:"due_date,omitempty"`
	Reason               string `json:"reason,omitempty"`
	RequiresConfirmation bool   `json:"requires_confirmation,omitempty"`
}

type ExecuteOptions struct {
	DryRun  bool
	Confirm bool
}

type ExecuteResult struct {
	Schema               string   `json:"schema"`
	Status               string   `json:"status"`
	DryRun               bool     `json:"dry_run"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	Total                int      `json:"total"`
	Updated              int      `json:"updated"`
	Skipped              int      `json:"skipped"`
	Errors               []string `json:"errors"`
}

type Executor struct {
	TaskStore storage.Storage
}

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file.Schema != "" && file.Schema != Schema {
		return nil, fmt.Errorf("unsupported action schema: %s", file.Schema)
	}
	if file.Schema == "" {
		file.Schema = Schema
	}
	return &file, nil
}

func (e Executor) Execute(ctx context.Context, file *File, opts ExecuteOptions) ExecuteResult {
	result := ExecuteResult{Schema: "taskbridge.action-result.v1", Status: "ok", DryRun: opts.DryRun, Total: len(file.Actions), Errors: []string{}}
	if e.TaskStore == nil {
		result.Status = "error"
		result.Errors = append(result.Errors, "task store is nil")
		return result
	}
	for _, action := range file.Actions {
		if action.RequiresConfirmation && !opts.DryRun && !opts.Confirm {
			result.RequiresConfirmation = true
			result.Skipped++
			continue
		}
		if isDangerous(action.Type) && !opts.DryRun && !opts.Confirm {
			result.RequiresConfirmation = true
			result.Skipped++
			continue
		}
		if err := e.apply(ctx, action, opts); err != nil {
			result.Status = "error"
			result.Skipped++
			result.Errors = append(result.Errors, err.Error())
			continue
		}
		result.Updated++
	}
	return result
}

func (e Executor) apply(ctx context.Context, action Action, opts ExecuteOptions) error {
	if strings.TrimSpace(action.TaskID) == "" {
		return fmt.Errorf("action %s missing task_id", action.ID)
	}
	task, err := e.TaskStore.GetTask(ctx, action.TaskID)
	if err != nil {
		return fmt.Errorf("action %s get task failed: %w", action.ID, err)
	}
	task = cloneTask(task)
	switch action.Type {
	case "defer_task":
		task.Status = model.StatusDeferred
	case "reschedule_task":
		due, err := time.Parse("2006-01-02", strings.TrimSpace(action.DueDate))
		if err != nil {
			return fmt.Errorf("action %s invalid due_date: %s", action.ID, action.DueDate)
		}
		task.DueDate = &due
	case "complete_task":
		now := time.Now()
		task.Status = model.StatusCompleted
		task.CompletedAt = &now
	case "split_task":
		if task.Metadata == nil {
			task.Metadata = &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{}}
		}
		if task.Metadata.CustomFields == nil {
			task.Metadata.CustomFields = map[string]interface{}{}
		}
		task.Metadata.CustomFields["tb_split_suggested"] = true
		task.Metadata.CustomFields["tb_split_suggested_reason"] = action.Reason
	default:
		return fmt.Errorf("action %s unsupported type: %s", action.ID, action.Type)
	}
	task.UpdatedAt = time.Now()
	if opts.DryRun {
		return nil
	}
	return e.TaskStore.SaveTask(ctx, task)
}

func cloneTask(task *model.Task) *model.Task {
	if task == nil {
		return nil
	}
	cp := *task
	if task.CompletedAt != nil {
		v := *task.CompletedAt
		cp.CompletedAt = &v
	}
	if task.DueDate != nil {
		v := *task.DueDate
		cp.DueDate = &v
	}
	if task.Metadata != nil {
		meta := *task.Metadata
		if task.Metadata.CustomFields != nil {
			meta.CustomFields = map[string]interface{}{}
			for k, v := range task.Metadata.CustomFields {
				meta.CustomFields[k] = v
			}
		}
		cp.Metadata = &meta
	}
	return &cp
}

func isDangerous(actionType string) bool {
	switch actionType {
	case "complete_task", "delete_task", "defer_task", "reschedule_task":
		return true
	default:
		return false
	}
}
