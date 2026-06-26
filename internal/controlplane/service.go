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

	_, overdue, _, inbox := classifyTasks(tasks, now)
	next := pickNext(tasks, now, defaultLimit(opts.Limit, 5))
	projects, projectWarnings := s.projectNext(ctx, next)
	if projectWarnings == nil {
		projectWarnings = []string{}
	}
	work := tasksForDomain(tasks, model.DomainWork, now)
	life := tasksForDomains(tasks, []model.TaskDomain{model.DomainLife, model.DomainPersonal}, now)

	sections := []Section{
		{ID: "work", Title: "Work", Tasks: taskRefs(work, now, "Work-domain tasks due soon or ready to move")},
		{ID: "life", Title: "Life", Tasks: taskRefs(life, now, "Life and personal tasks due soon or ready to move")},
		{ID: "inbox", Title: "Inbox", Tasks: taskRefs(inbox, now, "Missing domain, due date, or project assignment")},
		{ID: "overdue", Title: "Overdue", Tasks: taskRefs(overdue, now, "Past due and still open")},
		{ID: "recommended_next", Title: "Recommended next", Tasks: taskRefs(next, now, "Best cross-provider task to move forward now")},
		{ID: "sync_warnings", Title: "Sync warnings", Tasks: []TaskRef{}},
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
			Reason:               "Task is overdue and still open; choose a new date or split it",
			RequiresConfirmation: true,
		})
	}

	return &TodayResult{
		Schema: SchemaToday,
		Date:   now.Format("2006-01-02"),
		Status: "ok",
		Summary: map[string]int{
			"work":          len(work),
			"life":          len(life),
			"overdue":       len(overdue),
			"inbox":         len(inbox),
			"recommended":   len(next),
			"project_next":  len(projects),
			"sync_warnings": len(projectWarnings),
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
	return &ListResult{Schema: SchemaNext, Status: "ok", Count: len(next), Tasks: taskRefs(next, now, "Best cross-provider task to move forward now")}, nil
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
	return &ListResult{Schema: SchemaInbox, Status: "ok", Count: len(inbox), Tasks: taskRefs(inbox, effectiveNow(opts.Now), "Missing domain, due date, or project assignment")}, nil
}

func (s *Service) Review(ctx context.Context, opts Options) (*ReviewResult, error) {
	now := effectiveNow(opts.Now)
	tasks, err := s.queryActiveTasks(ctx, opts.Source)
	if err != nil {
		return nil, err
	}
	_, overdue, _, inbox := classifyTasks(tasks, now)
	large := largeTasks(tasks)
	coverage := domainCoverage(tasks)
	providerHealth := providerHealthSummary(tasks)
	actions := make([]SuggestedAction, 0, len(overdue)+len(large))
	for i, task := range overdue {
		actions = append(actions, SuggestedAction{
			ActionID:             fmt.Sprintf("act_review_overdue_%d", i+1),
			Type:                 "defer_task",
			TaskID:               task.ID,
			Reason:               "Task is overdue; decide whether to defer, split, or cancel it",
			RequiresConfirmation: true,
		})
	}
	for i, task := range large {
		actions = append(actions, SuggestedAction{
			ActionID:             fmt.Sprintf("act_review_split_%d", i+1),
			Type:                 "split_task",
			TaskID:               task.ID,
			Reason:               "Task estimate exceeds 180 minutes; split it into executable steps",
			RequiresConfirmation: true,
		})
	}
	return &ReviewResult{
		Schema: SchemaReview,
		Status: "ok",
		Summary: map[string]int{
			"active":                 len(tasks),
			"work":                   coverage[string(model.DomainWork)],
			"life":                   coverage[string(model.DomainLife)] + coverage[string(model.DomainPersonal)],
			"personal":               coverage[string(model.DomainPersonal)],
			"unknown_domain":         coverage[string(model.DomainUnknown)],
			"overdue_backlog":        len(overdue),
			"inbox":                  len(inbox),
			"large":                  len(large),
			"providers":              len(providerHealth),
			"providers_with_risk":    providerHealthRiskCount(providerHealth),
			"suggested_action_count": len(actions),
		},
		SuggestedActions: actions,
	}, nil
}

func (s *Service) queryActiveTasks(ctx context.Context, source string) ([]model.Task, error) {
	query := storage.Query{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress, model.StatusDeferred}}
	if strings.TrimSpace(source) != "" {
		resolved := provider.ResolveProviderName(source)
		if !provider.IsValidProvider(resolved) && resolved != string(model.SourceLocal) {
			return nil, fmt.Errorf("invalid provider: %s", source)
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
			if taskProjectID(task) == "" || classifyDomain(task) == model.DomainUnknown {
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

func tasksForDomain(tasks []model.Task, domain model.TaskDomain, now time.Time) []model.Task {
	return tasksForDomains(tasks, []model.TaskDomain{domain}, now)
}

func tasksForDomains(tasks []model.Task, domains []model.TaskDomain, now time.Time) []model.Task {
	wanted := make(map[model.TaskDomain]bool, len(domains))
	for _, domain := range domains {
		wanted[domain] = true
	}
	items := make([]model.Task, 0)
	for _, task := range tasks {
		if wanted[classifyDomain(task)] {
			items = append(items, task)
		}
	}
	sortTasks(items, now)
	if len(items) > 5 {
		items = items[:5]
	}
	return items
}

func domainCoverage(tasks []model.Task) map[string]int {
	coverage := map[string]int{
		string(model.DomainWork):     0,
		string(model.DomainLife):     0,
		string(model.DomainPersonal): 0,
		string(model.DomainUnknown):  0,
	}
	for _, task := range tasks {
		coverage[string(classifyDomain(task))]++
	}
	return coverage
}

type providerHealth struct {
	Tasks int
	Risks int
}

func providerHealthSummary(tasks []model.Task) map[string]providerHealth {
	providers := make(map[string]providerHealth)
	for _, task := range tasks {
		name := string(task.Source)
		if name == "" {
			name = string(model.SourceLocal)
		}
		health := providers[name]
		health.Tasks++
		if hasSyncRisk(task) {
			health.Risks++
		}
		providers[name] = health
	}
	return providers
}

func providerHealthRiskCount(providers map[string]providerHealth) int {
	count := 0
	for _, health := range providers {
		if health.Risks > 0 {
			count++
		}
	}
	return count
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
	if classifyDomain(task) == model.DomainUnknown {
		score -= 5
	}
	if hasSyncRisk(task) {
		score -= 60
	}
	return score
}

func taskRefs(tasks []model.Task, now time.Time, fallbackReason string) []TaskRef {
	refs := make([]TaskRef, 0, len(tasks))
	for _, task := range tasks {
		domain := classifyDomain(task)
		syncState := syncState(task)
		refs = append(refs, TaskRef{
			ID:               task.ID,
			Title:            task.Title,
			Status:           string(task.Status),
			Source:           string(task.Source),
			Domain:           string(domain),
			ListID:           task.ListID,
			ListName:         task.ListName,
			Priority:         int(task.Priority),
			Quadrant:         int(task.Quadrant),
			DueDate:          task.DueDate,
			EstimatedMinutes: task.EstimatedMinutes,
			ProjectID:        taskProjectID(task),
			SyncState:        syncState,
			NextAction:       recommendedAction(task),
			Reason:           recommendationReason(task, now, fallbackReason),
		})
	}
	return refs
}

func recommendedAction(task model.Task) string {
	if hasSyncRisk(task) {
		return "review_sync_state"
	}
	if task.EstimatedMinutes > 180 {
		return "split_or_review"
	}
	return "work_next"
}

func recommendationReason(task model.Task, now time.Time, fallback string) string {
	reasons := []string{fallback}
	if task.DueDate != nil {
		days := int(task.DueDate.Sub(dayStart(now)).Hours() / 24)
		switch {
		case days < 0:
			reasons = append(reasons, "overdue")
		case days == 0:
			reasons = append(reasons, "due today")
		case days <= 3:
			reasons = append(reasons, "due soon")
		}
	}
	if task.Priority > 0 {
		reasons = append(reasons, fmt.Sprintf("P%d", task.Priority))
	}
	reasons = append(reasons, "domain "+string(classifyDomain(task)))
	if task.Source != "" {
		reasons = append(reasons, "source "+string(task.Source))
	}
	if projectID := taskProjectID(task); projectID != "" {
		reasons = append(reasons, "project "+projectID)
	}
	if hasSyncRisk(task) {
		reasons = append(reasons, "sync risk: review before mutation")
	}
	return strings.Join(reasons, "; ")
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

func ptrTime(t time.Time) *time.Time { return &t }

// MockTasks generates a set of mock tasks covering all quadrants, statuses, and sources.
// Used by --mock flag on today/next/inbox commands for demo and testing.
func MockTasks(now time.Time) []model.Task {
	if now.IsZero() {
		now = time.Now()
	}
	return []model.Task{
		{ID: "mock_today", Title: "Finish TaskBridge daily workbench design", Status: model.StatusTodo, Source: model.SourceLocal, Domain: model.DomainWork, Priority: model.PriorityHigh, DueDate: ptrTime(dayStart(now).Add(17 * time.Hour)), CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour), EstimatedMinutes: 90},
		{ID: "mock_overdue", Title: "Review overdue tasks and decide outcomes", Status: model.StatusTodo, Source: model.SourceLocal, Domain: model.DomainWork, Priority: model.PriorityMedium, DueDate: ptrTime(dayStart(now).AddDate(0, 0, -2)), CreatedAt: now.AddDate(0, 0, -7), UpdatedAt: now.AddDate(0, 0, -2), EstimatedMinutes: 45},
		{ID: "mock_large", Title: "Split Agent safe execution layer into deliverable tasks", Status: model.StatusTodo, Source: model.SourceLocal, Domain: model.DomainWork, Priority: model.PriorityHigh, CreatedAt: now.AddDate(0, 0, -1), UpdatedAt: now.Add(-3 * time.Hour), EstimatedMinutes: 240},
		{ID: "mock_inbox", Title: "Confirm Todoist sync strategy", Status: model.StatusTodo, Source: model.SourceLocal, Priority: model.PriorityLow, CreatedAt: now.AddDate(0, 0, -1), UpdatedAt: now.AddDate(0, 0, -1)},
		{ID: "mock_q2_plan", Title: "Define Q3 product roadmap", Status: model.StatusTodo, Source: "todoist", Domain: model.DomainWork, Priority: model.PriorityMedium, DueDate: ptrTime(dayStart(now).AddDate(0, 0, 14)), CreatedAt: now.AddDate(0, 0, -3), UpdatedAt: now.Add(-24 * time.Hour), EstimatedMinutes: 120},
		{ID: "mock_q2_health", Title: "Schedule annual health check", Status: model.StatusInProgress, Source: model.SourceLocal, Domain: model.DomainLife, Priority: model.PriorityMedium, DueDate: ptrTime(dayStart(now).AddDate(0, 0, 2)), CreatedAt: now.AddDate(0, 0, -5), UpdatedAt: now.Add(-4 * time.Hour), EstimatedMinutes: 60},
		{ID: "mock_q3_reply", Title: "Reply to customer technical questions", Status: model.StatusTodo, Source: "google", Domain: model.DomainWork, Priority: model.PriorityLow, DueDate: ptrTime(dayStart(now).Add(12 * time.Hour)), CreatedAt: now.Add(-6 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour), EstimatedMinutes: 30},
		{ID: "mock_q3_meeting", Title: "Prepare weekly meeting update", Status: model.StatusTodo, Source: "microsoft", Domain: model.DomainWork, Priority: model.PriorityMedium, DueDate: ptrTime(dayStart(now).AddDate(0, 0, 1).Add(10 * time.Hour)), CreatedAt: now.AddDate(0, 0, -2), UpdatedAt: now.Add(-12 * time.Hour), EstimatedMinutes: 45},
		{ID: "mock_q4_cleanup", Title: "Clean up home paperwork", Status: model.StatusTodo, Source: model.SourceLocal, Domain: model.DomainLife, Priority: model.PriorityLow, CreatedAt: now.AddDate(0, 0, -10), UpdatedAt: now.AddDate(0, 0, -10), EstimatedMinutes: 20},
		{ID: "mock_completed", Title: "Complete auth module refactor", Status: model.StatusCompleted, Source: model.SourceLocal, Domain: model.DomainWork, Priority: model.PriorityHigh, DueDate: ptrTime(dayStart(now).AddDate(0, 0, -1).Add(18 * time.Hour)), CreatedAt: now.AddDate(0, 0, -14), UpdatedAt: now.AddDate(0, 0, -1), EstimatedMinutes: 180},
	}
}
