package sync

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	msprovider "github.com/yeisme/taskbridge/internal/provider/microsoft"
	"github.com/yeisme/taskbridge/internal/storage"
)

var markdownCheckboxPrefix = regexp.MustCompile(`^\s*\[\s*[xX ]?\s*\]\s*`)

type ProjectSyncResult struct {
	ProjectID string   `json:"project_id"`
	Provider  string   `json:"provider"`
	Status    string   `json:"status"`
	Pushed    int      `json:"pushed"`
	Updated   int      `json:"updated"`
	Errors    []string `json:"errors,omitempty"`
	Message   string   `json:"message,omitempty"`
}

func FindDefaultTaskListID(taskLists []model.TaskList) string {
	for _, list := range taskLists {
		if list.Name == "我的任务" || list.Name == "My Tasks" || list.ID == "@default" {
			return list.ID
		}
	}
	if len(taskLists) > 0 {
		return taskLists[0].ID
	}
	return ""
}

func PushProjectLocalTasks(ctx context.Context, taskStore storage.Storage, p provider.Provider, tasks []model.Task, defaultListID string, source model.TaskSource, dryRun bool, result *ProjectSyncResult) {
	planTaskToLocalID := buildPlanTaskToLocalIDMap(tasks)
	ordered := orderTasksByParent(tasks, planTaskToLocalID)
	remoteByLocalID := make(map[string]string, len(ordered))

	for _, task := range ordered {
		if task.Source != "" && task.Source != source && task.Source != model.SourceLocal {
			continue
		}
		listID := task.ListID
		if listID == "" {
			listID = defaultListID
		}

		parentRemoteID := ""
		parentLocalID := localParentIDFromTask(task, planTaskToLocalID)
		if parentLocalID != "" {
			parentRemoteID = remoteByLocalID[parentLocalID]
			if parentRemoteID == "" {
				parentRemoteID = findRemoteIDByLocalParent(ctx, taskStore, tasks, parentLocalID, source)
			}
			if parentRemoteID == "" && source == model.SourceGoogle {
				parentRemoteID = toGoogleParentRawID(parentLocalID, listID)
			}
			if parentRemoteID == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("skip child %s: parent remote id not resolved", task.ID))
				continue
			}
		}

		if source == model.SourceMicrosoft && parentRemoteID != "" {
			if err := syncMicrosoftChecklistStep(ctx, taskStore, p, listID, parentRemoteID, &task, dryRun, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("step %s: %v", task.ID, err))
			}
			continue
		}

		taskToSync := sanitizeTaskForRemote(task)
		if source == model.SourceGoogle && parentRemoteID != "" {
			taskToSync.ParentID = &parentRemoteID
		}
		if taskToSync.SourceRawID != "" {
			existingTask, err := p.GetTask(ctx, listID, taskToSync.SourceRawID)
			if err == nil && existingTask != nil {
				if existingTask.UpdatedAt.Before(taskToSync.UpdatedAt) {
					if !dryRun {
						if _, err := p.UpdateTask(ctx, listID, &taskToSync); err != nil {
							result.Errors = append(result.Errors, fmt.Sprintf("update %s: %v", taskToSync.ID, err))
							continue
						}
					}
					result.Updated++
				}
				remoteByLocalID[task.ID] = taskToSync.SourceRawID
				continue
			}
		}

		if !dryRun {
			createdTask, err := p.CreateTask(ctx, listID, &taskToSync)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create %s: %v", taskToSync.ID, err))
				continue
			}
			task.SourceRawID = createdTask.SourceRawID
			task.ListID = listID
			task.Source = source
			_ = taskStore.SaveTask(ctx, &task)
			remoteByLocalID[task.ID] = task.SourceRawID
		} else if task.SourceRawID != "" {
			remoteByLocalID[task.ID] = task.SourceRawID
		}
		result.Pushed++
	}
}

