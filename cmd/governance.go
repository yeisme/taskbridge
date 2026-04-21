package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
	pkgconfig "github.com/yeisme/taskbridge/pkg/config"
)

var (
	governanceFormat             string
	governanceSource             string
	governanceListIDs            []string
	governanceIncludeSuggestions bool
	governanceDryRun             bool
	governanceActionItems        []string
	governanceConfirmDelete      bool
	governanceLimit              int
	governanceProvider           string
	governanceStrategy           string
	governanceWriteTasks         bool
	governanceWindowDays         int
	governanceComparePrevious    bool
)

var governanceCmd = &cobra.Command{
	Use:   "governance",
	Short: "任务治理与智能辅助",
	Long: `运行 TaskBridge 的 CLI 治理能力，包括逾期健康分析、长期任务调配、
复杂任务识别、任务拆分建议和成就分析。`,
}

var governanceOverdueHealthCmd = &cobra.Command{
	Use:   "overdue-health",
	Short: "分析逾期任务健康度",
	Run:   runGovernanceOverdueHealth,
}

var governanceResolveOverdueCmd = &cobra.Command{
	Use:   "resolve-overdue",
	Short: "批量处理逾期任务",
	Run:   runGovernanceResolveOverdue,
}

var governanceRebalanceLongTermCmd = &cobra.Command{
	Use:   "rebalance-longterm",
	Short: "调配长期无排期任务",
	Run:   runGovernanceRebalanceLongTerm,
}

var governanceDetectDecompositionCmd = &cobra.Command{
	Use:   "detect-decomposition",
	Short: "识别复杂且缺少子任务的候选任务",
	Run:   runGovernanceDetectDecomposition,
}

var governanceDecomposeTaskCmd = &cobra.Command{
	Use:   "decompose-task <task-id>",
	Short: "将单个任务拆分为执行步骤",
	Args:  cobra.ExactArgs(1),
	Run:   runGovernanceDecomposeTask,
}

