package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/actionfile"
	"github.com/yeisme/taskbridge/internal/controlplane"
)

var (
	controlFormat   string
	controlSource   string
	controlLimit    int
	reviewApplyFile string
	reviewDryRun    bool
	reviewConfirm   bool
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "每日任务工作台",
	RunE:  runToday,
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "推荐当前下一步任务",
	RunE:  runNext,
}

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "列出待整理任务",
	RunE:  runInbox,
}

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "任务健康复盘",
	RunE:  runReview,
}

func init() {
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(inboxCmd)
	rootCmd.AddCommand(reviewCmd)

	for _, cmd := range []*cobra.Command{todayCmd, nextCmd, inboxCmd, reviewCmd} {
		cmd.Flags().StringVarP(&controlFormat, "format", "f", "text", "输出格式 (text, json)")
		cmd.Flags().StringVar(&controlSource, "source", "", "按来源筛选（支持简写）")
	}
	for _, cmd := range []*cobra.Command{nextCmd, inboxCmd} {
		cmd.Flags().IntVar(&controlLimit, "limit", 0, "最多返回任务数")
	}
	reviewCmd.Flags().StringVar(&reviewApplyFile, "apply-file", "", "执行结构化 action file")
	reviewCmd.Flags().BoolVar(&reviewDryRun, "dry-run", false, "模拟执行 action file")
	reviewCmd.Flags().BoolVar(&reviewConfirm, "confirm", false, "确认执行 action file")
}

func controlService() (*controlplane.Service, func(), error) {
	taskStore, projectStore, cleanup, err := getCLIStores()
	if err != nil {
		return nil, cleanup, err
	}
	return &controlplane.Service{TaskStore: taskStore, ProjectStore: projectStore}, cleanup, nil
}

func runToday(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	service, cleanup, err := controlService()
	defer cleanup()
	if err != nil {
		return commandError("初始化控制面失败", err)
	}
	result, err := service.Today(ctx, controlplane.Options{Source: controlSource})
	if err != nil {
		return commandError("生成今日工作台失败", err)
	}
	return printTodayResult(controlFormat, result)
}

func runNext(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	service, cleanup, err := controlService()
	defer cleanup()
	if err != nil {
		return commandError("初始化控制面失败", err)
	}
	result, err := service.Next(ctx, controlplane.Options{Source: controlSource, Limit: controlLimit})
	if err != nil {
		return commandError("生成下一步失败", err)
	}
	return printTaskListResult(controlFormat, "建议下一步", result)
}

func runInbox(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	service, cleanup, err := controlService()
	defer cleanup()
	if err != nil {
		return commandError("初始化控制面失败", err)
	}
	result, err := service.Inbox(ctx, controlplane.Options{Source: controlSource, Limit: controlLimit})
	if err != nil {
		return commandError("生成 inbox 失败", err)
	}
	return printTaskListResult(controlFormat, "待整理任务", result)
}

func runReview(_ *cobra.Command, _ []string) error {
	if reviewApplyFile != "" {
		return runReviewApplyFile()
	}
	ctx := context.Background()
	service, cleanup, err := controlService()
	defer cleanup()
	if err != nil {
		return commandError("初始化控制面失败", err)
	}
	result, err := service.Review(ctx, controlplane.Options{Source: controlSource})
	if err != nil {
		return commandError("生成复盘失败", err)
	}
	return printStructured(controlFormat, result, func() {
		fmt.Println("任务健康复盘")
		for k, v := range result.Summary {
			fmt.Printf("- %s: %d\n", k, v)
		}
		if len(result.SuggestedActions) == 0 {
			fmt.Println("暂无建议动作。")
			return
		}
		fmt.Println("\n建议动作:")
		for _, action := range result.SuggestedActions {
			fmt.Printf("- %s %s: %s\n", action.Type, action.TaskID, action.Reason)
		}
	})
}

func runReviewApplyFile() error {
	if err := validateReviewApplyMode(reviewDryRun, reviewConfirm); err != nil {
		return err
	}
	actions, err := actionfile.Load(reviewApplyFile)
	if err != nil {
		return commandError("读取 action file 失败", err)
	}
	taskStore, _, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	result := actionfile.Executor{TaskStore: taskStore}.Execute(context.Background(), actions, actionfile.ExecuteOptions{DryRun: reviewDryRun, Confirm: reviewConfirm})
	return printStructured(controlFormat, result, func() {
		fmt.Printf("action file: %s\n", reviewApplyFile)
		fmt.Printf("dry_run: %v\n", result.DryRun)
		fmt.Printf("updated: %d skipped: %d\n", result.Updated, result.Skipped)
		if result.RequiresConfirmation {
			fmt.Println("需要确认：重新运行时加入 --confirm")
		}
		for _, errText := range result.Errors {
			fmt.Printf("- error: %s\n", errText)
		}
	})
}

func validateReviewApplyMode(dryRun, confirm bool) error {
	if dryRun || confirm {
		return nil
	}
	return usageError("执行 --apply-file 时必须显式加入 --dry-run 或 --confirm")
}

func printTodayResult(format string, result *controlplane.TodayResult) error {
	return printStructured(format, result, func() {
		fmt.Printf("TaskBridge Today %s\n", result.Date)
		for _, section := range result.Sections {
			fmt.Printf("\n%s (%d)\n", section.Title, len(section.Tasks))
			for _, task := range section.Tasks {
				fmt.Printf("- [%s] %s", task.Source, task.Title)
				if task.DueDate != nil {
					fmt.Printf(" due=%s", task.DueDate.Format("2006-01-02"))
				}
				fmt.Println()
			}
		}
		if len(result.ProjectNext) > 0 {
			fmt.Println("\n项目下一步")
			for _, item := range result.ProjectNext {
				fmt.Printf("- %s: %s\n", item.ProjectName, item.NextTaskID)
			}
		}
		if len(result.SuggestedActions) > 0 {
			fmt.Println("\n建议动作")
			for _, action := range result.SuggestedActions {
				fmt.Printf("- %s %s: %s\n", action.Type, action.TaskID, action.Reason)
			}
		}
	})
}

func printTaskListResult(format, title string, result *controlplane.ListResult) error {
	return printStructured(format, result, func() {
		fmt.Printf("%s (%d)\n", title, result.Count)
		for _, task := range result.Tasks {
			fmt.Printf("- [%s] %s", task.Source, task.Title)
			if task.DueDate != nil {
				fmt.Printf(" due=%s", task.DueDate.Format("2006-01-02"))
			}
			if task.Reason != "" {
				fmt.Printf(" - %s", task.Reason)
			}
			fmt.Println()
		}
	})
}
