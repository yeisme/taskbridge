package governance

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
	pkgconfig "github.com/yeisme/taskbridge/pkg/config"
)

type Service struct {
	TaskStore storage.Storage
	Providers map[string]provider.Provider
	Config    pkgconfig.IntelligenceConfig
}

type FilterOptions struct {
	Source  string
	ListIDs []string
}

type OverdueHealthOptions struct {
	Filter             FilterOptions
	IncludeSuggestions bool
}

type ResolveOverdueOptions struct {
	ActionItems   []string
	DryRun        bool
	ConfirmDelete bool
}

type RebalanceLongTermOptions struct {
	Filter FilterOptions
	DryRun bool
}

type DetectDecompositionOptions struct {
	Filter FilterOptions
	Limit  int
}

type DecomposeTaskOptions struct {
	Provider   string
	Strategy   string
	WriteTasks bool
}

type AchievementOptions struct {
	WindowDays      int
	ComparePrevious bool
}

type overdueCandidate struct {
	TaskID        string     `json:"task_id"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	Source        string     `json:"source"`
	ListID        string     `json:"list_id,omitempty"`
	Priority      int        `json:"priority,omitempty"`
	Quadrant      int        `json:"quadrant,omitempty"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	DaysOverdue   int        `json:"days_overdue"`
	SevereOverdue bool       `json:"severe_overdue"`
}

type overdueActionItem struct {
	TaskID  string `json:"task_id"`
	Type    string `json:"type"`
	DueDate string `json:"due_date,omitempty"`
}

func (s *Service) OverdueHealth(ctx context.Context, opts OverdueHealthOptions) (map[string]interface{}, error) {
	cfg := effectiveIntelligenceConfig(s.Config)
	query := storage.Query{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress}, ListIDs: opts.Filter.ListIDs}
	if source := strings.TrimSpace(opts.Filter.Source); source != "" {
		resolved := provider.ResolveProviderName(source)
		if !provider.IsValidProvider(resolved) {
			return nil, fmt.Errorf("无效 provider: %s", source)
		}
		query.Sources = []model.TaskSource{model.TaskSource(resolved)}
	}
	tasks, err := s.TaskStore.QueryTasks(ctx, query)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	candidates := make([]overdueCandidate, 0)
	overdueCount, severeCount := 0, 0
	for _, task := range tasks {
		if task.DueDate == nil {
			continue
		}
		days := calcOverdueDays(task.DueDate, now)
		if days <= 0 {
			continue
		}
		overdueCount++
		severe := days >= cfg.Overdue.SevereDays
		if severe {
			severeCount++
		}
		if len(candidates) < cfg.Overdue.MaxCandidates {
			candidates = append(candidates, overdueCandidate{TaskID: task.ID, Title: task.Title, Status: string(task.Status), Source: string(task.Source), ListID: task.ListID, Priority: int(task.Priority), Quadrant: int(task.Quadrant), DueDate: task.DueDate, DaysOverdue: days, SevereOverdue: severe})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].DaysOverdue == candidates[j].DaysOverdue {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].DaysOverdue > candidates[j].DaysOverdue
	})
	result := map[string]interface{}{
		"summary":        map[string]interface{}{"overdue_count": overdueCount, "severe_overdue_count": severeCount, "is_warning": overdueCount > cfg.Overdue.WarningThreshold, "is_overload": overdueCount > cfg.Overdue.OverloadThreshold},
		"candidates":     candidates,
		"config_applied": map[string]interface{}{"warning_threshold": cfg.Overdue.WarningThreshold, "overload_threshold": cfg.Overdue.OverloadThreshold, "severe_days": cfg.Overdue.SevereDays, "max_candidates": cfg.Overdue.MaxCandidates},
	}
	if opts.IncludeSuggestions {
		result["actions"] = []string{"defer", "reschedule", "delete", "split_then_schedule"}
		result["questions"] = buildOverdueQuestions(overdueCount, overdueCount > cfg.Overdue.OverloadThreshold)
	}
	return result, nil
}

