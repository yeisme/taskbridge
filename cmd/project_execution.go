package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/actionfile"
	"github.com/yeisme/taskbridge/internal/projectservice"
)

var (
	projectAdjustReason  string
	projectAdjustDryRun  bool
	projectAdjustConfirm bool
)

var projectReviewCmd = &cobra.Command{
	Use:   "review <project-id>",
	Short: "复盘项目执行状态",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectReview,
}

var projectNextCmd = &cobra.Command{
	Use:   "next <project-id>",
	Short: "输出项目下一步",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectNext,
}

var projectAdjustCmd = &cobra.Command{
	Use:   "adjust <project-id>",
	Short: "根据执行状态生成或应用项目调整",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectAdjust,
}

var projectDoneCmd = &cobra.Command{
	Use:   "done <project-id>",
	Short: "标记项目完成",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectDone,
}

var projectArchiveCmd = &cobra.Command{
	Use:   "archive <project-id>",
	Short: "归档项目",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectArchive,
}

func init() {
	projectCmd.AddCommand(projectReviewCmd)
	projectCmd.AddCommand(projectNextCmd)
	projectCmd.AddCommand(projectAdjustCmd)
	projectCmd.AddCommand(projectDoneCmd)
	projectCmd.AddCommand(projectArchiveCmd)
	for _, cmd := range []*cobra.Command{projectReviewCmd, projectNextCmd, projectAdjustCmd, projectDoneCmd, projectArchiveCmd} {
		cmd.Flags().StringVarP(&projectFormat, "format", "f", "text", "输出格式 (text, json)")
	}
	projectAdjustCmd.Flags().StringVar(&projectAdjustReason, "reason", "", "调整原因")
	projectAdjustCmd.Flags().BoolVar(&projectAdjustDryRun, "dry-run", true, "模拟执行")
	projectAdjustCmd.Flags().BoolVar(&projectAdjustConfirm, "confirm", false, "确认应用调整")
}

func projectExecutionService() (*projectservice.ExecutionService, func(), error) {
	taskStore, projectStore, cleanup, err := getCLIStores()
	if err != nil {
		return nil, cleanup, err
	}
	return &projectservice.ExecutionService{TaskStore: taskStore, ProjectStore: projectStore}, cleanup, nil
}

func runProjectReview(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	defer cleanup()
	if err != nil {
		return commandError("初始化项目服务失败", err)
	}
	result, err := service.Review(context.Background(), args[0])
	if err != nil {
		return commandError("项目复盘失败", err)
	}
	return printStructured(projectFormat, result, func() {
		fmt.Printf("项目复盘: %s\n", result.ProjectName)
		fmt.Printf("状态: %s 下一步: %s\n", result.Status, result.NextTaskID)
		fmt.Printf("风险: %v\n", result.Risk)
	})
}

func runProjectNext(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	defer cleanup()
	if err != nil {
		return commandError("初始化项目服务失败", err)
	}
	result, err := service.Next(context.Background(), args[0])
	if err != nil {
		return commandError("项目下一步失败", err)
	}
	return printStructured(projectFormat, result, func() {
		fmt.Printf("项目下一步: %v\n", result["next_task_id"])
	})
}

func runProjectAdjust(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	defer cleanup()
	if err != nil {
		return commandError("初始化项目服务失败", err)
	}
	actions, err := service.Adjust(context.Background(), args[0], projectAdjustReason)
	if err != nil {
		return commandError("生成项目调整失败", err)
	}
	if projectAdjustConfirm {
		result := actionfile.Executor{TaskStore: service.TaskStore}.Execute(context.Background(), actions, actionfile.ExecuteOptions{DryRun: false, Confirm: true})
		return printStructured(projectFormat, result, func() {
			fmt.Printf("项目调整已应用: updated=%d skipped=%d\n", result.Updated, result.Skipped)
		})
	}
	result := map[string]interface{}{"schema": "taskbridge.project-adjust.v1", "dry_run": projectAdjustDryRun, "requires_confirmation": len(actions.Actions) > 0, "actions": actions.Actions}
	return printStructured(projectFormat, result, func() {
		fmt.Printf("项目调整预览: %d action(s)\n", len(actions.Actions))
		if len(actions.Actions) > 0 {
			fmt.Println("应用调整请重新运行并加入 --confirm")
		}
	})
}

func runProjectDone(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	defer cleanup()
	if err != nil {
		return commandError("初始化项目服务失败", err)
	}
	result, err := service.Done(context.Background(), args[0])
	if err != nil {
		return commandError("标记项目完成失败", err)
	}
	return printStructured(projectFormat, result, func() {
		fmt.Printf("项目已完成: %s\n", args[0])
	})
}

func runProjectArchive(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	defer cleanup()
	if err != nil {
		return commandError("初始化项目服务失败", err)
	}
	result, err := service.Archive(context.Background(), args[0])
	if err != nil {
		return commandError("归档项目失败", err)
	}
	return printStructured(projectFormat, result, func() {
		fmt.Printf("项目已归档: %s\n", args[0])
	})
}
