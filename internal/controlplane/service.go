package controlplane

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
)

type Service struct {
	TaskStore    storage.Storage
	ProjectStore project.Store
}

func (s *Service) Today(ctx context.Context, opts Options) (*TodayResult, error) {
	now := effectiveNow(opts.Now)
	tasks, err := s.queryActiveTasks(ctx, opts.Source)
	if err != nil {
		return nil, err
	}

	mustDo, overdue, atRisk, inbox := classifyTasks(tasks, now)
	next := pickNext(tasks, now, defaultLimit(opts.Limit, 5))
	projects, projectWarnings := s.projectNext(ctx, next)
	if projectWarnings == nil {
		projectWarnings = []string{}
	}

	sections := []Section{
		{ID: "must_do", Title: "今日必须做", Tasks: taskRefs(mustDo, "今天到期或已经逾期")},
		{ID: "at_risk", Title: "即将失控", Tasks: taskRefs(atRisk, "未来 3 天内到期")},
		{ID: "next", Title: "建议下一步", Tasks: taskRefs(next, "当前最值得推进")},
	}

	actions := make([]SuggestedAction, 0)
	for i, task := range overdue {
		if i >= 5 {
			break
		}
		actions = append(actions, SuggestedAction{
			ActionID:             fmt.Sprintf("act_overdue_%d", i+1),
			Type:                 "defer_task",
			TaskID:               task.ID,
			Reason:               "任务已逾期且仍未完成，建议重新决策日期或拆分",
			RequiresConfirmation: true,
		})
	}

	return &TodayResult{
		Schema: SchemaToday,
		Date:   now.Format("2006-01-02"),
		Status: "ok",
		Summary: map[string]int{
			"must_do":       len(mustDo),
			"overdue":       len(overdue),
			"at_risk":       len(atRisk),
			"inbox":         len(inbox),
			"project_next":  len(projects),
			"sync_warnings": 0,
		},
		Sections:         sections,
		ProjectNext:      projects,
		SuggestedActions: actions,
		Warnings:         projectWarnings,
	}, nil
}

func (s *Service) Next(ctx context.Context, opts Options) (*ListResult, error) {
	now := effectiveNow(opts.Now)
	tasks, err := s.queryActiveTasks(ctx, opts.Source)
	if err != nil {
		return nil, err
	}
	next := pickNext(tasks, now, defaultLimit(opts.Limit, 5))
	return &ListResult{Schema: SchemaNext, Status: "ok", Count: len(next), Tasks: taskRefs(next, "当前最值得推进")}, nil
}

func (s *Service) Inbox(ctx context.Context, opts Options) (*ListResult, error) {
	tasks, err := s.queryActiveTasks(ctx, opts.Source)
	if err != nil {
		return nil, err
	}
	_, _, _, inbox := classifyTasks(tasks, effectiveNow(opts.Now))
	limit := defaultLimit(opts.Limit, 50)
	if len(inbox) > limit {
		inbox = inbox[:limit]
	}
	return &ListResult{Schema: SchemaInbox, Status: "ok", Count: len(inbox), Tasks: taskRefs(inbox, "缺少日期或项目归属")}, nil
}

func (s *Service) Review(ctx context.Context, opts Options) (*ReviewResult, error) {
	now := effectiveNow(opts.Now)
	tasks, err := s.queryActiveTasks(ctx, opts.Source)
	if err != nil {
		return nil, err
	}
	_, overdue, _, inbox := classifyTasks(tasks, now)
	large := largeTasks(tasks)
	actions := make([]SuggestedAction, 0, len(overdue)+len(large))
	for i, task := range overdue {
		actions = append(actions, SuggestedAction{
			ActionID:             fmt.Sprintf("act_review_overdue_%d", i+1),
			Type:                 "defer_task",
			TaskID:               task.ID,
			Reason:               "任务已逾期，建议确认是否延期、拆分或取消",
			RequiresConfirmation: true,
		})
	}
	for i, task := range large {
		actions = append(actions, SuggestedAction{
			ActionID:             fmt.Sprintf("act_review_split_%d", i+1),
			Type:                 "split_task",
			TaskID:               task.ID,
			Reason:               "任务预估超过 180 分钟，建议拆成可执行步骤",
			RequiresConfirmation: true,
		})
	}
	return &ReviewResult{
		Schema: SchemaReview,
		Status: "ok",
		Summary: map[string]int{
			"active":  len(tasks),
			"overdue": len(overdue),
			"inbox":   len(inbox),
			"large":   len(large),
		},
		SuggestedActions: actions,
	}, nil
}

func (s *Service) queryActiveTasks(ctx context.Context, source string) ([]model.Task, error) {
	query := storage.Query{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress, model.StatusDeferred}}
	if strings.TrimSpace(source) != "" {
		resolved := provider.ResolveProviderName(source)
		if !provider.IsValidProvider(resolved) && resolved != string(model.SourceLocal) {
			return nil, fmt.Errorf("无效 provider: %s", source)
		}
		query.Sources = []model.TaskSource{model.TaskSource(resolved)}
	}
	return s.TaskStore.QueryTasks(ctx, query)
}