func (s *Service) ResolveOverdue(ctx context.Context, opts ResolveOverdueOptions) (map[string]interface{}, error) {
	if len(opts.ActionItems) == 0 {
		return nil, fmt.Errorf("请至少传入一个 --action taskID:type[:due_date]")
	}
	cfg := effectiveIntelligenceConfig(s.Config)
	allowDelete := !cfg.Overdue.AskBeforeDelete || opts.ConfirmDelete || opts.DryRun
	result := map[string]interface{}{"total": len(opts.ActionItems), "updated": 0, "deferred": 0, "rescheduled": 0, "deleted": 0, "split_suggested": 0, "skipped": 0, "errors": []string{}, "dry_run": opts.DryRun, "requires_confirm": cfg.Overdue.AskBeforeDelete, "confirm_token_match": allowDelete}
	appendErr := func(msg string) { result["errors"] = append(result["errors"].([]string), msg) }
	now := time.Now()
	for _, raw := range opts.ActionItems {
		action, err := parseOverdueAction(raw)
		if err != nil {
			result["skipped"] = result["skipped"].(int) + 1
			appendErr(err.Error())
			continue
		}
		task, err := s.TaskStore.GetTask(ctx, action.TaskID)
		if err != nil {
			result["skipped"] = result["skipped"].(int) + 1
			appendErr(fmt.Sprintf("task not found: %s", action.TaskID))
			continue
		}
		switch action.Type {
		case "defer":
			task.Status = model.StatusDeferred
			task.UpdatedAt = now
			if !opts.DryRun {
				if err := s.TaskStore.SaveTask(ctx, task); err != nil {
					result["skipped"] = result["skipped"].(int) + 1
					appendErr(fmt.Sprintf("defer %s failed: %v", action.TaskID, err))
					continue
				}
			}
			result["deferred"] = result["deferred"].(int) + 1
			result["updated"] = result["updated"].(int) + 1
		case "reschedule":
			dueDate, err := time.Parse("2006-01-02", strings.TrimSpace(action.DueDate))
			if err != nil {
				result["skipped"] = result["skipped"].(int) + 1
				appendErr(fmt.Sprintf("invalid due_date for %s: %s", action.TaskID, action.DueDate))
				continue
			}
			task.DueDate = &dueDate
			task.UpdatedAt = now
			if !opts.DryRun {
				if err := s.TaskStore.SaveTask(ctx, task); err != nil {
					result["skipped"] = result["skipped"].(int) + 1
					appendErr(fmt.Sprintf("reschedule %s failed: %v", action.TaskID, err))
					continue
				}
			}
			result["rescheduled"] = result["rescheduled"].(int) + 1
			result["updated"] = result["updated"].(int) + 1
		case "delete":
			if !allowDelete {
				result["skipped"] = result["skipped"].(int) + 1
				appendErr(fmt.Sprintf("delete %s blocked: add --confirm-delete", action.TaskID))
				continue
			}
			if !opts.DryRun {
				if err := s.TaskStore.DeleteTask(ctx, action.TaskID); err != nil {
					result["skipped"] = result["skipped"].(int) + 1
					appendErr(fmt.Sprintf("delete %s failed: %v", action.TaskID, err))
					continue
				}
			}
			result["deleted"] = result["deleted"].(int) + 1
		case "split_then_schedule":
			if task.Metadata == nil {
				task.Metadata = &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{}}
			}
			if task.Metadata.CustomFields == nil {
				task.Metadata.CustomFields = map[string]interface{}{}
			}
			task.Metadata.CustomFields["tb_split_suggested"] = true
			task.Metadata.CustomFields["tb_split_suggested_at"] = now.Format(time.RFC3339)
			task.UpdatedAt = now
			if !opts.DryRun {
				if err := s.TaskStore.SaveTask(ctx, task); err != nil {
					result["skipped"] = result["skipped"].(int) + 1
					appendErr(fmt.Sprintf("split_then_schedule %s failed: %v", action.TaskID, err))
					continue
				}
			}
			result["split_suggested"] = result["split_suggested"].(int) + 1
			result["updated"] = result["updated"].(int) + 1
		default:
			result["skipped"] = result["skipped"].(int) + 1
			appendErr(fmt.Sprintf("unsupported action type for %s: %s", action.TaskID, action.Type))
		}
	}
	return result, nil
}

