package cmd

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/projectplanner"
	"github.com/yeisme/taskbridge/internal/provider"
	msprovider "github.com/yeisme/taskbridge/internal/provider/microsoft"
	"github.com/yeisme/taskbridge/internal/storage"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
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

var (
	unorderedListItemPattern = regexp.MustCompile(`^[-*+]\s+(.+)$`)
	orderedListItemPattern   = regexp.MustCompile(`^\d+[.)]\s+(.+)$`)
	orderedTitlePrefix       = regexp.MustCompile(`^\d+[.)]\s+`)
	markdownCheckboxPrefix   = regexp.MustCompile(`^\s*\[\s*[xX ]?\s*\]\s*`)
)

type markdownNode struct {
	Title        string
	Indent       int
	SiblingIndex int
	Children     []*markdownNode
}

type markdownParseStats struct {
	TotalNodes   int `json:"total_nodes"`
	LeafTasks    int `json:"leaf_tasks"`
	IgnoredLines int `json:"ignored_lines"`
}

type syncProjectResult struct {
	ProjectID string   `json:"project_id"`
	Provider  string   `json:"provider"`
	Status    string   `json:"status"`
	Pushed    int      `json:"pushed"`
	Updated   int      `json:"updated"`
	Errors    []string `json:"errors,omitempty"`
	Message   string   `json:"message,omitempty"`
}

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
	Run:   runProjectCreate,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出项目",
	Run:   runProjectList,
}

var projectSplitCmd = &cobra.Command{
	Use:   "split <project-id>",
	Short: "生成拆分建议",
	Args:  cobra.ExactArgs(1),
	Run:   runProjectSplit,
}

var projectSplitMarkdownCmd = &cobra.Command{
	Use:   "split-markdown <project-id>",
	Short: "从 Markdown 任务树生成拆分建议",
	Args:  cobra.ExactArgs(1),
	Run:   runProjectSplitMarkdown,
}

var projectConfirmCmd = &cobra.Command{
	Use:   "confirm <project-id>",
	Short: "确认项目并落库任务",
	Args:  cobra.ExactArgs(1),
	Run:   runProjectConfirm,
}

var projectSyncCmd = &cobra.Command{
	Use:   "sync <project-id>",
	Short: "同步指定项目任务到 Provider",
	Args:  cobra.ExactArgs(1),
	Run:   runProjectSync,
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

func runProjectCreate(_ *cobra.Command, args []string) {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		exitErr("初始化项目存储失败", err)
	}

	source := strings.TrimSpace(projectSource)
	if source != "" {
		source = provider.ResolveProviderName(source)
		if !provider.IsValidProvider(source) {
			exitErr("无效 provider", fmt.Errorf("%s", projectSource))
		}
	}

	goalText := strings.TrimSpace(projectGoalText)
	if goalText == "" {
		goalText = strings.TrimSpace(args[0])
	}
	now := time.Now()
	item := &project.Project{
		ID:           generateProjectID(),
		Name:         strings.TrimSpace(args[0]),
		Description:  strings.TrimSpace(projectDescription),
		ParentID:     strings.TrimSpace(projectParentID),
		GoalText:     goalText,
		GoalType:     projectplanner.DetectGoalType(goalText),
		Status:       project.StatusDraft,
		ListID:       strings.TrimSpace(projectListID),
		Source:       source,
		HorizonDays:  normalizeHorizon(projectHorizonDays, 14),
		LatestPlanID: "",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := projectStore.SaveProject(ctx, item); err != nil {
		exitErr("保存项目失败", err)
	}
	printResult(item)
}

func runProjectList(_ *cobra.Command, _ []string) {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		exitErr("初始化项目存储失败", err)
	}
	items, err := projectStore.ListProjects(ctx, strings.TrimSpace(projectStatusFilter))
	if err != nil {
		exitErr("列出项目失败", err)
	}
	if projectFormat == "json" {
		printResult(items)
		return
	}
	if len(items) == 0 {
		fmt.Println("暂无项目")
		return
	}
	fmt.Println("项目列表:")
	for _, item := range items {
		summary := ""
		if plan, err := projectStore.GetLatestPlan(ctx, item.ID); err == nil {
			summary = fmt.Sprintf(" | %d phases / %d tasks", len(plan.Phases), len(plan.TasksPreview))
		}
		fmt.Printf("- %s | %s | %s%s\n", item.ID, item.Name, item.Status, summary)
	}
}

