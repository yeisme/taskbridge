package cmd

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/pkg/ui"

	govsvc "github.com/yeisme/taskbridge/internal/governance"
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
	Short: "Task management and intelligent assistance",
	Long: `Run TaskBridge CLI governance capabilities, including overdue health analysis, long-term task scheduling,
	complex task identification, task splitting suggestions, and achievement analysis.`,
}

var governanceOverdueHealthCmd = &cobra.Command{
	Use:   "overdue-health",
	Short: "Analyze the health of overdue tasks",
	RunE:  runGovernanceOverdueHealth,
}

var governanceResolveOverdueCmd = &cobra.Command{
	Use:   "resolve-overdue",
	Short: "Process overdue tasks in batches",
	RunE:  runGovernanceResolveOverdue,
}

var governanceRebalanceLongTermCmd = &cobra.Command{
	Use:   "rebalance-longterm",
	Short: "Deploy long-term unscheduled tasks",
	RunE:  runGovernanceRebalanceLongTerm,
}

var governanceDetectDecompositionCmd = &cobra.Command{
	Use:   "detect-decomposition",
	Short: "Identify candidate tasks that are complex and missing subtasks",
	RunE:  runGovernanceDetectDecomposition,
}

var governanceDecomposeTaskCmd = &cobra.Command{
	Use:   "decompose-task <task-id>",
	Short: "Split a single task into execution steps",
	Args:  cobra.ExactArgs(1),
	RunE:  runGovernanceDecomposeTask,
}

var governanceAchievementCmd = &cobra.Command{
	Use:   "achievement",
	Short: "Analyze completion status and achievement feedback",
	RunE:  runGovernanceAchievement,
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
		cmd.Flags().StringVarP(&governanceFormat, "format", "f", "table", "Output format (table, json, text)")
	}

	for _, cmd := range []*cobra.Command{
		governanceOverdueHealthCmd,
		governanceRebalanceLongTermCmd,
		governanceDetectDecompositionCmd,
	} {
		cmd.Flags().StringVar(&governanceSource, "source", "", "Filter by source (abbreviations supported)")
		cmd.Flags().StringSliceVar(&governanceListIDs, "list-id", nil, "Filter by list ID (repeatable)")
	}

	governanceOverdueHealthCmd.Flags().BoolVar(&governanceIncludeSuggestions, "include-suggestions", true, "Return suggested actions and follow-up questions")

	governanceResolveOverdueCmd.Flags().StringSliceVar(&governanceActionItems, "action", nil, "Overdue action taskID:type[:due_date], repeatable")
	governanceResolveOverdueCmd.Flags().BoolVar(&governanceDryRun, "dry-run", false, "Simulate execution")
	governanceResolveOverdueCmd.Flags().BoolVar(&governanceConfirmDelete, "confirm-delete", false, "Confirm delete actions")

	governanceRebalanceLongTermCmd.Flags().BoolVar(&governanceDryRun, "dry-run", false, "Simulate execution")

	governanceDetectDecompositionCmd.Flags().IntVar(&governanceLimit, "limit", 20, "Return the number of items")

	governanceDecomposeTaskCmd.Flags().StringVar(&governanceProvider, "provider", "", "Preferred Provider")
	governanceDecomposeTaskCmd.Flags().StringVar(&governanceStrategy, "strategy", "", "Split strategy")
	governanceDecomposeTaskCmd.Flags().BoolVar(&governanceWriteTasks, "write-tasks", false, "Write split results to local tasks")

	governanceAchievementCmd.Flags().IntVar(&governanceWindowDays, "window-days", 30, "Statistics window days")
	governanceAchievementCmd.Flags().BoolVar(&governanceComparePrevious, "compare-previous", true, "Compare with the previous cycle")
}

func runGovernanceOverdueHealth(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	result, err := (&govsvc.Service{TaskStore: taskStore, Config: cfg.Intelligence}).OverdueHealth(ctx, govsvc.OverdueHealthOptions{
		Filter:             govsvc.FilterOptions{Source: governanceSource, ListIDs: governanceListIDs},
		IncludeSuggestions: governanceIncludeSuggestions,
	})
	if err != nil {
		return commandError("Overdue analysis failed", err)
	}
	return printProjectionWithLegacyJSON(governanceFormat, result, buildGovernanceOverdueHealthProjection(result), func() {
		fmt.Print(renderGovernanceOverdueHealth(result, buildGovernanceOverdueHealthProjection(result)))
	})
}