func (s *Service) RebalanceLongTerm(ctx context.Context, opts RebalanceLongTermOptions) (map[string]interface{}, error) {
	cfg := effectiveIntelligenceConfig(s.Config)
	query := storage.Query{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress}, ListIDs: opts.Filter.ListIDs}
	if source := strings.TrimSpace(opts.Filter.Source); source != "" {
		resolved := provider.ResolveProviderName(source)
		if !provider.IsValidProvider(resolved) {
			return nil, fmt.Errorf("无效 provider: %s", source)
		}
		query.Sources = []model.TaskSource{model.TaskSource(resolved)}
	}
	tasks, err := s.TaskStore.QueryTasks(ctx, query)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	windowEnd := startOfToday.AddDate(0, 0, cfg.LongTerm.ShortTermWindowDays)
	shortTerm, longTerm := make([]model.Task, 0), make([]model.Task, 0)
	for _, task := range tasks {
		if task.DueDate != nil {
			if !task.DueDate.Before(startOfToday) && !task.DueDate.After(windowEnd) {
				shortTerm = append(shortTerm, task)
			}
			continue
		}
		if calcAgeDays(task.CreatedAt, now) >= cfg.LongTerm.MinAgeDays {
			longTerm = append(longTerm, task)
		}
	}
	sort.SliceStable(longTerm, func(i, j int) bool {
		si, sj := scoreLongTermTask(longTerm[i], now), scoreLongTermTask(longTerm[j], now)
		if si == sj {
			return longTerm[i].UpdatedAt.After(longTerm[j].UpdatedAt)
		}
		return si > sj
	})
	promotedIDs, retainedIDs, adjustedIDs := []string{}, []string{}, []string{}
	mode := "balanced"
	if len(shortTerm) < cfg.LongTerm.ShortTermMin {
		mode = "shortage"
		promoteCount := cfg.LongTerm.PromoteCountWhenShortage
		if promoteCount <= 0 {
			promoteCount = 1
		}
		if promoteCount > len(longTerm) {
			promoteCount = len(longTerm)
		}
		for i := 0; i < promoteCount; i++ {
			task := longTerm[i]
			due := startOfToday.AddDate(0, 0, i+1)
			task.DueDate = &due
			task.UpdatedAt = now
			if task.Status == model.StatusDeferred {
				task.Status = model.StatusTodo
			}
			if !opts.DryRun {
				if err := s.TaskStore.SaveTask(ctx, &task); err != nil {
					return nil, err
				}
			}
			promotedIDs = append(promotedIDs, task.ID)
			adjustedIDs = append(adjustedIDs, task.ID)
		}
		for i := promoteCount; i < len(longTerm); i++ {
			retainedIDs = append(retainedIDs, longTerm[i].ID)
		}
	} else {
		for _, task := range longTerm {
			retainedIDs = append(retainedIDs, task.ID)
		}
	}
	return map[string]interface{}{"mode": mode, "short_term_count": len(shortTerm), "long_term_count": len(longTerm), "promoted_task_ids": promotedIDs, "retained_task_ids": retainedIDs, "adjusted_task_ids": adjustedIDs, "dry_run": opts.DryRun}, nil
}