func runProjectSplit(_ *cobra.Command, args []string) {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		exitErr("初始化项目存储失败", err)
	}
	item, err := projectStore.GetProject(ctx, strings.TrimSpace(args[0]))
	if err != nil {
		exitErr("读取项目失败", err)
	}

	goalText := strings.TrimSpace(projectGoalText)
	if goalText == "" {
		goalText = item.GoalText
	}
	suggestion := projectplanner.Decompose(projectplanner.DecomposeInput{
		ProjectID:   item.ID,
		ProjectName: item.Name,
		GoalText:    goalText,
		GoalType:    item.GoalType,
		HorizonDays: normalizeHorizon(projectHorizonDays, item.HorizonDays),
		MaxTasks:    projectMaxTasks,
		AIHint:      strings.TrimSpace(projectAIHint),
		Constraints: project.PlanConstraints{
			RequireDeliverable: projectRequireDeliverable,
			MinEstimateMinutes: projectMinEstimateMinutes,
			MaxEstimateMinutes: projectMaxEstimateMinutes,
			MinTasks:           projectMinTasks,
			MaxTasks:           projectConstraintMaxTasks,
			MinPracticeTasks:   projectMinPracticeTasks,
		},
	})
	suggestion.PlanID = generatePlanID()
	if err := projectStore.SavePlan(ctx, suggestion); err != nil {
		exitErr("保存项目计划失败", err)
	}
	item.GoalText = goalText
	item.GoalType = suggestion.GoalType
	item.Status = project.StatusSplitSuggested
	item.LatestPlanID = suggestion.PlanID
	item.HorizonDays = normalizeHorizon(projectHorizonDays, item.HorizonDays)
	if err := projectStore.SaveProject(ctx, item); err != nil {
		exitErr("更新项目失败", err)
	}
	printResult(map[string]interface{}{
		"project_id":    item.ID,
		"plan_id":       suggestion.PlanID,
		"status":        suggestion.Status,
		"confidence":    suggestion.Confidence,
		"constraints":   suggestion.Constraints,
		"tasks_preview": suggestion.TasksPreview,
		"phases":        suggestion.Phases,
		"warnings":      suggestion.Warnings,
	})
}

func runProjectSplitMarkdown(_ *cobra.Command, args []string) {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		exitErr("初始化项目存储失败", err)
	}
	item, err := projectStore.GetProject(ctx, strings.TrimSpace(args[0]))
	if err != nil {
		exitErr("读取项目失败", err)
	}
	markdown := strings.TrimSpace(projectMarkdownInline)
	if projectMarkdownFile != "" {
		bytes, err := os.ReadFile(projectMarkdownFile)
		if err != nil {
			exitErr("读取 Markdown 文件失败", err)
		}
		markdown = string(bytes)
	}
	if strings.TrimSpace(markdown) == "" {
		exitErr("缺少 markdown 输入", fmt.Errorf("请传入 --file 或 --markdown"))
	}

	root, stats, warnings := parseMarkdownTaskTree(markdown)
	horizonDays := normalizeHorizon(projectHorizonDays, item.HorizonDays)
	maxTasks := normalizeMarkdownMaxTasks(projectMaxTasks)
	tasks, phases, buildWarnings := buildPlanTasksFromMarkdown(root, item.ID, horizonDays, maxTasks)
	warnings = append(warnings, buildWarnings...)
	stats.LeafTasks = countLeafNodes(root)
	if stats.LeafTasks == 0 {
		exitErr("解析 Markdown 失败", fmt.Errorf("no valid markdown task leaf nodes found"))
	}
	if len(tasks) < stats.LeafTasks {
		warnings = append(warnings, fmt.Sprintf("leaf tasks truncated by max_tasks=%d", maxTasks))
	}

	suggestion := &project.PlanSuggestion{
		PlanID:       generatePlanID(),
		ProjectID:    item.ID,
		GoalText:     item.GoalText,
		GoalType:     item.GoalType,
		Status:       project.StatusSplitSuggested,
		TasksPreview: tasks,
		Phases:       phases,
		Confidence:   0.9,
		Warnings:     warnings,
		CreatedAt:    time.Now(),
	}
	if err := projectStore.SavePlan(ctx, suggestion); err != nil {
		exitErr("保存项目计划失败", err)
	}
	item.Status = project.StatusSplitSuggested
	item.LatestPlanID = suggestion.PlanID
	item.HorizonDays = horizonDays
	if err := projectStore.SaveProject(ctx, item); err != nil {
		exitErr("更新项目失败", err)
	}
	printResult(map[string]interface{}{
		"project_id":    item.ID,
		"plan_id":       suggestion.PlanID,
		"status":        suggestion.Status,
		"confidence":    suggestion.Confidence,
		"tasks_preview": suggestion.TasksPreview,
		"phases":        suggestion.Phases,
		"warnings":      suggestion.Warnings,
		"stats":         stats,
	})
}