func (s *Service) projectNext(ctx context.Context, next []model.Task) ([]ProjectNextItem, []string) {
	if s.ProjectStore == nil {
		return nil, nil
	}
	projects, err := s.ProjectStore.ListProjects(ctx, "")
	if err != nil {
		return nil, []string{fmt.Sprintf("读取项目失败: %v", err)}
	}
	projectByID := map[string]project.Project{}
	for _, p := range projects {
		if p.Status == project.StatusConfirmed || p.Status == project.StatusSynced || p.Status == project.StatusActive {
			projectByID[p.ID] = p
		}
	}
	seen := map[string]bool{}
	items := make([]ProjectNextItem, 0)
	for _, task := range next {
		projectID := taskProjectID(task)
		if projectID == "" || seen[projectID] {
			continue
		}
		p, ok := projectByID[projectID]
		if !ok {
			continue
		}
		seen[projectID] = true
		items = append(items, ProjectNextItem{ProjectID: p.ID, ProjectName: p.Name, NextTaskID: task.ID, RiskLevel: "low"})
	}
	return items, nil
}

func classifyTasks(tasks []model.Task, now time.Time) (mustDo, overdue, atRisk, inbox []model.Task) {
	start := dayStart(now)
	endToday := start.AddDate(0, 0, 1)
	atRiskEnd := start.AddDate(0, 0, 4)
	for _, task := range tasks {
		if task.DueDate == nil {
			if taskProjectID(task) == "" {
				inbox = append(inbox, task)
			}
			continue
		}
		if task.DueDate.Before(start) {
			overdue = append(overdue, task)
			mustDo = append(mustDo, task)
			continue
		}
		if task.DueDate.Before(endToday) {
			mustDo = append(mustDo, task)
			continue
		}
		if task.DueDate.Before(atRiskEnd) {
			atRisk = append(atRisk, task)
		}
	}
	sortTasks(mustDo, now)
	sortTasks(overdue, now)
	sortTasks(atRisk, now)
	sortTasks(inbox, now)
	return
}

func pickNext(tasks []model.Task, now time.Time, limit int) []model.Task {
	candidates := append([]model.Task(nil), tasks...)
	sortTasks(candidates, now)
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func largeTasks(tasks []model.Task) []model.Task {
	items := make([]model.Task, 0)
	for _, task := range tasks {
		if task.EstimatedMinutes > 180 {
			items = append(items, task)
		}
	}
	return items
}

func sortTasks(tasks []model.Task, now time.Time) {
	sort.SliceStable(tasks, func(i, j int) bool {
		return scoreTask(tasks[i], now) > scoreTask(tasks[j], now)
	})
}

func scoreTask(task model.Task, now time.Time) int {
	score := int(task.Priority) * 20
	if task.DueDate != nil {
		days := int(task.DueDate.Sub(dayStart(now)).Hours() / 24)
		switch {
		case days < 0:
			score += 100
		case days == 0:
			score += 80
		case days <= 3:
			score += 50
		case days <= 7:
			score += 20
		}
	}
	if taskProjectID(task) != "" {
		score += 15
	}
	if task.EstimatedMinutes >= 30 && task.EstimatedMinutes <= 180 {
		score += 10
	}
	if task.EstimatedMinutes > 180 {
		score -= 20
	}
	return score
}

func taskRefs(tasks []model.Task, reason string) []TaskRef {
	refs := make([]TaskRef, 0, len(tasks))
	for _, task := range tasks {
		refs = append(refs, TaskRef{
			ID:               task.ID,
			Title:            task.Title,
			Status:           string(task.Status),
			Source:           string(task.Source),
			ListID:           task.ListID,
			ListName:         task.ListName,
			Priority:         int(task.Priority),
			Quadrant:         int(task.Quadrant),
			DueDate:          task.DueDate,
			EstimatedMinutes: task.EstimatedMinutes,
			ProjectID:        taskProjectID(task),
			Reason:           reason,
		})
	}
	return refs
}

func taskProjectID(task model.Task) string {
	if task.Metadata == nil || task.Metadata.CustomFields == nil {
		return ""
	}
	if value, ok := task.Metadata.CustomFields["tb_project_id"].(string); ok {
		return value
	}
	return ""
}

func effectiveNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now()
	}
	return now
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func defaultLimit(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func DemoTasks(now time.Time) []model.Task {
	if now.IsZero() {
		now = time.Now()
	}
	return []model.Task{
		{ID: "demo_today", Title: "完成 TaskBridge 今日工作台设计", Status: model.StatusTodo, Source: model.SourceLocal, Priority: model.PriorityHigh, DueDate: ptrTime(dayStart(now).Add(17 * time.Hour)), CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour), EstimatedMinutes: 90},
		{ID: "demo_overdue", Title: "整理逾期任务并决定去留", Status: model.StatusTodo, Source: model.SourceLocal, Priority: model.PriorityMedium, DueDate: ptrTime(dayStart(now).AddDate(0, 0, -2)), CreatedAt: now.AddDate(0, 0, -7), UpdatedAt: now.AddDate(0, 0, -2), EstimatedMinutes: 45},
		{ID: "demo_large", Title: "把 Agent 安全执行层拆成可交付任务", Status: model.StatusTodo, Source: model.SourceLocal, Priority: model.PriorityHigh, CreatedAt: now.AddDate(0, 0, -1), UpdatedAt: now.Add(-3 * time.Hour), EstimatedMinutes: 240},
		{ID: "demo_inbox", Title: "确认 Todoist 同步策略", Status: model.StatusTodo, Source: model.SourceLocal, Priority: model.PriorityLow, CreatedAt: now.AddDate(0, 0, -1), UpdatedAt: now.AddDate(0, 0, -1)},
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
