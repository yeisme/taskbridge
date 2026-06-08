package filestore

import (
	"sort"
	"strings"
	"time"

	"github.com/yeisme/taskbridge/internal/filter"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage"
)

func queryTasksFromMap(tasks map[string]*model.Task, query storage.Query) []model.Task {
	queryText := normalizedQueryText(query)
	result := make([]model.Task, 0, len(tasks))
	for _, task := range tasks {
		if taskMatchesQuery(task, query, queryText) {
			result = append(result, *model.CloneTask(task))
		}
	}
	if query.OrderBy != "" {
		sortTasks(result, query.OrderBy, query.OrderDesc)
	}
	return applyPagination(result, query.Offset, query.Limit)
}

func applyPagination(tasks []model.Task, offset, limit int) []model.Task {
	if offset <= 0 && limit <= 0 {
		return tasks
	}
	start := offset
	if start > len(tasks) {
		return []model.Task{}
	}
	end := len(tasks)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return tasks[start:end]
}

func normalizedQueryText(query storage.Query) string {
	if query.QueryText != "" {
		return query.QueryText
	}
	return query.FullText
}

func taskMatchesQuery(task *model.Task, query storage.Query, queryText string) bool {
	return matchTaskID(query.TaskIDs, task.ID) &&
		matchSource(query.Sources, task.Source) &&
		matchListID(query.ListIDs, task.ListID) &&
		matchListName(query.ListNames, task.ListName) &&
		matchStatus(query.Statuses, task.Status) &&
		matchQuadrant(query.Quadrants, task.Quadrant) &&
		matchPriority(query.Priorities, task.Priority) &&
		matchTags(query.Tags, task.Tags) &&
		matchDueDate(query.DueBefore, query.DueAfter, task.DueDate) &&
		matchQueryText(queryText, task)
}

func matchTaskID(ids []string, taskID string) bool {
	return len(ids) == 0 || containsStringCI(ids, taskID)
}

func matchSource(sources []model.TaskSource, source model.TaskSource) bool {
	if len(sources) == 0 {
		return true
	}
	for _, s := range sources {
		if source == s {
			return true
		}
	}
	return false
}

func matchListID(ids []string, listID string) bool {
	return len(ids) == 0 || containsStringCI(ids, listID)
}

func matchListName(listNames []string, listName string) bool {
	if len(listNames) == 0 {
		return true
	}
	for _, name := range listNames {
		if filter.MatchListNameExactNormalized(name, listName) {
			return true
		}
	}
	return false
}

func matchStatus(statuses []model.TaskStatus, status model.TaskStatus) bool {
	if len(statuses) == 0 {
		return true
	}
	for _, s := range statuses {
		if status == s {
			return true
		}
	}
	return false
}

func matchQuadrant(quadrants []model.Quadrant, quadrant model.Quadrant) bool {
	if len(quadrants) == 0 {
		return true
	}
	for _, q := range quadrants {
		if quadrant == q {
			return true
		}
	}
	return false
}

func matchPriority(priorities []model.Priority, priority model.Priority) bool {
	if len(priorities) == 0 {
		return true
	}
	for _, p := range priorities {
		if priority == p {
			return true
		}
	}
	return false
}

func matchTags(queryTags []string, taskTags []string) bool {
	if len(queryTags) == 0 {
		return true
	}
	for _, tag := range queryTags {
		found := false
		for _, taskTag := range taskTags {
			if strings.EqualFold(strings.TrimSpace(tag), strings.TrimSpace(taskTag)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func matchDueDate(dueBefore, dueAfter, dueDate *time.Time) bool {
	if dueBefore != nil && dueDate != nil && dueDate.After(*dueBefore) {
		return false
	}
	if dueAfter != nil && dueDate != nil && dueDate.Before(*dueAfter) {
		return false
	}
	return true
}

func matchQueryText(queryText string, task *model.Task) bool {
	if queryText == "" {
		return true
	}
	return filter.MatchQueryText(task, queryText)
}

func sortTasks(tasks []model.Task, orderBy string, orderDesc bool) {
	sort.Slice(tasks, func(i, j int) bool {
		cmp := compareTaskByOrder(tasks[i], tasks[j], orderBy)
		if orderDesc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareTaskByOrder(a, b model.Task, orderBy string) int {
	switch orderBy {
	case "due_date":
		return compareTimePointers(a.DueDate, b.DueDate)
	case "priority":
		return int(a.Priority) - int(b.Priority)
	case "created_at":
		return compareTimes(a.CreatedAt, b.CreatedAt)
	case "updated_at":
		return compareTimes(a.UpdatedAt, b.UpdatedAt)
	default:
		return compareTimes(a.UpdatedAt, b.UpdatedAt)
	}
}

func compareTimePointers(a, b *time.Time) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	default:
		return compareTimes(*a, *b)
	}
}

func compareTimes(a, b time.Time) int {
	if a.Before(b) {
		return -1
	}
	if a.After(b) {
		return 1
	}
	return 0
}

func containsStringCI(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), target) {
			return true
		}
	}
	return false
}