func runProjectConfirm(_ *cobra.Command, args []string) {
	ctx := context.Background()
	taskStore, projectStore, err := getCLIStores()
	if err != nil {
		exitErr("初始化存储失败", err)
	}
	item, err := projectStore.GetProject(ctx, strings.TrimSpace(args[0]))
	if err != nil {
		exitErr("读取项目失败", err)
	}

	var plan *project.PlanSuggestion
	if strings.TrimSpace(projectPlanID) != "" {
		plan, err = projectStore.GetPlan(ctx, item.ID, strings.TrimSpace(projectPlanID))
	} else {
		plan, err = projectStore.GetLatestPlan(ctx, item.ID)
	}
	if err != nil {
		exitErr("读取项目计划失败", err)
	}

	if projectWriteTasks && len(plan.ConfirmedTaskIDs) > 0 {
		printResult(map[string]interface{}{
			"project_id":       item.ID,
			"status":           project.StatusConfirmed,
			"created_task_ids": plan.ConfirmedTaskIDs,
			"count":            len(plan.ConfirmedTaskIDs),
		})
		return
	}

	createdTaskIDs := make([]string, 0)
	if projectWriteTasks {
		now := time.Now()
		planToLocalID := make(map[string]string, len(plan.TasksPreview))
		for idx, planTask := range plan.TasksPreview {
			task := &model.Task{
				ID:          generateID(),
				Title:       planTask.Title,
				Description: planTask.Description,
				Status:      model.StatusTodo,
				CreatedAt:   now,
				UpdatedAt:   now,
				Source:      model.SourceLocal,
				ListID:      item.ListID,
				Priority:    clampTaskPriority(planTask.Priority),
				Quadrant:    clampTaskQuadrant(planTask.Quadrant),
				Tags:        append([]string{}, planTask.Tags...),
			}
			dueDate := now.AddDate(0, 0, maxInt(1, planTask.DueOffsetDays))
			task.DueDate = &dueDate
			task.Metadata = &model.TaskMetadata{
				Version:    "1.0",
				Quadrant:   int(task.Quadrant),
				Priority:   int(task.Priority),
				LocalID:    task.ID,
				SyncSource: "local",
				CustomFields: map[string]interface{}{
					"tb_project_id": item.ID,
					"tb_plan_id":    plan.PlanID,
					"tb_goal_type":  string(item.GoalType),
					"tb_phase":      planTask.Phase,
					"tb_step_index": idx + 1,
				},
			}
			if strings.TrimSpace(planTask.ParentID) != "" {
				if parentLocalID, ok := planToLocalID[strings.TrimSpace(planTask.ParentID)]; ok {
					parentID := parentLocalID
					task.ParentID = &parentID
				}
			}
			if strings.TrimSpace(planTask.ID) != "" {
				task.Metadata.CustomFields["tb_plan_task_id"] = strings.TrimSpace(planTask.ID)
			}
			if strings.TrimSpace(planTask.ParentID) != "" {
				task.Metadata.CustomFields["tb_parent_plan_task_id"] = strings.TrimSpace(planTask.ParentID)
			}
			if err := taskStore.SaveTask(ctx, task); err != nil {
				exitErr("保存项目任务失败", err)
			}
			createdTaskIDs = append(createdTaskIDs, task.ID)
			if strings.TrimSpace(planTask.ID) != "" {
				planToLocalID[strings.TrimSpace(planTask.ID)] = task.ID
			}
		}
	}

	now := time.Now()
	plan.Status = project.StatusConfirmed
	plan.ConfirmedTaskIDs = createdTaskIDs
	plan.ConfirmedAt = &now
	if err := projectStore.SavePlan(ctx, plan); err != nil {
		exitErr("更新项目计划失败", err)
	}
	item.Status = project.StatusConfirmed
	item.LatestPlanID = plan.PlanID
	if err := projectStore.SaveProject(ctx, item); err != nil {
		exitErr("更新项目失败", err)
	}
	printResult(map[string]interface{}{
		"project_id":       item.ID,
		"status":           item.Status,
		"created_task_ids": createdTaskIDs,
		"count":            len(createdTaskIDs),
	})
}