func (s *Service) DetectDecomposition(ctx context.Context, opts DetectDecompositionOptions) (map[string]interface{}, error) {
	cfg := effectiveIntelligenceConfig(s.Config)
	query := storage.Query{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress}, ListIDs: opts.Filter.ListIDs}
	if source := strings.TrimSpace(opts.Filter.Source); source != "" {
		resolved := provider.ResolveProviderName(source)
		if !provider.IsValidProvider(resolved) {
			return nil, fmt.Errorf("无效 provider: %s", source)
		}
		query.Sources = []model.TaskSource{model.TaskSource(resolved)}
	}
	tasks, err := s.TaskStore.QueryTasks(ctx, query)
	if err != nil {
		return nil, err
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	type candidateItem struct {
		TaskID                   string   `json:"task_id"`
		Title                    string   `json:"title"`
		ComplexityScore          int      `json:"complexity_score"`
		ReasonCodes              []string `json:"reason_codes"`
		HasSubtasks              bool     `json:"has_subtasks"`
		RecommendedProvider      string   `json:"recommended_provider"`
		RecommendedStrategy      string   `json:"recommended_strategy"`
		ProviderSupportsSubtasks bool     `json:"provider_supports_subtasks"`
	}
	candidates := make([]candidateItem, 0)
	for _, task := range tasks {
		score, reasons := computeTaskComplexity(task, cfg.Decompose)
		if score < cfg.Decompose.ComplexityThreshold {
			continue
		}
		hasSubtasks := taskCustomInt(task, "tb_subtask_count") > 0 || containsFold(task.Tags, "subtask")
		if hasSubtasks {
			continue
		}
		recommendedProvider, supportsSubtasks := recommendProviderForTask(s.Providers, task, "")
		strategy := strings.TrimSpace(cfg.Decompose.PreferredStrategy)
		if strategy == "" {
			strategy = "project_split"
		}
		candidates = append(candidates, candidateItem{TaskID: task.ID, Title: task.Title, ComplexityScore: score, ReasonCodes: reasons, HasSubtasks: hasSubtasks, RecommendedProvider: recommendedProvider, RecommendedStrategy: strategy, ProviderSupportsSubtasks: supportsSubtasks})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].ComplexityScore == candidates[j].ComplexityScore {
			return candidates[i].TaskID < candidates[j].TaskID
		}
		return candidates[i].ComplexityScore > candidates[j].ComplexityScore
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return map[string]interface{}{"summary": map[string]interface{}{"total_scanned": len(tasks), "candidate_count": len(candidates), "threshold": cfg.Decompose.ComplexityThreshold, "limit": limit}, "candidates": candidates}, nil
}

func (s *Service) DecomposeTask(ctx context.Context, taskID string, opts DecomposeTaskOptions) (map[string]interface{}, error) {
	task, err := s.TaskStore.GetTask(ctx, strings.TrimSpace(taskID))
	if err != nil {
		return nil, err
	}
	cfg := effectiveIntelligenceConfig(s.Config)
	providerName, supportsSubtasks := recommendProviderForTask(s.Providers, *task, opts.Provider)
	strategy := strings.TrimSpace(opts.Strategy)
	if strategy == "" {
		strategy = strings.TrimSpace(cfg.Decompose.PreferredStrategy)
	}
	if strategy == "" {
		strategy = "project_split"
	}
	preview := buildDecomposePreview(*task, supportsSubtasks)
	createdIDs := make([]string, 0)
	planID := fmt.Sprintf("decomp_%d", time.Now().UnixNano())
	if opts.WriteTasks {
		createdIDs, err = writeDecomposePreviewTasks(ctx, s.TaskStore, *task, providerName, strategy, planID, preview)
		if err != nil {
			return nil, err
		}
	}
	warnings := make([]string, 0)
	if !supportsSubtasks {
		warnings = append(warnings, "目标 provider 不支持子任务，已建议使用扁平任务与阶段标签")
	}
	return map[string]interface{}{"task_id": task.ID, "plan_id": planID, "provider": providerName, "strategy": strategy, "provider_capability_used": map[string]interface{}{"supports_subtasks": supportsSubtasks}, "tasks_preview": preview, "created_task_ids": createdIDs, "write_tasks": opts.WriteTasks, "warnings": warnings}, nil
}

