package taskquery

import (
	"fmt"
	"strings"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
)

type ListOptions struct {
	Source        string
	Status        string
	StatusChanged bool
	Quadrant      int
	Priority      int
	Tag           string
	ListNames     []string
	ListIDs       []string
	TaskIDs       []string
	QueryText     string
	All           bool
}

type ListQuery struct {
	Source string
	Query  storage.Query
}

func BuildListQuery(opts ListOptions) (ListQuery, error) {
	resolvedSource := ""
	if strings.TrimSpace(opts.Source) != "" {
		resolvedSource = provider.ResolveProviderName(strings.TrimSpace(opts.Source))
		if !provider.IsValidProvider(resolvedSource) {
			return ListQuery{}, fmt.Errorf("不支持的来源: %s", opts.Source)
		}
	}

	query := storage.Query{
		ListIDs:   cleanStrings(opts.ListIDs),
		ListNames: cleanStrings(opts.ListNames),
		TaskIDs:   cleanStrings(opts.TaskIDs),
		QueryText: strings.TrimSpace(opts.QueryText),
	}
	if resolvedSource != "" {
		query.Sources = []model.TaskSource{model.TaskSource(resolvedSource)}
	}
	if opts.StatusChanged && strings.TrimSpace(opts.Status) != "" {
		for _, status := range splitCSV(opts.Status) {
			query.Statuses = append(query.Statuses, model.TaskStatus(status))
		}
	}
	if opts.Quadrant > 0 && opts.Quadrant <= 4 {
		query.Quadrants = []model.Quadrant{model.Quadrant(opts.Quadrant)}
	}
	if opts.Priority > 0 && opts.Priority <= 4 {
		query.Priorities = []model.Priority{model.Priority(opts.Priority)}
	}
	if strings.TrimSpace(opts.Tag) != "" {
		query.Tags = []string{strings.TrimSpace(opts.Tag)}
	}
	if !opts.StatusChanged && !opts.All {
		query.Statuses = []model.TaskStatus{model.StatusTodo, model.StatusInProgress}
	}

	return ListQuery{Source: resolvedSource, Query: query}, nil
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