func runProjectSync(_ *cobra.Command, args []string) {
	ctx := context.Background()
	taskStore, projectStore, err := getCLIStores()
	if err != nil {
		exitErr("初始化存储失败", err)
	}
	projectID := strings.TrimSpace(args[0])
	if _, err := projectStore.GetProject(ctx, projectID); err != nil {
		exitErr("读取项目失败", err)
	}
	providers, err := loadAuthenticatedProviders(projectProvider)
	if err != nil {
		exitErr("初始化 provider 失败", err)
	}
	providerName := provider.ResolveProviderName(projectProvider)
	p := providers[providerName]

	localTasks, err := taskStore.ListTasks(ctx, storage.ListOptions{})
	if err != nil {
		exitErr("读取本地任务失败", err)
	}
	projectTasks := make([]model.Task, 0)
	for _, task := range localTasks {
		if taskBelongsToProject(task, projectID) {
			projectTasks = append(projectTasks, task)
		}
	}

	result := syncProjectResult{
		ProjectID: projectID,
		Provider:  providerName,
		Status:    string(project.StatusConfirmed),
		Errors:    []string{},
	}
	if len(projectTasks) == 0 {
		result.Message = "未找到可同步的项目任务"
		printResult(result)
		return
	}

	taskLists, err := p.ListTaskLists(ctx)
	if err != nil {
		exitErr("读取远程清单失败", err)
	}
	defaultListID := findDefaultTaskListID(taskLists)
	targetSource := model.TaskSource(providerName)
	if targetSource == model.SourceGoogle {
		planTaskToLocalID := buildPlanTaskToLocalIDMap(projectTasks)
		parentTasks, childTasks := splitTasksByParentRelation(projectTasks, planTaskToLocalID)
		pushProjectLocalTasks(ctx, taskStore, p, parentTasks, defaultListID, targetSource, false, &result)
		if len(childTasks) > 0 {
			if err := pullProviderTasksIntoLocal(ctx, taskStore, p, providerName); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("pull-before-children: %v", err))
			}
			pushProjectLocalTasks(ctx, taskStore, p, childTasks, defaultListID, targetSource, false, &result)
		}
	} else {
		pushProjectLocalTasks(ctx, taskStore, p, projectTasks, defaultListID, targetSource, false, &result)
	}

	item, err := projectStore.GetProject(ctx, projectID)
	if err == nil {
		item.Status = project.StatusSynced
		_ = projectStore.SaveProject(ctx, item)
		result.Status = string(project.StatusSynced)
	}
	result.Message = fmt.Sprintf("项目已同步到 %s", providerName)
	printResult(result)
}

func getCLIStores() (storage.Storage, project.Store, error) {
	taskStore, err := filestore.New(cfg.Storage.Path, cfg.Storage.File.Format)
	if err != nil {
		return nil, nil, err
	}
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return nil, nil, err
	}
	return taskStore, projectStore, nil
}

func printResult(value interface{}) {
	if strings.EqualFold(projectFormat, "json") {
		bytes, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			exitErr("序列化输出失败", err)
		}
		fmt.Println(string(bytes))
		return
	}
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		exitErr("序列化输出失败", err)
	}
	fmt.Println(string(bytes))
}

func exitErr(msg string, err error) {
	fmt.Fprintf(os.Stderr, "❌ %s: %v\n", msg, err)
	os.Exit(1)
}

func generateProjectID() string {
	return fmt.Sprintf("proj_%d", time.Now().UnixNano())
}

func generatePlanID() string {
	return fmt.Sprintf("plan_%d", time.Now().UnixNano())
}

func normalizeHorizon(value, fallback int) int {
	if value <= 0 {
		value = fallback
	}
	if value <= 0 {
		value = 14
	}
	if value < 7 {
		return 7
	}
	if value > 30 {
		return 30
	}
	return value
}

func clampTaskPriority(priority int) model.Priority {
	if priority < 0 {
		return model.Priority(0)
	}
	if priority > 4 {
		return model.Priority(4)
	}
	return model.Priority(priority)
}

func clampTaskQuadrant(quadrant int) model.Quadrant {
	if quadrant < 1 {
		return model.QuadrantNotUrgentImportant
	}
	if quadrant > 4 {
		return model.QuadrantNotUrgentNotImportant
	}
	return model.Quadrant(quadrant)
}

