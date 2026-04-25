package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/projectservice"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
)

var (
	projectFormat             string
	projectStatusFilter       string
	projectDescription        string
	projectParentID           string
	projectGoalText           string
	projectHorizonDays        int
	projectListID             string
	projectSource             string
	projectAIHint             string
	projectMaxTasks           int
	projectRequireDeliverable bool
	projectMinEstimateMinutes int
	projectMaxEstimateMinutes int
	projectMinTasks           int
	projectConstraintMaxTasks int
	projectMinPracticeTasks   int
	projectMarkdownFile       string
	projectMarkdownInline     string
	projectPlanID             string
	projectWriteTasks         bool
	projectProvider           string
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "项目规划与落地",
	Long: `管理项目草稿、拆分建议和按项目同步的 CLI 工作流。

子命令:
  create          创建项目草稿
  list            列出项目
  split           生成拆分建议
  split-markdown  从 Markdown 任务树生成拆分建议
  confirm         确认计划并落库任务
  sync            仅同步指定项目的任务

示例:
  taskbridge project create "学习 OpenClaw" --goal-text "我希望学习 openclaw"
  taskbridge project split <project-id> --max-tasks 10
  taskbridge project confirm <project-id> --write-tasks
  taskbridge project sync <project-id> --provider google`,
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "创建项目草稿",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectCreate,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出项目",
	RunE:  runProjectList,
}

var projectSplitCmd = &cobra.Command{
	Use:   "split <project-id>",
	Short: "生成拆分建议",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectSplit,
}

var projectSplitMarkdownCmd = &cobra.Command{
	Use:   "split-markdown <project-id>",
	Short: "从 Markdown 任务树生成拆分建议",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectSplitMarkdown,
}

var projectConfirmCmd = &cobra.Command{
	Use:   "confirm <project-id>",
	Short: "确认项目并落库任务",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectConfirm,
}

var projectSyncCmd = &cobra.Command{
	Use:   "sync <project-id>",
	Short: "同步指定项目任务到 Provider",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectSync,
}

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectSplitCmd)
	projectCmd.AddCommand(projectSplitMarkdownCmd)
	projectCmd.AddCommand(projectConfirmCmd)
	projectCmd.AddCommand(projectSyncCmd)

	for _, cmd := range []*cobra.Command{projectCreateCmd, projectListCmd, projectSplitCmd, projectSplitMarkdownCmd, projectConfirmCmd, projectSyncCmd} {
		cmd.Flags().StringVarP(&projectFormat, "format", "f", "text", "输出格式 (text, json)")
	}

	projectCreateCmd.Flags().StringVar(&projectDescription, "description", "", "项目描述")
	projectCreateCmd.Flags().StringVar(&projectParentID, "parent-id", "", "父项目 ID")
	projectCreateCmd.Flags().StringVar(&projectGoalText, "goal-text", "", "自然语言目标")
	projectCreateCmd.Flags().IntVar(&projectHorizonDays, "horizon-days", 14, "规划周期天数")
	projectCreateCmd.Flags().StringVar(&projectListID, "list-id", "", "默认任务清单 ID")
	projectCreateCmd.Flags().StringVar(&projectSource, "source", "", "目标来源（支持简写）")

	projectListCmd.Flags().StringVar(&projectStatusFilter, "status", "", "按状态过滤")

	projectSplitCmd.Flags().StringVar(&projectAIHint, "ai-hint", "", "拆分提示")
	projectSplitCmd.Flags().StringVar(&projectGoalText, "goal-text", "", "临时覆盖目标文本")
	projectSplitCmd.Flags().IntVar(&projectHorizonDays, "horizon-days", 0, "临时覆盖规划周期")
	projectSplitCmd.Flags().IntVar(&projectMaxTasks, "max-tasks", 12, "最大拆分任务数")
	projectSplitCmd.Flags().BoolVar(&projectRequireDeliverable, "require-deliverable", false, "强制每个子任务有交付物")
	projectSplitCmd.Flags().IntVar(&projectMinEstimateMinutes, "min-estimate-minutes", 0, "最小时长")
	projectSplitCmd.Flags().IntVar(&projectMaxEstimateMinutes, "max-estimate-minutes", 0, "最大时长")
	projectSplitCmd.Flags().IntVar(&projectMinTasks, "min-tasks", 0, "最少任务数")
	projectSplitCmd.Flags().IntVar(&projectConstraintMaxTasks, "constraint-max-tasks", 0, "约束里的最多任务数")
	projectSplitCmd.Flags().IntVar(&projectMinPracticeTasks, "min-practice-tasks", 0, "最少实战任务数")

	projectSplitMarkdownCmd.Flags().StringVar(&projectMarkdownFile, "file", "", "Markdown 文件路径")
	projectSplitMarkdownCmd.Flags().StringVar(&projectMarkdownInline, "markdown", "", "内联 Markdown 文本")
	projectSplitMarkdownCmd.Flags().IntVar(&projectHorizonDays, "horizon-days", 0, "临时覆盖规划周期")
	projectSplitMarkdownCmd.Flags().IntVar(&projectMaxTasks, "max-tasks", 200, "最多保留任务数")

	projectConfirmCmd.Flags().StringVar(&projectPlanID, "plan-id", "", "指定计划 ID")
	projectConfirmCmd.Flags().BoolVar(&projectWriteTasks, "write-tasks", true, "是否写入本地任务")

	projectSyncCmd.Flags().StringVar(&projectProvider, "provider", "", "目标 Provider（支持简写）")
	_ = projectSyncCmd.MarkFlagRequired("provider")
}