func runGovernanceResolveOverdue(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	if len(governanceActionItems) == 0 {
		return usageError("Pass at least one --action taskID:type[:due_date]")
	}
	result, err := (&govsvc.Service{TaskStore: taskStore, Config: cfg.Intelligence}).ResolveOverdue(ctx, govsvc.ResolveOverdueOptions{
		ActionItems:   governanceActionItems,
		DryRun:        governanceDryRun,
		ConfirmDelete: governanceConfirmDelete,
	})
	if err != nil {
		return commandError("Failed to process overdue tasks", err)
	}
	return printProjectionWithLegacyJSON(governanceFormat, result, buildGovernanceResolveOverdueProjection(result), func() { fmt.Print(renderGovernanceProjection(buildGovernanceResolveOverdueProjection(result))) })
}

func runGovernanceRebalanceLongTerm(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	result, err := (&govsvc.Service{TaskStore: taskStore, Config: cfg.Intelligence}).RebalanceLongTerm(ctx, govsvc.RebalanceLongTermOptions{
		Filter: govsvc.FilterOptions{Source: governanceSource, ListIDs: governanceListIDs},
		DryRun: governanceDryRun,
	})
	if err != nil {
		return commandError("Long-term task deployment failed", err)
	}
	return printProjectionWithLegacyJSON(governanceFormat, result, buildGovernanceRebalanceProjection(result), func() { fmt.Print(renderGovernanceProjection(buildGovernanceRebalanceProjection(result))) })
}

func runGovernanceDetectDecomposition(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	providers, _ := loadAuthenticatedProviders("")
	result, err := (&govsvc.Service{TaskStore: taskStore, Providers: providers, Config: cfg.Intelligence}).DetectDecomposition(ctx, govsvc.DetectDecompositionOptions{
		Filter: govsvc.FilterOptions{Source: governanceSource, ListIDs: governanceListIDs},
		Limit:  governanceLimit,
	})
	if err != nil {
		return commandError("Identify complex task failures", err)
	}
	return printProjectionWithLegacyJSON(governanceFormat, result, buildGovernanceDetectProjection(result), func() { fmt.Print(renderGovernanceProjection(buildGovernanceDetectProjection(result))) })
}

func runGovernanceDecomposeTask(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	providers, _ := loadAuthenticatedProviders("")
	result, err := (&govsvc.Service{TaskStore: taskStore, Providers: providers, Config: cfg.Intelligence}).DecomposeTask(ctx, strings.TrimSpace(args[0]), govsvc.DecomposeTaskOptions{
		Provider:   governanceProvider,
		Strategy:   governanceStrategy,
		WriteTasks: governanceWriteTasks,
	})
	if err != nil {
		return commandError("Split task failed", err)
	}
	return printProjectionWithLegacyJSON(governanceFormat, result, buildGovernanceDecomposeProjection(result), func() { fmt.Print(renderGovernanceProjection(buildGovernanceDecomposeProjection(result))) })
}

func runGovernanceAchievement(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	result, err := (&govsvc.Service{TaskStore: taskStore, Config: cfg.Intelligence}).Achievement(ctx, govsvc.AchievementOptions{
		WindowDays:      governanceWindowDays,
		ComparePrevious: governanceComparePrevious,
	})
	if err != nil {
		return commandError("Achievement analysis failed", err)
	}
	return printProjectionWithLegacyJSON(governanceFormat, result, buildGovernanceAchievementProjection(result), func() { fmt.Print(renderGovernanceProjection(buildGovernanceAchievementProjection(result))) })
}

func buildGovernanceOverdueHealthProjection(result map[string]interface{}) clioutput.Projection {
	p := clioutput.New("governance.overdue_health")
	p.Summary = "Overdue health analyzed."
	if summary, ok := result["summary"].(map[string]interface{}); ok {
		copyProjectionFacts(p.Facts, summary, "overdue_count", "severe_overdue_count", "is_warning", "is_overload")
		if value, ok := summary["is_warning"].(bool); ok && value {
			p.Status = clioutput.StatusPartial
		}
		if value, ok := summary["is_overload"].(bool); ok && value {
			p.Status = clioutput.StatusPartial
			p.Risks = append(p.Risks, "Overdue workload exceeds the configured overload threshold.")
		}
	}
	p.Data = result
	return p
}