func findDefaultTaskListID(taskLists []model.TaskList) string {
	for _, list := range taskLists {
		if list.Name == "我的任务" || list.Name == "My Tasks" || list.ID == "@default" {
			return list.ID
		}
	}
	if len(taskLists) > 0 {
		return taskLists[0].ID
	}
	return ""
}

func pushProjectLocalTasks(ctx context.Context, taskStore storage.Storage, p provider.Provider, tasks []model.Task, defaultListID string, source model.TaskSource, dryRun bool, result *syncProjectResult) {
	planTaskToLocalID := buildPlanTaskToLocalIDMap(tasks)
	ordered := orderTasksByParent(tasks, planTaskToLocalID)
	remoteByLocalID := make(map[string]string, len(ordered))

	for _, task := range ordered {
		if task.Source != "" && task.Source != source && task.Source != model.SourceLocal {
			continue
		}
		listID := task.ListID
		if listID == "" {
			listID = defaultListID
		}

		parentRemoteID := ""
		parentLocalID := localParentIDFromTask(task, planTaskToLocalID)
		if parentLocalID != "" {
			parentRemoteID = remoteByLocalID[parentLocalID]
			if parentRemoteID == "" {
				parentRemoteID = findRemoteIDByLocalParent(ctx, taskStore, tasks, parentLocalID, source)
			}
			if parentRemoteID == "" && source == model.SourceGoogle {
				parentRemoteID = toGoogleParentRawID(parentLocalID, listID)
			}
			if parentRemoteID == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("skip child %s: parent remote id not resolved", task.ID))
				continue
			}
		}

		if source == model.SourceMicrosoft && parentRemoteID != "" {
			if err := syncMicrosoftChecklistStep(ctx, taskStore, p, listID, parentRemoteID, &task, dryRun, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("step %s: %v", task.ID, err))
			}
			continue
		}

		taskToSync := sanitizeTaskForRemote(task)
		if source == model.SourceGoogle && parentRemoteID != "" {
			taskToSync.ParentID = &parentRemoteID
		}
		if taskToSync.SourceRawID != "" {
			existingTask, err := p.GetTask(ctx, listID, taskToSync.SourceRawID)
			if err == nil && existingTask != nil {
				if existingTask.UpdatedAt.Before(taskToSync.UpdatedAt) {
					if !dryRun {
						if _, err := p.UpdateTask(ctx, listID, &taskToSync); err != nil {
							result.Errors = append(result.Errors, fmt.Sprintf("update %s: %v", taskToSync.ID, err))
							continue
						}
					}
					result.Updated++
				}
				remoteByLocalID[task.ID] = taskToSync.SourceRawID
				continue
			}
		}

		if !dryRun {
			createdTask, err := p.CreateTask(ctx, listID, &taskToSync)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("create %s: %v", taskToSync.ID, err))
				continue
			}
			task.SourceRawID = createdTask.SourceRawID
			task.ListID = listID
			task.Source = source
			_ = taskStore.SaveTask(ctx, &task)
			remoteByLocalID[task.ID] = task.SourceRawID
		} else if task.SourceRawID != "" {
			remoteByLocalID[task.ID] = task.SourceRawID
		}
		result.Pushed++
	}
}

