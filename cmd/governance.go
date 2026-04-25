package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

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
	Short: "任务治理与智能辅助",
	Long: `运行 TaskBridge 的 CLI 治理能力，包括逾期健康分析、长期任务调配、
	复杂任务识别、任务拆分建议和成就分析。`,
}

var governanceOverdueHealthCmd = &cobra.Command{
	Use:   "overdue-health",
	Short: "分析逾期任务健康度",
	RunE:  runGovernanceOverdueHealth,
}

var governanceResolveOverdueCmd = &cobra.Command{
	Use:   "resolve-overdue",
	Short: "批量处理逾期任务",
	RunE:  runGovernanceResolveOverdue,
}

var governanceRebalanceLongTermCmd = &cobra.Command{
	Use:   "rebalance-longterm",
	Short: "调配长期无排期任务",
	RunE:  runGovernanceRebalanceLongTerm,
}

var governanceDetectDecompositionCmd = &cobra.Command{
	Use:   "detect-decomposition",
	Short: "识别复杂且缺少子任务的候选任务",
	RunE:  runGovernanceDetectDecomposition,
}

var governanceDecomposeTaskCmd = &cobra.Command{
	Use:   "decompose-task <task-id>",
	Short: "将单个任务拆分为执行步骤",
	Args:  cobra.ExactArgs(1),
	RunE:  runGovernanceDecomposeTask,
}

var governanceAchievementCmd = &cobra.Command{
	Use:   "achievement",
	Short: "分析完成情况与成就反馈",
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

func runGovernanceOverdueHealth(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	result, err := (&govsvc.Service{TaskStore: taskStore, Config: cfg.Intelligence}).OverdueHealth(ctx, govsvc.OverdueHealthOptions{
		Filter:             govsvc.FilterOptions{Source: governanceSource, ListIDs: governanceListIDs},
		IncludeSuggestions: governanceIncludeSuggestions,
	})
	if err != nil {
		return commandError("逾期分析失败", err)
	}
	return printResult(result)
}

func runGovernanceResolveOverdue(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	if len(governanceActionItems) == 0 {
		return usageError("请至少传入一个 --action taskID:type[:due_date]")
	}
	result, err := (&govsvc.Service{TaskStore: taskStore, Config: cfg.Intelligence}).ResolveOverdue(ctx, govsvc.ResolveOverdueOptions{
		ActionItems:   governanceActionItems,
		DryRun:        governanceDryRun,
		ConfirmDelete: governanceConfirmDelete,
	})
	if err != nil {
		return commandError("处理逾期任务失败", err)
	}
	return printResult(result)
}

func runGovernanceRebalanceLongTerm(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	result, err := (&govsvc.Service{TaskStore: taskStore, Config: cfg.Intelligence}).RebalanceLongTerm(ctx, govsvc.RebalanceLongTermOptions{
		Filter: govsvc.FilterOptions{Source: governanceSource, ListIDs: governanceListIDs},
		DryRun: governanceDryRun,
	})
	if err != nil {
		return commandError("长期任务调配失败", err)
	}
	return printResult(result)
}

func runGovernanceDetectDecomposition(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	providers, _ := loadAuthenticatedProviders("")
	result, err := (&govsvc.Service{TaskStore: taskStore, Providers: providers, Config: cfg.Intelligence}).DetectDecomposition(ctx, govsvc.DetectDecompositionOptions{
		Filter: govsvc.FilterOptions{Source: governanceSource, ListIDs: governanceListIDs},
		Limit:  governanceLimit,
	})
	if err != nil {
		return commandError("识别复杂任务失败", err)
	}
	return printResult(result)
}

func runGovernanceDecomposeTask(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	providers, _ := loadAuthenticatedProviders("")
	result, err := (&govsvc.Service{TaskStore: taskStore, Providers: providers, Config: cfg.Intelligence}).DecomposeTask(ctx, strings.TrimSpace(args[0]), govsvc.DecomposeTaskOptions{
		Provider:   governanceProvider,
		Strategy:   governanceStrategy,
		WriteTasks: governanceWriteTasks,
	})
	if err != nil {
		return commandError("拆分任务失败", err)
	}
	return printResult(result)
}

func runGovernanceAchievement(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	taskStore, _, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	result, err := (&govsvc.Service{TaskStore: taskStore, Config: cfg.Intelligence}).Achievement(ctx, govsvc.AchievementOptions{
		WindowDays:      governanceWindowDays,
		ComparePrevious: governanceComparePrevious,
	})
	if err != nil {
		return commandError("成就分析失败", err)
	}
	return printResult(result)
}