var governanceAchievementCmd = &cobra.Command{
	Use:   "achievement",
	Short: "分析完成情况与成就反馈",
	Run:   runGovernanceAchievement,
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

func init() {
	rootCmd.AddCommand(governanceCmd)
	governanceCmd.AddCommand(governanceOverdueHealthCmd)
	governanceCmd.AddCommand(governanceResolveOverdueCmd)
	governanceCmd.AddCommand(governanceRebalanceLongTermCmd)
	governanceCmd.AddCommand(governanceDetectDecompositionCmd)
	governanceCmd.AddCommand(governanceDecomposeTaskCmd)
	governanceCmd.AddCommand(governanceAchievementCmd)

	for _, cmd := range []*cobra.Command{
		governanceOverdueHealthCmd,
		governanceResolveOverdueCmd,
		governanceRebalanceLongTermCmd,
		governanceDetectDecompositionCmd,
		governanceDecomposeTaskCmd,
		governanceAchievementCmd,
	} {
		cmd.Flags().StringVarP(&governanceFormat, "format", "f", "json", "输出格式 (json, text)")
	}

	for _, cmd := range []*cobra.Command{
		governanceOverdueHealthCmd,
		governanceRebalanceLongTermCmd,
		governanceDetectDecompositionCmd,
	} {
		cmd.Flags().StringVar(&governanceSource, "source", "", "按来源筛选（支持简写）")
		cmd.Flags().StringSliceVar(&governanceListIDs, "list-id", nil, "按清单 ID 筛选（可重复）")
	}

	governanceOverdueHealthCmd.Flags().BoolVar(&governanceIncludeSuggestions, "include-suggestions", true, "返回建议动作与追问")

	governanceResolveOverdueCmd.Flags().StringSliceVar(&governanceActionItems, "action", nil, "处理动作 taskID:type[:due_date]，可重复")
	governanceResolveOverdueCmd.Flags().BoolVar(&governanceDryRun, "dry-run", false, "模拟执行")
	governanceResolveOverdueCmd.Flags().BoolVar(&governanceConfirmDelete, "confirm-delete", false, "确认允许 delete 动作")

	governanceRebalanceLongTermCmd.Flags().BoolVar(&governanceDryRun, "dry-run", false, "模拟执行")

	governanceDetectDecompositionCmd.Flags().IntVar(&governanceLimit, "limit", 20, "返回条数")

	governanceDecomposeTaskCmd.Flags().StringVar(&governanceProvider, "provider", "", "优先使用的 Provider")
	governanceDecomposeTaskCmd.Flags().StringVar(&governanceStrategy, "strategy", "", "拆分策略")
	governanceDecomposeTaskCmd.Flags().BoolVar(&governanceWriteTasks, "write-tasks", false, "将拆分结果写入本地任务")

	governanceAchievementCmd.Flags().IntVar(&governanceWindowDays, "window-days", 30, "统计窗口天数")
	governanceAchievementCmd.Flags().BoolVar(&governanceComparePrevious, "compare-previous", true, "是否对比上一周期")
}

func runGovernanceOverdueHealth(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	taskStore, _, err := getCLIStores()
	if err != nil {
		exitErr("初始化存储失败", err)
	}
	cfg := effectiveIntelligenceConfig()
	query := storage.Query{
		Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress},
		ListIDs:  governanceListIDs,
	}
	if source := strings.TrimSpace(governanceSource); source != "" {
		resolved := provider.ResolveProviderName(source)
		if !provider.IsValidProvider(resolved) {
			exitErr("无效 provider", fmt.Errorf("%s", source))
		}
		query.Sources = []model.TaskSource{model.TaskSource(resolved)}
	}

	tasks, err := taskStore.QueryTasks(ctx, query)
	if err != nil {
		exitErr("查询任务失败", err)
	}

	now := time.Now()
	candidates := make([]overdueCandidate, 0)
	overdueCount := 0
	severeCount := 0
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
			candidates = append(candidates, overdueCandidate{
				TaskID:        task.ID,
				Title:         task.Title,
				Status:        string(task.Status),
				Source:        string(task.Source),
				ListID:        task.ListID,
				Priority:      int(task.Priority),
				Quadrant:      int(task.Quadrant),
				DueDate:       task.DueDate,
				DaysOverdue:   days,
				SevereOverdue: severe,
			})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].DaysOverdue == candidates[j].DaysOverdue {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].DaysOverdue > candidates[j].DaysOverdue
	})

	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"overdue_count":        overdueCount,
			"severe_overdue_count": severeCount,
			"is_warning":           overdueCount > cfg.Overdue.WarningThreshold,
			"is_overload":          overdueCount > cfg.Overdue.OverloadThreshold,
		},
		"candidates": candidates,
		"config_applied": map[string]interface{}{
			"warning_threshold":  cfg.Overdue.WarningThreshold,
			"overload_threshold": cfg.Overdue.OverloadThreshold,
			"severe_days":        cfg.Overdue.SevereDays,
			"max_candidates":     cfg.Overdue.MaxCandidates,
		},
	}
	if governanceIncludeSuggestions {
		result["actions"] = []string{"defer", "reschedule", "delete", "split_then_schedule"}
		result["questions"] = buildOverdueQuestions(overdueCount, overdueCount > cfg.Overdue.OverloadThreshold)
	}
	printGovernanceResult(result)
}