func (s *Service) Achievement(ctx context.Context, opts AchievementOptions) (map[string]interface{}, error) {
	cfg := effectiveIntelligenceConfig(s.Config)
	windowDays := opts.WindowDays
	if windowDays < 7 {
		windowDays = 7
	}
	tasks, err := s.TaskStore.ListTasks(ctx, storage.ListOptions{})
	if err != nil {
		return nil, err
	}
	loc := resolveLocation(cfg.Timezone)
	now := time.Now().In(loc)
	start := now.AddDate(0, 0, -windowDays)
	prevStart := start.AddDate(0, 0, -windowDays)
	completedCount, activeCount, onTimeCount, overdueFixedCount, previousCompleted := 0, 0, 0, 0, 0
	completedByQuadrant := map[string]int{"q1": 0, "q2": 0, "q3": 0, "q4": 0}
	daily := make(map[string]int)
	for _, task := range tasks {
		if task.Status != model.StatusCompleted {
			activeCount++
		}
		completedAt := completionTime(task)
		if completedAt == nil {
			continue
		}
		completedTime := completedAt.In(loc)
		if !completedTime.Before(start) {
			completedCount++
			daily[completedTime.Format("2006-01-02")]++
			quadrantKey := fmt.Sprintf("q%d", int(task.Quadrant))
			if _, ok := completedByQuadrant[quadrantKey]; !ok {
				quadrantKey = "q4"
			}
			completedByQuadrant[quadrantKey]++
			if task.DueDate != nil {
				if !completedTime.After(task.DueDate.In(loc)) {
					onTimeCount++
				} else {
					overdueFixedCount++
				}
			}
		}
		if opts.ComparePrevious && !completedTime.Before(prevStart) && completedTime.Before(start) {
			previousCompleted++
		}
	}
	streakGoal := cfg.Achievement.StreakGoalPerDay
	if streakGoal <= 0 {
		streakGoal = 1
	}
	streak := calcCompletionStreak(daily, now, streakGoal)
	onTimeRate := 0.0
	if completedCount > 0 {
		onTimeRate = float64(onTimeCount) / float64(completedCount)
	}
	avgPerDay := float64(completedCount) / float64(windowDays)
	delta := completedCount - previousCompleted
	trendText := "持平"
	if delta > 0 {
		trendText = fmt.Sprintf("上升 %+d", delta)
	} else if delta < 0 {
		trendText = fmt.Sprintf("下降 %d", delta)
	}
	badges := []string{}
	if cfg.Achievement.BadgeEnabled {
		if streak >= 7 {
			badges = append(badges, "steady-7")
		}
		if overdueFixedCount >= 5 {
			badges = append(badges, "overdue-cleaner")
		}
		if completedCount > 0 && float64(completedByQuadrant["q2"])/float64(completedCount) >= 0.4 {
			badges = append(badges, "q2-builder")
		}
	}
	narrative := ""
	if cfg.Achievement.NarrativeEnabled {
		narrative = fmt.Sprintf("过去 %d 天你完成了 %d 项任务，按时完成率 %.1f%%，当前连续完成 %d 天。趋势：%s。", windowDays, completedCount, onTimeRate*100, streak, trendText)
	}
	nextActions := []string{}
	if onTimeRate < 0.6 {
		nextActions = append(nextActions, "按时完成率偏低：建议先清理逾期项，并减少本周承诺任务数")
	}
	if streak < 3 {
		nextActions = append(nextActions, "连续完成天数较低：建议设置“每天至少完成 1 个任务”的目标")
	}
	if completedCount > 0 && completedByQuadrant["q2"]*3 < completedCount {
		nextActions = append(nextActions, "Q2（重要不紧急）占比偏低：建议每周固定投入 2~3 个 Q2 任务")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "保持当前节奏，逐步提升任务拆分质量与计划稳定性")
	}
	return map[string]interface{}{"metrics": map[string]interface{}{"window_days": windowDays, "completed_count": completedCount, "active_count": activeCount, "on_time_rate": onTimeRate, "streak_days": streak, "avg_completed_per_day": avgPerDay, "overdue_fixed_count": overdueFixedCount, "quadrant_completed": completedByQuadrant, "compare_previous": opts.ComparePrevious, "previous_completed": previousCompleted, "delta_completed": delta, "trend": trendText}, "badges": badges, "narrative": narrative, "next_actions": nextActions}, nil
}