func syncMicrosoftChecklistStep(ctx context.Context, taskStore storage.Storage, p provider.Provider, listID, parentRemoteID string, task *model.Task, dryRun bool, result *syncProjectResult) error {
	msProvider, ok := p.(*msprovider.Provider)
	if !ok {
		return fmt.Errorf("provider is not microsoft")
	}
	stepID := parseMicrosoftStepID(task.SourceRawID)
	if task.Metadata != nil && task.Metadata.CustomFields != nil {
		if v, ok := task.Metadata.CustomFields["tb_ms_step_id"]; ok {
			stepID = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	isChecked := task.Status == model.StatusCompleted
	cleanTitle := sanitizeMarkdownText(task.Title)
	if dryRun {
		if stepID == "" {
			result.Pushed++
		} else {
			result.Updated++
		}
		return nil
	}
	if stepID == "" {
		item, err := msProvider.CreateChecklistItem(ctx, listID, parentRemoteID, cleanTitle, isChecked)
		if err != nil {
			return err
		}
		originalID := task.ID
		canonicalID := buildMicrosoftStepLocalID(parentRemoteID, item.ID)
		ensureTaskMetadata(task)
		task.Metadata.CustomFields["tb_ms_step_id"] = item.ID
		task.Metadata.CustomFields["tb_ms_parent_source_raw_id"] = parentRemoteID
		task.Metadata.LocalID = canonicalID
		task.ID = canonicalID
		task.SourceRawID = "ms_step:" + item.ID
		task.Source = model.SourceMicrosoft
		task.ListID = listID
		_ = taskStore.SaveTask(ctx, task)
		if originalID != "" && originalID != canonicalID {
			_ = taskStore.DeleteTask(ctx, originalID)
		}
		result.Pushed++
		return nil
	}

	if _, err := msProvider.UpdateChecklistItem(ctx, listID, parentRemoteID, stepID, cleanTitle, isChecked); err != nil {
		return err
	}
	canonicalID := buildMicrosoftStepLocalID(parentRemoteID, stepID)
	ensureTaskMetadata(task)
	task.Metadata.CustomFields["tb_ms_step_id"] = stepID
	task.Metadata.CustomFields["tb_ms_parent_source_raw_id"] = parentRemoteID
	task.Metadata.LocalID = canonicalID
	task.ID = canonicalID
	task.SourceRawID = "ms_step:" + stepID
	task.Source = model.SourceMicrosoft
	task.ListID = listID
	_ = taskStore.SaveTask(ctx, task)
	result.Updated++
	return nil
}

func ensureTaskMetadata(task *model.Task) {
	if task.Metadata == nil {
		task.Metadata = &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{}}
	}
	if task.Metadata.CustomFields == nil {
		task.Metadata.CustomFields = map[string]interface{}{}
	}
}

func orderTasksByParent(tasks []model.Task, planTaskToLocalID map[string]string) []model.Task {
	byID := make(map[string]model.Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}
	visited := make(map[string]bool, len(tasks))
	inStack := make(map[string]bool, len(tasks))
	ordered := make([]model.Task, 0, len(tasks))
	var visit func(string)
	visit = func(id string) {
		if visited[id] || inStack[id] {
			return
		}
		task, ok := byID[id]
		if !ok {
			return
		}
		inStack[id] = true
		if parentLocalID := localParentIDFromTask(task, planTaskToLocalID); parentLocalID != "" {
			visit(parentLocalID)
		}
		inStack[id] = false
		visited[id] = true
		ordered = append(ordered, task)
	}
	for _, task := range tasks {
		visit(task.ID)
	}
	return ordered
}

func findRemoteIDByLocalParent(ctx context.Context, taskStore storage.Storage, tasks []model.Task, parentLocalID string, source model.TaskSource) string {
	for _, task := range tasks {
		if task.ID == parentLocalID && task.SourceRawID != "" && (task.Source == source || task.Source == model.SourceLocal || task.Source == "") {
			return task.SourceRawID
		}
	}
	if parentTask, err := taskStore.GetTask(ctx, parentLocalID); err == nil && parentTask != nil {
		if parentTask.SourceRawID != "" && (parentTask.Source == source || parentTask.Source == model.SourceLocal || parentTask.Source == "") {
			return parentTask.SourceRawID
		}
	}
	all, err := taskStore.ListTasks(ctx, storage.ListOptions{})
	if err != nil {
		return ""
	}
	for _, task := range all {
		if getCustomFieldString(task, "tb_plan_task_id") != parentLocalID {
			continue
		}
		if task.SourceRawID != "" && (task.Source == source || task.Source == model.SourceLocal || task.Source == "") {
			return task.SourceRawID
		}
	}
	return ""
}

func pullProviderTasksIntoLocal(ctx context.Context, taskStore storage.Storage, p provider.Provider, providerName string) error {
	taskLists, err := p.ListTaskLists(ctx)
	if err != nil {
		return err
	}
	for _, list := range taskLists {
		_ = taskStore.SaveTaskList(ctx, &list)
		tasks, err := p.ListTasks(ctx, list.ID, provider.ListOptions{})
		if err != nil {
			continue
		}
		for _, task := range tasks {
			if task.ListID == "" {
				task.ListID = list.ID
			}
			if task.ListName == "" {
				task.ListName = list.Name
			}
			if task.Source == "" {
				task.Source = model.TaskSource(providerName)
			}
			_ = taskStore.SaveTask(ctx, &task)
		}
	}
	return nil
}

func buildPlanTaskToLocalIDMap(tasks []model.Task) map[string]string {
	result := make(map[string]string, len(tasks))
	for _, task := range tasks {
		planTaskID := getCustomFieldString(task, "tb_plan_task_id")
		if planTaskID != "" {
			result[planTaskID] = task.ID
		}
	}
	return result
}

func localParentIDFromTask(task model.Task, planTaskToLocalID map[string]string) string {
	if task.ParentID != nil {
		if parentID := strings.TrimSpace(*task.ParentID); parentID != "" {
			return parentID
		}
	}
	parentPlanTaskID := getCustomFieldString(task, "tb_parent_plan_task_id")
	if parentPlanTaskID == "" {
		return ""
	}
	return strings.TrimSpace(planTaskToLocalID[parentPlanTaskID])
}

func getCustomFieldString(task model.Task, key string) string {
	if task.Metadata == nil || task.Metadata.CustomFields == nil {
		return ""
	}
	raw, ok := task.Metadata.CustomFields[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func sanitizeTaskForRemote(task model.Task) model.Task {
	task.Title = sanitizeMarkdownText(task.Title)
	task.Description = sanitizeMarkdownText(task.Description)
	return task
}

func sanitizeMarkdownText(text string) string {
	out := strings.TrimSpace(text)
	if out == "" {
		return out
	}
	out = markdownCheckboxPrefix.ReplaceAllString(out, "")
	out = strings.ReplaceAll(out, "**", "")
	out = strings.ReplaceAll(out, "__", "")
	out = strings.Join(strings.Fields(out), " ")
	return out
}

func parseMicrosoftStepID(sourceRawID string) string {
	if !strings.HasPrefix(strings.TrimSpace(sourceRawID), "ms_step:") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(sourceRawID), "ms_step:"))
}

func buildMicrosoftStepLocalID(parentRemoteID, stepID string) string {
	parent := strings.TrimSpace(parentRemoteID)
	step := strings.TrimSpace(stepID)
	if parent == "" || step == "" {
		return ""
	}
	return fmt.Sprintf("ms-step-%s-%s", parent, step)
}

func toGoogleParentRawID(parentID, listID string) string {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" || listID == "" {
		return ""
	}
	prefix := "google-" + listID + "-"
	if strings.HasPrefix(parentID, prefix) {
		return strings.TrimPrefix(parentID, prefix)
	}
	return ""
}