func runGovernanceResolveOverdue(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	taskStore, _, err := getCLIStores()
	if err != nil {
		exitErr("初始化存储失败", err)
	}
	if len(governanceActionItems) == 0 {
		exitErr("缺少 action", fmt.Errorf("请至少传入一个 --action taskID:type[:due_date]"))
	}
	cfg := effectiveIntelligenceConfig()
	allowDelete := !cfg.Overdue.AskBeforeDelete || governanceConfirmDelete || governanceDryRun

	result := map[string]interface{}{
		"total":               len(governanceActionItems),
		"updated":             0,
		"deferred":            0,
		"rescheduled":         0,
		"deleted":             0,
		"split_suggested":     0,
		"skipped":             0,
		"errors":              []string{},
		"dry_run":             governanceDryRun,
		"requires_confirm":    cfg.Overdue.AskBeforeDelete,
		"confirm_token_match": allowDelete,
	}
	appendErr := func(msg string) {
		result["errors"] = append(result["errors"].([]string), msg)
	}

	now := time.Now()
	for _, raw := range governanceActionItems {
		action, err := parseOverdueAction(raw)
		if err != nil {
			result["skipped"] = result["skipped"].(int) + 1
			appendErr(err.Error())
			continue
		}
		task, err := taskStore.GetTask(ctx, action.TaskID)
		if err != nil {
			result["skipped"] = result["skipped"].(int) + 1
			appendErr(fmt.Sprintf("task not found: %s", action.TaskID))
			continue
		}

		switch action.Type {
		case "defer":
			task.Status = model.StatusDeferred
			task.UpdatedAt = now
			if !governanceDryRun {
				if err := taskStore.SaveTask(ctx, task); err != nil {
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
			if !governanceDryRun {
				if err := taskStore.SaveTask(ctx, task); err != nil {
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
			if !governanceDryRun {
				if err := taskStore.DeleteTask(ctx, action.TaskID); err != nil {
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
			if !governanceDryRun {
				if err := taskStore.SaveTask(ctx, task); err != nil {
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

	printGovernanceResult(result)
}

func runGovernanceRebalanceLongTerm(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	taskStore, _, err := getCLIStores()
	if err != nil {
		exitErr("初始化存储失败", err)
	}
	cfg := effectiveIntelligenceConfig()
	query := storage.Query{
		Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress},
		ListIDs:  governanceListIDs,
	}
	if source := strings.TrimSpace(governanceSource); source != "" {
		resolved := provider.ResolveProviderName(source)
		if !provider.IsValidProvider(resolved) {
			exitErr("无效 provider", fmt.Errorf("%s", source))
		}
		query.Sources = []model.TaskSource{model.TaskSource(resolved)}
	}

	tasks, err := taskStore.QueryTasks(ctx, query)
	if err != nil {
		exitErr("查询任务失败", err)
	}

	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	windowEnd := startOfToday.AddDate(0, 0, cfg.LongTerm.ShortTermWindowDays)

	shortTerm := make([]model.Task, 0)
	longTerm := make([]model.Task, 0)
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
		scoreI := scoreLongTermTask(longTerm[i], now)
		scoreJ := scoreLongTermTask(longTerm[j], now)
		if scoreI == scoreJ {
			return longTerm[i].UpdatedAt.After(longTerm[j].UpdatedAt)
		}
		return scoreI > scoreJ
	})

	promotedIDs := make([]string, 0)
	retainedIDs := make([]string, 0)
	adjustedIDs := make([]string, 0)
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
			if !governanceDryRun {
				_ = taskStore.SaveTask(ctx, &task)
			}
			promotedIDs = append(promotedIDs, task.ID)
		}
	}

	if len(shortTerm) > cfg.LongTerm.ShortTermMax {
		mode = "overflow"
		retainCount := cfg.LongTerm.RetainCountWhenOverflow
		if retainCount <= 0 {
			retainCount = 1
		}
		if retainCount > len(longTerm) {
			retainCount = len(longTerm)
		}
		for i := 0; i < retainCount; i++ {
			retainedIDs = append(retainedIDs, longTerm[i].ID)
		}
		for i := retainCount; i < len(longTerm); i++ {
			task := longTerm[i]
			if strings.EqualFold(strings.TrimSpace(cfg.LongTerm.OverflowStrategy), "backlog_tag") {
				if !containsFold(task.Tags, "backlog") {
					task.Tags = append(task.Tags, "backlog")
				}
			} else {
				task.Status = model.StatusDeferred
			}
			task.UpdatedAt = now
			if !governanceDryRun {
				_ = taskStore.SaveTask(ctx, &task)
			}
			adjustedIDs = append(adjustedIDs, task.ID)
		}
	}

	shortTermAfter := len(shortTerm)
	if mode == "shortage" {
		shortTermAfter += len(promotedIDs)
	}
	printGovernanceResult(map[string]interface{}{
		"mode":               mode,
		"short_term_before":  len(shortTerm),
		"short_term_after":   shortTermAfter,
		"long_term_pool":     len(longTerm),
		"promoted_tasks":     promotedIDs,
		"retained_long_term": retainedIDs,
		"adjusted_long_term": adjustedIDs,
		"dry_run":            governanceDryRun,
		"config_applied": map[string]interface{}{
			"short_term_min":              cfg.LongTerm.ShortTermMin,
			"short_term_max":              cfg.LongTerm.ShortTermMax,
			"promote_count_when_shortage": cfg.LongTerm.PromoteCountWhenShortage,
			"retain_count_when_overflow":  cfg.LongTerm.RetainCountWhenOverflow,
			"overflow_strategy":           cfg.LongTerm.OverflowStrategy,
		},
	})
}

func runGovernanceDetectDecomposition(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	taskStore, _, err := getCLIStores()
	if err != nil {
		exitErr("初始化存储失败", err)
	}
	cfg := effectiveIntelligenceConfig()
	query := storage.Query{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress}}
	if source := strings.TrimSpace(governanceSource); source != "" {
		resolved := provider.ResolveProviderName(source)
		if !provider.IsValidProvider(resolved) {
			exitErr("无效 provider", fmt.Errorf("%s", source))
		}
		query.Sources = []model.TaskSource{model.TaskSource(resolved)}
	}
	tasks, err := taskStore.QueryTasks(ctx, query)
	if err != nil {
		exitErr("查询任务失败", err)
	}
	limit := governanceLimit
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

	providers, _ := loadAuthenticatedProviders("")
	candidates := make([]candidateItem, 0)
	for _, task := range tasks {
		hasSubtasks := len(task.SubtaskIDs) > 0
		score, reasons := computeTaskComplexity(task, cfg.Decompose)
		if score < cfg.Decompose.ComplexityThreshold || hasSubtasks {
			continue
		}
		recommendedProvider, supportsSubtasks := recommendProviderForTask(providers, task, "")
		strategy := strings.TrimSpace(cfg.Decompose.PreferredStrategy)
		if strategy == "" {
			strategy = "project_split"
		}
		candidates = append(candidates, candidateItem{
			TaskID:                   task.ID,
			Title:                    task.Title,
			ComplexityScore:          score,
			ReasonCodes:              reasons,
			HasSubtasks:              hasSubtasks,
			RecommendedProvider:      recommendedProvider,
			RecommendedStrategy:      strategy,
			ProviderSupportsSubtasks: supportsSubtasks,
		})
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
	printGovernanceResult(map[string]interface{}{
		"summary": map[string]interface{}{
			"total_scanned":   len(tasks),
			"candidate_count": len(candidates),
			"threshold":       cfg.Decompose.ComplexityThreshold,
			"limit":           limit,
		},
		"candidates": candidates,
	})
}

func runGovernanceDecomposeTask(_ *cobra.Command, args []string) {
	ctx := context.Background()
	taskStore, _, err := getCLIStores()
	if err != nil {
		exitErr("初始化存储失败", err)
	}
	task, err := taskStore.GetTask(ctx, strings.TrimSpace(args[0]))
	if err != nil {
		exitErr("读取任务失败", err)
	}
	providers, _ := loadAuthenticatedProviders("")
	cfg := effectiveIntelligenceConfig()
	providerName, supportsSubtasks := recommendProviderForTask(providers, *task, governanceProvider)
	strategy := strings.TrimSpace(governanceStrategy)
	if strategy == "" {
		strategy = strings.TrimSpace(cfg.Decompose.PreferredStrategy)
	}
	if strategy == "" {
		strategy = "project_split"
	}
	preview := buildDecomposePreview(*task, supportsSubtasks)
	createdIDs := make([]string, 0)
	planID := fmt.Sprintf("decomp_%d", time.Now().UnixNano())
	if governanceWriteTasks {
		createdIDs, err = writeDecomposePreviewTasks(ctx, taskStore, *task, providerName, strategy, planID, preview)
		if err != nil {
			exitErr("写入拆分任务失败", err)
		}
	}
	warnings := make([]string, 0)
	if !supportsSubtasks {
		warnings = append(warnings, "目标 provider 不支持子任务，已建议使用扁平任务与阶段标签")
	}
	printGovernanceResult(map[string]interface{}{
		"task_id":                  task.ID,
		"plan_id":                  planID,
		"provider":                 providerName,
		"strategy":                 strategy,
		"provider_capability_used": map[string]interface{}{"supports_subtasks": supportsSubtasks},
		"tasks_preview":            preview,
		"created_task_ids":         createdIDs,
		"write_tasks":              governanceWriteTasks,
		"warnings":                 warnings,
	})
}

func runGovernanceAchievement(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	taskStore, _, err := getCLIStores()
	if err != nil {
		exitErr("初始化存储失败", err)
	}
	cfg := effectiveIntelligenceConfig()
	windowDays := governanceWindowDays
	if windowDays < 7 {
		windowDays = 7
	}
	tasks, err := taskStore.ListTasks(ctx, storage.ListOptions{})
	if err != nil {
		exitErr("读取任务失败", err)
	}
	loc := resolveLocation(cfg.Timezone)
	now := time.Now().In(loc)
	start := now.AddDate(0, 0, -windowDays)
	prevStart := start.AddDate(0, 0, -windowDays)

	completedCount := 0
	activeCount := 0
	onTimeCount := 0
	overdueFixedCount := 0
	completedByQuadrant := map[string]int{"q1": 0, "q2": 0, "q3": 0, "q4": 0}
	previousCompleted := 0
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
		if governanceComparePrevious && !completedTime.Before(prevStart) && completedTime.Before(start) {
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

	badges := make([]string, 0)
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
	nextActions := make([]string, 0)
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

	printGovernanceResult(map[string]interface{}{
		"metrics": map[string]interface{}{
			"window_days":           windowDays,
			"completed_count":       completedCount,
			"active_count":          activeCount,
			"on_time_rate":          onTimeRate,
			"streak_days":           streak,
			"avg_completed_per_day": avgPerDay,
			"overdue_fixed_count":   overdueFixedCount,
			"quadrant_completed":    completedByQuadrant,
			"compare_previous":      governanceComparePrevious,
			"previous_completed":    previousCompleted,
			"delta_completed":       delta,
			"trend":                 trendText,
		},
		"badges":       badges,
		"narrative":    narrative,
		"next_actions": nextActions,
	})
}

func effectiveIntelligenceConfig() pkgconfig.IntelligenceConfig {
	defaults := pkgconfig.DefaultConfig().Intelligence
	out := cfg.Intelligence
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
	if out.LongTerm.RetainCountWhenOverflow == 0 {
		out.LongTerm.RetainCountWhenOverflow = defaults.LongTerm.RetainCountWhenOverflow
	}
	if strings.TrimSpace(out.LongTerm.OverflowStrategy) == "" {
		out.LongTerm.OverflowStrategy = defaults.LongTerm.OverflowStrategy
	}
	if out.Decompose.ComplexityThreshold == 0 {
		out.Decompose.ComplexityThreshold = defaults.Decompose.ComplexityThreshold
	}
	if strings.TrimSpace(out.Decompose.PreferredStrategy) == "" {
		out.Decompose.PreferredStrategy = defaults.Decompose.PreferredStrategy
	}
	if len(out.Decompose.AbstractKeywords) == 0 {
		out.Decompose.AbstractKeywords = append([]string(nil), defaults.Decompose.AbstractKeywords...)
	}
	if strings.TrimSpace(out.Achievement.SnapshotGranularity) == "" {
		out.Achievement.SnapshotGranularity = defaults.Achievement.SnapshotGranularity
	}
	if out.Achievement.StreakGoalPerDay == 0 {
		out.Achievement.StreakGoalPerDay = defaults.Achievement.StreakGoalPerDay
	}
	return out
}

func parseOverdueAction(raw string) (overdueActionItem, error) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) < 2 {
		return overdueActionItem{}, fmt.Errorf("invalid action %q", raw)
	}
	item := overdueActionItem{
		TaskID: strings.TrimSpace(parts[0]),
		Type:   strings.TrimSpace(parts[1]),
	}
	if len(parts) > 2 {
		item.DueDate = strings.TrimSpace(parts[2])
	}
	if item.TaskID == "" || item.Type == "" {
		return overdueActionItem{}, fmt.Errorf("invalid action %q", raw)
	}
	return item, nil
}

func computeTaskComplexity(task model.Task, cfg pkgconfig.DecomposeConfig) (int, []string) {
	score := 0
	reasons := make([]string, 0)
	title := strings.TrimSpace(task.Title)
	description := strings.TrimSpace(task.Description)
	fullText := strings.ToLower(title + " " + description)
	if len([]rune(title)) >= 20 {
		score += 10
		reasons = append(reasons, "long_title")
	}
	if len([]rune(title)) >= 36 {
		score += 10
		reasons = append(reasons, "very_long_title")
	}
	if len([]rune(description)) >= 80 {
		score += 10
		reasons = append(reasons, "large_description")
	}
	if cfg.DetectAbstractKeywords {
		matchedKeywords := make([]string, 0)
		for _, keyword := range cfg.AbstractKeywords {
			k := strings.ToLower(strings.TrimSpace(keyword))
			if k != "" && strings.Contains(fullText, k) {
				matchedKeywords = append(matchedKeywords, keyword)
			}
		}
		if len(matchedKeywords) > 0 {
			score += minInt(30, 10+len(matchedKeywords)*5)
			reasons = append(reasons, "abstract_keyword")
		}
	}
	if task.DueDate == nil {
		score += 10
		reasons = append(reasons, "no_due_date")
	}
	if task.EstimatedMinutes <= 0 {
		score += 5
		reasons = append(reasons, "no_estimate")
	}
	if task.ParentID == nil && len(task.SubtaskIDs) == 0 {
		score += 15
		reasons = append(reasons, "no_subtasks")
	}
	if overdueCount := taskCustomInt(task, "tb_overdue_count"); overdueCount > 0 {
		score += minInt(15, overdueCount*3)
		reasons = append(reasons, "historical_overdue")
	}
	if score > 100 {
		score = 100
	}
	return score, reasons
}

func recommendProviderForTask(providers map[string]provider.Provider, task model.Task, preferred string) (string, bool) {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		resolved := provider.ResolveProviderName(preferred)
		if p, ok := providers[resolved]; ok {
			return resolved, p.Capabilities().SupportsSubtasks
		}
		if provider.IsValidProvider(resolved) {
			return resolved, false
		}
	}
	taskSource := strings.TrimSpace(string(task.Source))
	if provider.IsValidProvider(taskSource) {
		if p, ok := providers[taskSource]; ok {
			return taskSource, p.Capabilities().SupportsSubtasks
		}
		return taskSource, false
	}
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if providers[name].Capabilities().SupportsSubtasks {
			return name, true
		}
	}
	if len(names) > 0 {
		return names[0], providers[names[0]].Capabilities().SupportsSubtasks
	}
	return "local", false
}

func buildDecomposePreview(task model.Task, supportsSubtasks bool) []map[string]interface{} {
	baseTitle := strings.TrimSpace(task.Title)
	if baseTitle == "" {
		baseTitle = "任务"
	}
	steps := []struct {
		Title    string
		Phase    string
		Offset   int
		Priority int
		Quadrant int
	}{
		{Title: "明确范围：" + baseTitle, Phase: "规划", Offset: 1, Priority: maxInt(2, int(task.Priority)), Quadrant: 2},
		{Title: "执行核心工作：" + baseTitle, Phase: "执行", Offset: 2, Priority: maxInt(2, int(task.Priority)), Quadrant: int(task.Quadrant)},
		{Title: "验证与修正：" + baseTitle, Phase: "验证", Offset: 3, Priority: 2, Quadrant: 2},
		{Title: "总结与归档：" + baseTitle, Phase: "收尾", Offset: 4, Priority: 1, Quadrant: 2},
	}
	result := make([]map[string]interface{}, 0, len(steps))
	for i, step := range steps {
		item := map[string]interface{}{
			"id":              fmt.Sprintf("preview_%d", i+1),
			"title":           step.Title,
			"description":     fmt.Sprintf("由任务“%s”拆分而来", baseTitle),
			"phase":           step.Phase,
			"due_offset_days": step.Offset,
			"priority":        clampTaskPriority(step.Priority),
			"quadrant":        clampTaskQuadrant(step.Quadrant),
		}
		if supportsSubtasks && i > 0 {
			item["parent_preview_id"] = "preview_1"
		}
		result = append(result, item)
	}
	return result
}

func writeDecomposePreviewTasks(ctx context.Context, taskStore storage.Storage, parentTask model.Task, providerName, strategy, planID string, preview []map[string]interface{}) ([]string, error) {
	now := time.Now()
	createdIDs := make([]string, 0, len(preview))
	previewToLocal := make(map[string]string, len(preview))
	for idx, item := range preview {
		title := strings.TrimSpace(fmt.Sprint(item["title"]))
		if title == "" {
			title = fmt.Sprintf("拆分任务 %d", idx+1)
		}
		desc := strings.TrimSpace(fmt.Sprint(item["description"]))
		phase := strings.TrimSpace(fmt.Sprint(item["phase"]))
		previewID := strings.TrimSpace(fmt.Sprint(item["id"]))
		parentPreviewID := strings.TrimSpace(fmt.Sprint(item["parent_preview_id"]))
		dueOffset := anyToInt(item["due_offset_days"])
		if dueOffset <= 0 {
			dueOffset = idx + 1
		}
		priority := clampTaskPriority(anyToInt(item["priority"]))
		quadrant := clampTaskQuadrant(anyToInt(item["quadrant"]))

		task := &model.Task{
			ID:          generateID(),
			Title:       title,
			Description: desc,
			Status:      model.StatusTodo,
			CreatedAt:   now,
			UpdatedAt:   now,
			Source:      model.SourceLocal,
			Priority:    priority,
			Quadrant:    quadrant,
		}
		due := now.AddDate(0, 0, dueOffset)
		task.DueDate = &due
		if parentPreviewID != "" {
			if localParentID := strings.TrimSpace(previewToLocal[parentPreviewID]); localParentID != "" {
				parentID := localParentID
				task.ParentID = &parentID
			}
		}
		task.Metadata = &model.TaskMetadata{
			Version:    "1.0",
			Quadrant:   int(task.Quadrant),
			Priority:   int(task.Priority),
			LocalID:    task.ID,
			SyncSource: "local",
			CustomFields: map[string]interface{}{
				"tb_parent_task_id":     parentTask.ID,
				"tb_decompose_plan_id":  planID,
				"tb_decompose_provider": providerName,
				"tb_decompose_strategy": strategy,
				"tb_phase":              phase,
				"tb_preview_id":         previewID,
			},
		}
		if err := taskStore.SaveTask(ctx, task); err != nil {
			return nil, err
		}
		createdIDs = append(createdIDs, task.ID)
		if previewID != "" {
			previewToLocal[previewID] = task.ID
		}
	}
	return createdIDs, nil
}

func resolveLocation(tz string) *time.Location {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Local
	}
	return loc
}

func completionTime(task model.Task) *time.Time {
	if task.CompletedAt != nil {
		return task.CompletedAt
	}
	if task.Status != model.StatusCompleted {
		return nil
	}
	return &task.UpdatedAt
}

func calcCompletionStreak(daily map[string]int, now time.Time, goalPerDay int) int {
	streak := 0
	for day := 0; day < 365; day++ {
		dateKey := now.AddDate(0, 0, -day).Format("2006-01-02")
		if daily[dateKey] < goalPerDay {
			break
		}
		streak++
	}
	return streak
}

func buildOverdueQuestions(overdueCount int, overload bool) []string {
	if overdueCount == 0 {
		return []string{"当前无逾期任务，是否要提前规划下周重点任务？"}
	}
	questions := []string{
		"这些逾期任务是否都属于延期任务？",
		"请确认需要延期的任务 ID 列表（可批量）。",
	}
	if overload {
		questions = append(questions,
			"当前逾期数量较多，是否删除低价值且长期逾期的任务？",
			"是否将复杂逾期任务先拆分为子任务再重排日期？",
		)
	}
	return questions
}

func calcOverdueDays(due *time.Time, now time.Time) int {
	if due == nil || !due.Before(now) {
		return 0
	}
	days := int(now.Sub(*due).Hours() / 24)
	if days <= 0 {
		return 1
	}
	return days
}

func calcAgeDays(createdAt, now time.Time) int {
	if createdAt.IsZero() {
		return 0
	}
	days := int(now.Sub(createdAt).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func scoreLongTermTask(task model.Task, now time.Time) int {
	score := int(task.Priority) * 100
	switch task.Quadrant {
	case model.QuadrantUrgentImportant:
		score += 40
	case model.QuadrantNotUrgentImportant:
		score += 30
	case model.QuadrantUrgentNotImportant:
		score += 10
	}
	score += minInt(30, calcAgeDays(task.CreatedAt, now))
	return score
}

func taskCustomInt(task model.Task, key string) int {
	if task.Metadata == nil || task.Metadata.CustomFields == nil {
		return 0
	}
	raw, ok := task.Metadata.CustomFields[key]
	if !ok {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}

func anyToInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(v))
		return n
	default:
		return 0
	}
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func printGovernanceResult(value interface{}) {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		exitErr("序列化输出失败", err)
	}
	fmt.Println(string(bytes))
}