func runProjectCreate(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return commandError("初始化项目存储失败", err)
	}
	source := strings.TrimSpace(projectSource)
	if source != "" {
		source = provider.ResolveProviderName(source)
		if !provider.IsValidProvider(source) {
			return usageError("无效 provider: " + projectSource)
		}
	}
	item, err := (&projectservice.Service{ProjectStore: projectStore}).CreateProject(ctx, projectservice.CreateInput{
		Name:        args[0],
		Description: projectDescription,
		ParentID:    projectParentID,
		GoalText:    projectGoalText,
		HorizonDays: projectHorizonDays,
		ListID:      projectListID,
		Source:      source,
	})
	if err != nil {
		return commandError("保存项目失败", err)
	}
	return printResult(item)
}

func runProjectList(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return commandError("初始化项目存储失败", err)
	}
	svc := &projectservice.Service{ProjectStore: projectStore}
	items, err := svc.ListProjects(ctx, projectStatusFilter)
	if err != nil {
		return commandError("列出项目失败", err)
	}
	if projectFormat == "json" {
		return printResult(items)
	}
	for _, line := range projectservice.ProjectListText(ctx, projectStore, items) {
		fmt.Println(line)
	}
	return nil
}

func runProjectSplit(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return commandError("初始化项目存储失败", err)
	}
	result, err := (&projectservice.Service{ProjectStore: projectStore}).SplitProject(ctx, projectservice.SplitInput{
		ProjectID:          args[0],
		GoalText:           projectGoalText,
		HorizonDays:        projectHorizonDays,
		MaxTasks:           projectMaxTasks,
		AIHint:             projectAIHint,
		RequireDeliverable: projectRequireDeliverable,
		MinEstimateMinutes: projectMinEstimateMinutes,
		MaxEstimateMinutes: projectMaxEstimateMinutes,
		MinTasks:           projectMinTasks,
		ConstraintMaxTasks: projectConstraintMaxTasks,
		MinPracticeTasks:   projectMinPracticeTasks,
	})
	if err != nil {
		return commandError("生成项目计划失败", err)
	}
	return printResult(result)
}

func runProjectSplitMarkdown(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return commandError("初始化项目存储失败", err)
	}
	markdown, err := projectservice.ReadMarkdownInput(projectMarkdownFile, projectMarkdownInline)
	if err != nil {
		return commandError("读取 Markdown 文件失败", err)
	}
	result, err := (&projectservice.Service{ProjectStore: projectStore}).SplitProjectMarkdown(ctx, projectservice.SplitMarkdownInput{
		ProjectID:   args[0],
		Markdown:    markdown,
		HorizonDays: projectHorizonDays,
		MaxTasks:    projectMaxTasks,
	})
	if err != nil {
		if err.Error() == "请传入 --file 或 --markdown" {
			return usageError(err.Error())
		}
		return commandError("解析 Markdown 项目计划失败", err)
	}
	return printResult(result)
}

func runProjectConfirm(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	taskStore, projectStore, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	result, err := (&projectservice.Service{TaskStore: taskStore, ProjectStore: projectStore}).ConfirmProject(ctx, projectservice.ConfirmInput{
		ProjectID:  args[0],
		PlanID:     projectPlanID,
		WriteTasks: projectWriteTasks,
	})
	if err != nil {
		return commandError("确认项目失败", err)
	}
	return printResult(result)
}

func runProjectSync(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	taskStore, projectStore, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	projectID := strings.TrimSpace(args[0])
	if _, err := projectStore.GetProject(ctx, projectID); err != nil {
		return commandError("读取项目失败", err)
	}
	providers, err := loadAuthenticatedProviders(projectProvider)
	if err != nil {
		return commandError("初始化 provider 失败", err)
	}
	providerName := provider.ResolveProviderName(projectProvider)
	p := providers[providerName]

	result, err := (&projectservice.SyncService{
		TaskStore:    taskStore,
		ProjectStore: projectStore,
	}).SyncProject(ctx, projectID, p, providerName)
	if err != nil {
		return commandError("同步项目失败", err)
	}
	return printResult(result)
}

func getCLIStores() (storage.Storage, project.Store, func(), error) {
	taskStore, cleanup, err := getStore()
	if err != nil {
		return nil, nil, func() {}, err
	}
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		cleanup()
		return nil, nil, func() {}, err
	}
	return taskStore, projectStore, cleanup, nil
}

func printResult(value interface{}) error {
	if IsQuietMode() {
		bytes, err := json.Marshal(value)
		if err != nil {
			return commandError("序列化输出失败", err)
		}
		fmt.Println(string(bytes))
		return nil
	}
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return commandError("序列化输出失败", err)
	}
	fmt.Println(string(bytes))
	return nil
}