func splitTasksByParentRelation(tasks []model.Task, planTaskToLocalID map[string]string) ([]model.Task, []model.Task) {
	parents := make([]model.Task, 0, len(tasks))
	children := make([]model.Task, 0, len(tasks))
	for _, task := range tasks {
		if localParentIDFromTask(task, planTaskToLocalID) == "" {
			parents = append(parents, task)
		} else {
			children = append(children, task)
		}
	}
	return parents, children
}

func taskBelongsToProject(task model.Task, projectID string) bool {
	if task.Metadata == nil || task.Metadata.CustomFields == nil {
		return false
	}
	raw, ok := task.Metadata.CustomFields["tb_project_id"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case string:
		return v == projectID
	case fmt.Stringer:
		return v.String() == projectID
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64) == projectID
	case int:
		return strconv.Itoa(v) == projectID
	default:
		return fmt.Sprint(v) == projectID
	}
}

func parseMarkdownTaskTree(markdown string) (*markdownNode, markdownParseStats, []string) {
	root := &markdownNode{Title: "__root__", Indent: -1}
	stack := []*markdownNode{root}
	stats := markdownParseStats{}
	warnings := make([]string, 0)
	previousIndent := -1

	for i, raw := range strings.Split(markdown, "\n") {
		line := strings.ReplaceAll(raw, "\t", "  ")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			stats.IgnoredLines++
			continue
		}
		indent := countLeadingSpaces(line)
		title, ok := parseMarkdownListTitle(strings.TrimSpace(line[indent:]))
		if !ok {
			stats.IgnoredLines++
			warnings = append(warnings, fmt.Sprintf("line %d ignored: not a markdown list item", i+1))
			continue
		}
		if previousIndent >= 0 && indent > previousIndent+2 {
			warnings = append(warnings, fmt.Sprintf("line %d indent jump from %d to %d; attached to nearest parent", i+1, previousIndent, indent))
		}
		previousIndent = indent
		for len(stack) > 1 && indent <= stack[len(stack)-1].Indent {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1]
		node := &markdownNode{
			Title:        title,
			Indent:       indent,
			SiblingIndex: len(parent.Children) + 1,
		}
		parent.Children = append(parent.Children, node)
		stack = append(stack, node)
		stats.TotalNodes++
	}
	return root, stats, warnings
}

