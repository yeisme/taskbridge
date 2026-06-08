package model

import "time"

func CloneTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	clone := *task
	clone.CompletedAt = cloneTimePtr(task.CompletedAt)
	clone.DueDate = cloneTimePtr(task.DueDate)
	clone.StartDate = cloneTimePtr(task.StartDate)
	clone.Reminder = cloneTimePtr(task.Reminder)
	clone.ParentID = cloneStringPtr(task.ParentID)
	clone.Tags = append([]string(nil), task.Tags...)
	clone.Categories = append([]string(nil), task.Categories...)
	clone.SubtaskIDs = append([]string(nil), task.SubtaskIDs...)
	clone.Metadata = CloneTaskMetadata(task.Metadata)
	return &clone
}

func CloneTaskList(list *TaskList) *TaskList {
	if list == nil {
		return nil
	}
	clone := *list
	return &clone
}

func CloneTaskMetadata(metadata *TaskMetadata) *TaskMetadata {
	if metadata == nil {
		return nil
	}
	clone := *metadata
	clone.CustomFields = cloneCustomFields(metadata.CustomFields)
	return &clone
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneCustomFields(fields map[string]interface{}) map[string]interface{} {
	if fields == nil {
		return nil
	}
	clone := make(map[string]interface{}, len(fields))
	for key, value := range fields {
		clone[key] = cloneCustomFieldValue(value)
	}
	return clone
}

// cloneCustomFieldValue copies the mutable container types normally produced
// by JSON metadata decoding so callers cannot mutate storage-owned maps/slices.
func cloneCustomFieldValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return cloneCustomFields(v)
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = cloneCustomFieldValue(item)
		}
		return out
	case []string:
		return append([]string(nil), v...)
	case *time.Time:
		return cloneTimePtr(v)
	default:
		return value
	}
}