func PullProviderTasksIntoLocal(ctx context.Context, taskStore storage.Storage, p provider.Provider, providerName string) error {
	taskLists, err := p.ListTaskLists(ctx)
	if err != nil {
		return err
	}
	for _, list := range taskLists {
		_ = taskStore.SaveTaskList(ctx, &list)
		tasks, err := p.ListTasks(ctx, list.ID, provider.ListOptions{})
		if err != nil {
			continue
		}
		for _, task := range tasks {
			if task.ListID == "" {
				task.ListID = list.ID
			}
			if task.ListName == "" {
				task.ListName = list.Name
			}
			if task.Source == "" {
				task.Source = model.TaskSource(providerName)
			}
			_ = taskStore.SaveTask(ctx, &task)
		}
	}
	return nil
}

func syncMicrosoftChecklistStep(ctx context.Context, taskStore storage.Storage, p provider.Provider, listID, parentRemoteID string, task *model.Task, dryRun bool, result *ProjectSyncResult) error {
	msProvider, ok := p.(*msprovider.Provider)
	if !ok {
		return fmt.Errorf("provider is not microsoft")
	}
	stepID := parseMicrosoftStepID(task.SourceRawID)
	if task.Metadata != nil && task.Metadata.CustomFields != nil {
		if v, ok := task.Metadata.CustomFields["tb_ms_step_id"]; ok {
			stepID = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	isChecked := task.Status == model.StatusCompleted
	cleanTitle := sanitizeMarkdownText(task.Title)
	if dryRun {
		if stepID == "" {
			result.Pushed++
		} else {
			result.Updated++
		}
		return nil
	}
	if stepID == "" {
		item, err := msProvider.CreateChecklistItem(ctx, listID, parentRemoteID, cleanTitle, isChecked)
		if err != nil {
			return err
		}
		originalID := task.ID
		canonicalID := buildMicrosoftStepLocalID(parentRemoteID, item.ID)
		ensureTaskMetadata(task)
		task.Metadata.CustomFields["tb_ms_step_id"] = item.ID
		task.Metadata.CustomFields["tb_ms_parent_source_raw_id"] = parentRemoteID
		task.Metadata.LocalID = canonicalID
		task.ID = canonicalID
		task.SourceRawID = "ms_step:" + item.ID
		task.Source = model.SourceMicrosoft
		task.ListID = listID
		_ = taskStore.SaveTask(ctx, task)
		if originalID != "" && originalID != canonicalID {
			_ = taskStore.DeleteTask(ctx, originalID)
		}
		result.Pushed++
		return nil
	}

	if _, err := msProvider.UpdateChecklistItem(ctx, listID, parentRemoteID, stepID, cleanTitle, isChecked); err != nil {
		return err
	}
	canonicalID := buildMicrosoftStepLocalID(parentRemoteID, stepID)
	ensureTaskMetadata(task)
	task.Metadata.CustomFields["tb_ms_step_id"] = stepID
	task.Metadata.CustomFields["tb_ms_parent_source_raw_id"] = parentRemoteID
	task.Metadata.LocalID = canonicalID
	task.ID = canonicalID
	task.SourceRawID = "ms_step:" + stepID
	task.Source = model.SourceMicrosoft
	task.ListID = listID
	_ = taskStore.SaveTask(ctx, task)
	result.Updated++
	return nil
}

func ensureTaskMetadata(task *model.Task) {
	if task.Metadata == nil {
		task.Metadata = &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{}}
	}
	if task.Metadata.CustomFields == nil {
		task.Metadata.CustomFields = map[string]interface{}{}
	}
}

func orderTasksByParent(tasks []model.Task, planTaskToLocalID map[string]string) []model.Task {
	byID := make(map[string]model.Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}
	visited := make(map[string]bool, len(tasks))
	inStack := make(map[string]bool, len(tasks))
	ordered := make([]model.Task, 0, len(tasks))
	var visit func(string)
	visit = func(id string) {
		if visited[id] || inStack[id] {
			return
		}
		task, ok := byID[id]
		if !ok {
			return
		}
		inStack[id] = true
		if parentLocalID := localParentIDFromTask(task, planTaskToLocalID); parentLocalID != "" {
			visit(parentLocalID)
		}
		inStack[id] = false
		visited[id] = true
		ordered = append(ordered, task)
	}
	for _, task := range tasks {
		visit(task.ID)
	}
	return ordered
}

func findRemoteIDByLocalParent(ctx context.Context, taskStore storage.Storage, tasks []model.Task, parentLocalID string, source model.TaskSource) string {
	for _, task := range tasks {
		if task.ID == parentLocalID && task.SourceRawID != "" && (task.Source == source || task.Source == model.SourceLocal || task.Source == "") {
			return task.SourceRawID
		}
	}
	if parentTask, err := taskStore.GetTask(ctx, parentLocalID); err == nil && parentTask != nil {
		if parentTask.SourceRawID != "" && (parentTask.Source == source || parentTask.Source == model.SourceLocal || parentTask.Source == "") {
			return parentTask.SourceRawID
		}
	}
	all, err := taskStore.ListTasks(ctx, storage.ListOptions{})
	if err != nil {
		return ""
	}
	for _, task := range all {
		if getCustomFieldString(task, "tb_plan_task_id") != parentLocalID {
			continue
		}
		if task.SourceRawID != "" && (task.Source == source || task.Source == model.SourceLocal || task.Source == "") {
			return task.SourceRawID
		}
	}
	return ""
}

func buildPlanTaskToLocalIDMap(tasks []model.Task) map[string]string {
	result := make(map[string]string, len(tasks))
	for _, task := range tasks {
		planTaskID := getCustomFieldString(task, "tb_plan_task_id")
		if planTaskID != "" {
			result[planTaskID] = task.ID
		}
	}
	return result
}

func localParentIDFromTask(task model.Task, planTaskToLocalID map[string]string) string {
	if task.ParentID != nil {
		if parentID := strings.TrimSpace(*task.ParentID); parentID != "" {
			return parentID
		}
	}
	parentPlanTaskID := getCustomFieldString(task, "tb_parent_plan_task_id")
	if parentPlanTaskID == "" {
		return ""
	}
	return strings.TrimSpace(planTaskToLocalID[parentPlanTaskID])
}

func getCustomFieldString(task model.Task, key string) string {
	if task.Metadata == nil || task.Metadata.CustomFields == nil {
		return ""
	}
	raw, ok := task.Metadata.CustomFields[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func sanitizeTaskForRemote(task model.Task) model.Task {
	task.Title = sanitizeMarkdownText(task.Title)
	task.Description = sanitizeMarkdownText(task.Description)
	return task
}

func sanitizeMarkdownText(text string) string {
	out := strings.TrimSpace(text)
	if out == "" {
		return out
	}
	out = markdownCheckboxPrefix.ReplaceAllString(out, "")
	out = strings.ReplaceAll(out, "**", "")
	out = strings.ReplaceAll(out, "__", "")
	out = strings.Join(strings.Fields(out), " ")
	return out
}

func parseMicrosoftStepID(sourceRawID string) string {
	if !strings.HasPrefix(strings.TrimSpace(sourceRawID), "ms_step:") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(sourceRawID), "ms_step:"))
}

func buildMicrosoftStepLocalID(parentRemoteID, stepID string) string {
	parent := strings.TrimSpace(parentRemoteID)
	step := strings.TrimSpace(stepID)
	if parent == "" || step == "" {
		return ""
	}
	return fmt.Sprintf("ms-step-%s-%s", parent, step)
}

func toGoogleParentRawID(parentID, listID string) string {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" || listID == "" {
		return ""
	}
	prefix := "google-" + listID + "-"
	if strings.HasPrefix(parentID, prefix) {
		return strings.TrimPrefix(parentID, prefix)
	}
	return ""
}

func SplitTasksByParentRelation(tasks []model.Task) ([]model.Task, []model.Task) {
	planTaskToLocalID := buildPlanTaskToLocalIDMap(tasks)
	parents := make([]model.Task, 0, len(tasks))
	children := make([]model.Task, 0, len(tasks))
	for _, task := range tasks {
		if localParentIDFromTask(task, planTaskToLocalID) == "" {
			parents = append(parents, task)
		} else {
			children = append(children, task)
		}
	}
	return parents, children
}