func buildGovernanceResolveOverdueProjection(result map[string]interface{}) clioutput.Projection {
	p := clioutput.New("governance.resolve_overdue")
	p.Summary = "Overdue actions processed."
	copyProjectionFacts(p.Facts, result, "total", "updated", "deferred", "rescheduled", "deleted", "split_suggested", "skipped", "dry_run", "requires_confirm", "confirm_token_match")
	if errors, ok := result["errors"].([]string); ok && len(errors) > 0 {
		p.Status = clioutput.StatusPartial
		p.Risks = append(p.Risks, errors...)
	}
	p.Data = result
	return p
}

func buildGovernanceRebalanceProjection(result map[string]interface{}) clioutput.Projection {
	p := clioutput.New("governance.rebalance_longterm")
	p.Summary = "Long-term task rebalance completed."
	copyProjectionFacts(p.Facts, result, "mode", "short_term_count", "long_term_count", "dry_run")
	p.Facts["promoted"] = stringSliceLen(result["promoted_task_ids"])
	p.Facts["retained"] = stringSliceLen(result["retained_task_ids"])
	p.Facts["adjusted"] = stringSliceLen(result["adjusted_task_ids"])
	p.Data = result
	return p
}

func buildGovernanceDetectProjection(result map[string]interface{}) clioutput.Projection {
	p := clioutput.New("governance.detect_decomposition")
	p.Summary = "Decomposition candidates detected."
	if summary, ok := result["summary"].(map[string]interface{}); ok {
		copyProjectionFacts(p.Facts, summary, "total_scanned", "candidate_count", "threshold", "limit")
	}
	p.Data = result
	return p
}

func buildGovernanceDecomposeProjection(result map[string]interface{}) clioutput.Projection {
	p := clioutput.New("governance.decompose_task")
	p.Summary = "Task decomposition preview generated."
	copyProjectionFacts(p.Facts, result, "task_id", "plan_id", "provider", "strategy", "write_tasks")
	p.Facts["created_tasks"] = stringSliceLen(result["created_task_ids"])
	if warnings, ok := result["warnings"].([]string); ok && len(warnings) > 0 {
		p.Status = clioutput.StatusPartial
		p.Risks = append(p.Risks, warnings...)
	}
	p.Data = result
	return p
}

func buildGovernanceAchievementProjection(result map[string]interface{}) clioutput.Projection {
	p := clioutput.New("governance.achievement")
	p.Summary = "Achievement analysis completed."
	if metrics, ok := result["metrics"].(map[string]interface{}); ok {
		copyProjectionFacts(p.Facts, metrics, "window_days", "completed_count", "active_count", "on_time_rate", "streak_days", "avg_completed_per_day", "overdue_fixed_count", "compare_previous", "previous_completed", "delta_completed", "trend")
	}
	if nextActions, ok := result["next_actions"].([]string); ok {
		p.Risks = append(p.Risks, nextActions...)
	}
	p.Data = result
	return p
}