func effectiveIntelligenceConfig(out pkgconfig.IntelligenceConfig) pkgconfig.IntelligenceConfig {
	defaults := pkgconfig.DefaultConfig().Intelligence
	if strings.TrimSpace(out.Timezone) == "" {
		out.Timezone = defaults.Timezone
	}
	if out.Overdue.WarningThreshold == 0 {
		out.Overdue.WarningThreshold = defaults.Overdue.WarningThreshold
	}
	if out.Overdue.OverloadThreshold == 0 {
		out.Overdue.OverloadThreshold = defaults.Overdue.OverloadThreshold
	}
	if out.Overdue.SevereDays == 0 {
		out.Overdue.SevereDays = defaults.Overdue.SevereDays
	}
	if out.Overdue.MaxCandidates == 0 {
		out.Overdue.MaxCandidates = defaults.Overdue.MaxCandidates
	}
	if out.LongTerm.MinAgeDays == 0 {
		out.LongTerm.MinAgeDays = defaults.LongTerm.MinAgeDays
	}
	if out.LongTerm.ShortTermWindowDays == 0 {
		out.LongTerm.ShortTermWindowDays = defaults.LongTerm.ShortTermWindowDays
	}
	if out.LongTerm.ShortTermMin == 0 {
		out.LongTerm.ShortTermMin = defaults.LongTerm.ShortTermMin
	}
	if out.LongTerm.ShortTermMax == 0 {
		out.LongTerm.ShortTermMax = defaults.LongTerm.ShortTermMax
	}
	if out.LongTerm.PromoteCountWhenShortage == 0 {
		out.LongTerm.PromoteCountWhenShortage = defaults.LongTerm.PromoteCountWhenShortage
	}
	if out.Decompose.ComplexityThreshold == 0 {
		out.Decompose.ComplexityThreshold = defaults.Decompose.ComplexityThreshold
	}
	if strings.TrimSpace(out.Decompose.PreferredStrategy) == "" {
		out.Decompose.PreferredStrategy = defaults.Decompose.PreferredStrategy
	}
	if out.Achievement.StreakGoalPerDay == 0 {
		out.Achievement.StreakGoalPerDay = defaults.Achievement.StreakGoalPerDay
	}
	return out
}

func parseOverdueAction(raw string) (overdueActionItem, error) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) < 2 {
		return overdueActionItem{}, fmt.Errorf("invalid action: %s", raw)
	}
	item := overdueActionItem{TaskID: strings.TrimSpace(parts[0]), Type: strings.TrimSpace(parts[1])}
	if item.TaskID == "" || item.Type == "" {
		return overdueActionItem{}, fmt.Errorf("invalid action: %s", raw)
	}
	if len(parts) > 2 {
		item.DueDate = strings.TrimSpace(parts[2])
	}
	return item, nil
}

func computeTaskComplexity(task model.Task, cfg pkgconfig.DecomposeConfig) (int, []string) {
	score, reasons := 0, []string{}
	titleLen := len([]rune(strings.TrimSpace(task.Title)))
	if titleLen >= 20 {
		score += 20
		reasons = append(reasons, "long_title")
	}
	if descLen := len([]rune(strings.TrimSpace(task.Description))); descLen >= 80 {
		score += 25
		reasons = append(reasons, "long_description")
	}
	if task.Priority >= model.PriorityHigh {
		score += 15
		reasons = append(reasons, "high_priority")
	}
	if task.Quadrant == model.QuadrantUrgentImportant {
		score += 10
		reasons = append(reasons, "quadrant_q1")
	}
	if calcAgeDays(task.CreatedAt, time.Now()) >= 14 {
		score += 10
		reasons = append(reasons, "old_task")
	}
	if len(task.Tags) >= 3 {
		score += 10
		reasons = append(reasons, "many_tags")
	}
	return score, reasons
}

func recommendProviderForTask(providers map[string]provider.Provider, task model.Task, preferred string) (string, bool) {
	preferred = provider.ResolveProviderName(strings.TrimSpace(preferred))
	if preferred != "" {
		if p, ok := providers[preferred]; ok {
			return preferred, p.Capabilities().SupportsSubtasks
		}
		return preferred, false
	}
	if task.Source != "" {
		if p, ok := providers[string(task.Source)]; ok {
			return string(task.Source), p.Capabilities().SupportsSubtasks
		}
	}
	for name, p := range providers {
		if p.Capabilities().SupportsSubtasks {
			return name, true
		}
	}
	for name, p := range providers {
		return name, p.Capabilities().SupportsSubtasks
	}
	return "", false
}