func buildPlanTasksFromMarkdown(root *markdownNode, projectID string, horizonDays, maxTasks int) ([]project.PlanTask, []string, []string) {
	tasks := make([]project.PlanTask, 0)
	phases := make([]string, 0)
	phaseSeen := map[string]bool{}
	warnings := make([]string, 0)

	for _, top := range root.Children {
		collectMarkdownTasks(top, projectID, []string{top.Title}, []int{top.SiblingIndex}, top.Title, "", &tasks, &phases, phaseSeen)
	}
	if maxTasks <= 0 {
		maxTasks = 200
	}
	if len(tasks) > maxTasks {
		tasks = tasks[:maxTasks]
	}
	for i := range tasks {
		tasks[i].DueOffsetDays = distributeDueOffset(i, len(tasks), horizonDays)
	}
	return tasks, phases, warnings
}

func collectMarkdownTasks(node *markdownNode, projectID string, pathTitles []string, pathIndexes []int, phase, parentPlanTaskID string, tasks *[]project.PlanTask, phases *[]string, phaseSeen map[string]bool) {
	phase = strings.TrimSpace(phase)
	if phase == "" {
		phase = "导入任务"
	}
	if !phaseSeen[phase] {
		phaseSeen[phase] = true
		*phases = append(*phases, phase)
	}
	pathHash := sha1.Sum([]byte(strings.Join(pathTitles, " > ")))
	planTaskID := hex.EncodeToString(pathHash[:8])
	*tasks = append(*tasks, project.PlanTask{
		ID:              planTaskID,
		ParentID:        parentPlanTaskID,
		Title:           node.Title,
		Description:     fmt.Sprintf("由 Markdown 任务树导入：%s", strings.Join(pathTitles, " > ")),
		EstimateMinutes: clampMarkdownEstimate(len(node.Children)),
		Priority:        clampMarkdownPriority(len(pathTitles)),
		Quadrant:        2,
		Tags:            []string{"markdown-import"},
		Phase:           phase,
	})
	for _, child := range node.Children {
		collectMarkdownTasks(child, projectID, append(append([]string{}, pathTitles...), child.Title), append(append([]int{}, pathIndexes...), child.SiblingIndex), phase, planTaskID, tasks, phases, phaseSeen)
	}
}

func countLeafNodes(root *markdownNode) int {
	if root == nil {
		return 0
	}
	if len(root.Children) == 0 {
		return 1
	}
	total := 0
	for _, child := range root.Children {
		total += countLeafNodes(child)
	}
	return total
}

func countLeadingSpaces(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func parseMarkdownListTitle(value string) (string, bool) {
	if matches := unorderedListItemPattern.FindStringSubmatch(value); len(matches) == 2 {
		return normalizeMarkdownTitle(matches[1]), true
	}
	if matches := orderedListItemPattern.FindStringSubmatch(value); len(matches) == 2 {
		return normalizeMarkdownTitle(matches[1]), true
	}
	return "", false
}

func normalizeMarkdownTitle(title string) string {
	out := strings.TrimSpace(title)
	for {
		next := strings.TrimSpace(orderedTitlePrefix.ReplaceAllString(out, ""))
		if next == out {
			break
		}
		out = next
	}
	out = sanitizeMarkdownText(out)
	return out
}

func normalizeMarkdownMaxTasks(value int) int {
	if value <= 0 {
		return 200
	}
	if value > 500 {
		return 500
	}
	return value
}

func distributeDueOffset(index, total, horizonDays int) int {
	if horizonDays <= 0 {
		horizonDays = 14
	}
	if total <= 1 {
		return 1
	}
	offset := int(mathRound(float64(index+1) * float64(horizonDays) / float64(total)))
	if offset < 1 {
		return 1
	}
	if offset > horizonDays {
		return horizonDays
	}
	return offset
}

func clampMarkdownEstimate(children int) int {
	switch {
	case children >= 3:
		return 120
	case children > 0:
		return 90
	default:
		return 60
	}
}

func clampMarkdownPriority(depth int) int {
	switch {
	case depth <= 1:
		return 3
	case depth == 2:
		return 2
	default:
		return 1
	}
}

func mathRound(value float64) float64 {
	if value < 0 {
		return float64(int(value - 0.5))
	}
	return float64(int(value + 0.5))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