func renderGovernanceOverdueHealth(result map[string]interface{}, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("📊 Overdue Health\n\n")

	summary := mapValue(result["summary"])
	summaryTable := ui.NewTable("Metric", "Value")
	summaryTable.AddRow("Overdue", fmt.Sprint(summary["overdue_count"]))
	summaryTable.AddRow("Severe", fmt.Sprint(summary["severe_overdue_count"]))
	summaryTable.AddRow("Warning", boolStatus(summary["is_warning"]))
	summaryTable.AddRow("Overload", boolStatus(summary["is_overload"]))
	b.WriteString("Summary\n")
	b.WriteString(summaryTable.Render())
	b.WriteString("\n\n")

	config := mapValue(result["config_applied"])
	if len(config) > 0 {
		configTable := ui.NewTable("Config", "Value")
		for _, key := range sortedKeys(config) {
			configTable.AddRow(key, fmt.Sprint(config[key]))
		}
		b.WriteString("Applied config\n")
		b.WriteString(configTable.Render())
		b.WriteString("\n\n")
	}

	candidates := candidateRows(result["candidates"])
	b.WriteString("Candidates\n")
	if len(candidates) == 0 {
		b.WriteString("No overdue tasks found.\n\n")
	} else {
		table := ui.NewTable("Task", "Title", "Days", "Priority", "Source")
		for _, row := range candidates {
			table.AddRow(row["task_id"], row["title"], row["days_overdue"], row["priority"], row["source"])
		}
		b.WriteString(table.Render())
		b.WriteString("\n\n")
	}

	if actions := stringList(result["actions"]); len(actions) > 0 {
		actionTable := ui.NewTable("Recommended actions")
		for _, action := range actions {
			actionTable.AddRow(action)
		}
		b.WriteString("Recommended actions\n")
		b.WriteString(actionTable.Render())
		b.WriteString("\n\n")
	}

	if questions := stringList(result["questions"]); len(questions) > 0 {
		questionTable := ui.NewTable("Follow-up questions")
		for _, question := range questions {
			questionTable.AddRow(question)
		}
		b.WriteString("Follow-up questions\n")
		b.WriteString(questionTable.Render())
		b.WriteString("\n\n")
	}

	if len(projection.Actions) > 0 {
		b.WriteString("Recommended next step\n")
		b.WriteString(projection.Actions[0].Command)
		b.WriteString("\n")
	}
	return b.String()
}

func renderGovernanceProjection(projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString(projection.Summary)
	b.WriteString("\n\n")
	if len(projection.Facts) > 0 {
		table := ui.NewTable("Fact", "Value")
		for _, key := range sortedAnyKeys(projection.Facts) {
			table.AddRow(key, fmt.Sprint(projection.Facts[key]))
		}
		b.WriteString(table.Render())
		b.WriteString("\n")
	}
	if len(projection.Risks) > 0 {
		b.WriteString("\nRisks\n")
		for _, risk := range projection.Risks {
			b.WriteString("- ")
			b.WriteString(risk)
			b.WriteString("\n")
		}
	}
	if len(projection.Actions) > 0 {
		b.WriteString("\nRecommended next step\n")
		b.WriteString(projection.Actions[0].Command)
		b.WriteString("\n")
	}
	return b.String()
}

func boolStatus(value interface{}) string {
	if v, ok := value.(bool); ok && v {
		return "⚠️ yes"
	}
	return "✅ no"
}

func sortedKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedAnyKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mapValue(value interface{}) map[string]interface{} {
	if typed, ok := value.(map[string]interface{}); ok {
		return typed
	}
	return map[string]interface{}{}
}

func stringList(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmt.Sprint(item))
		}
		return items
	default:
		return nil
	}
}

func candidateRows(value interface{}) []map[string]string {
	v := reflect.ValueOf(value)
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return nil
	}
	rows := make([]map[string]string, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Interface {
			item = item.Elem()
		}
		row := map[string]string{
			"task_id":      fieldString(item, "TaskID"),
			"title":        fieldString(item, "Title"),
			"days_overdue": fieldString(item, "DaysOverdue"),
			"priority":     fieldString(item, "Priority"),
			"source":       fieldString(item, "Source"),
		}
		if row["task_id"] != "" || row["title"] != "" {
			rows = append(rows, row)
		}
	}
	return rows
}

func fieldString(item reflect.Value, name string) string {
	if !item.IsValid() || item.Kind() != reflect.Struct {
		return ""
	}
	field := item.FieldByName(name)
	if !field.IsValid() {
		return ""
	}
	if name == "DueDate" && !field.IsNil() {
		if due, ok := field.Interface().(*time.Time); ok {
			return due.Format("2006-01-02")
		}
	}
	return fmt.Sprint(field.Interface())
}

func copyProjectionFacts(dst map[string]any, src map[string]interface{}, keys ...string) {
	for _, key := range keys {
		if value, ok := src[key]; ok {
			dst[key] = value
		}
	}
}

func stringSliceLen(value interface{}) int {
	items, ok := value.([]string)
	if !ok {
		return 0
	}
	return len(items)
}