func buildDecomposePreview(task model.Task, supportsSubtasks bool) []map[string]interface{} {
	steps := []string{"明确目标", "准备上下文", "执行核心动作"}
	if supportsSubtasks {
		steps = append(steps, "验证并收尾")
	}
	out := make([]map[string]interface{}, 0, len(steps))
	for i, title := range steps {
		out = append(out, map[string]interface{}{"title": fmt.Sprintf("%s-%d", title, i+1), "estimate_minutes": 30 + i*15, "priority": max(1, int(task.Priority)), "quadrant": int(task.Quadrant)})
	}
	return out
}

func writeDecomposePreviewTasks(ctx context.Context, taskStore storage.Storage, parentTask model.Task, providerName, strategy, planID string, preview []map[string]interface{}) ([]string, error) {
	createdIDs := make([]string, 0, len(preview))
	now := time.Now()
	for idx, item := range preview {
		title := strings.TrimSpace(fmt.Sprint(item["title"]))
		if title == "" {
			title = fmt.Sprintf("%s-%d", parentTask.Title, idx+1)
		}
		child := &model.Task{ID: fmt.Sprintf("decomp-%d-%d", now.UnixNano(), idx), Title: title, Status: model.StatusTodo, Source: parentTask.Source, ListID: parentTask.ListID, Priority: parentTask.Priority, Quadrant: parentTask.Quadrant, CreatedAt: now, UpdatedAt: now, Metadata: &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{"tb_parent_task_id": parentTask.ID, "tb_decompose_plan_id": planID, "tb_provider": providerName, "tb_strategy": strategy}}}
		if err := taskStore.SaveTask(ctx, child); err != nil {
			return nil, err
		}
		createdIDs = append(createdIDs, child.ID)
	}
	return createdIDs, nil
}

func resolveLocation(tz string) *time.Location {
	if strings.TrimSpace(tz) == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(strings.TrimSpace(tz))
	if err != nil {
		return time.Local
	}
	return loc
}
func completionTime(task model.Task) *time.Time {
	if task.CompletedAt != nil {
		return task.CompletedAt
	}
	if task.Status == model.StatusCompleted && task.UpdatedAt.After(task.CreatedAt) {
		t := task.UpdatedAt
		return &t
	}
	return nil
}
func calcCompletionStreak(daily map[string]int, now time.Time, goalPerDay int) int {
	streak := 0
	cursor := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for {
		if daily[cursor.Format("2006-01-02")] < goalPerDay {
			break
		}
		streak++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return streak
}
func buildOverdueQuestions(overdueCount int, overload bool) []string {
	qs := []string{"哪些任务应该延期？", "哪些任务应该删除？"}
	if overdueCount > 0 {
		qs = append(qs, "哪些任务值得拆分后再排期？")
	}
	if overload {
		qs = append(qs, "是否需要降低近期承诺数量？")
	}
	return qs
}
func calcOverdueDays(due *time.Time, now time.Time) int {
	if due == nil {
		return 0
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
	return int(start.Sub(dueDay).Hours() / 24)
}
func calcAgeDays(createdAt, now time.Time) int { return int(now.Sub(createdAt).Hours() / 24) }
func scoreLongTermTask(task model.Task, now time.Time) int {
	score := calcAgeDays(task.CreatedAt, now)
	if task.Priority >= model.PriorityHigh {
		score += 20
	}
	if task.Quadrant == model.QuadrantNotUrgentImportant {
		score += 10
	}
	return score
}
func taskCustomInt(task model.Task, key string) int {
	if task.Metadata == nil || task.Metadata.CustomFields == nil {
		return 0
	}
	return anyToInt(task.Metadata.CustomFields[key])
}
func anyToInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(v))
		return parsed
	default:
		return 0
	}
}
func containsFold(values []string, target string) bool {
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), target) {
			return true
		}
	}
	return false
}
