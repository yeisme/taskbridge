package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/projectservice"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
	syncengine "github.com/yeisme/taskbridge/internal/sync"
	"github.com/yeisme/taskbridge/pkg/ui"
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
	Short: "Project planning and implementation",
	Long: `Manage project drafts, split suggestions, confirmation, and per-project sync workflows.

Subcommands:
  create          Create a project draft
  list            List projects
  split           Generate split recommendations
  split-markdown  Generate split recommendations from a Markdown task tree
  confirm         Confirm a plan and optionally write tasks
  sync            Sync tasks for one project

Examples:
  taskbridge project create "Learn OpenClaw" --goal-text "I want to learn openclaw"
  taskbridge project split <project-id> --max-tasks 10
  taskbridge project confirm <project-id> --write-tasks
  taskbridge project sync <project-id> --provider google`,
}

var projectCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a project draft",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectCreate,
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	RunE:  runProjectList,
}

var projectSplitCmd = &cobra.Command{
	Use:   "split <project-id>",
	Short: "Generate split recommendations",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectSplit,
}

var projectSplitMarkdownCmd = &cobra.Command{
	Use:   "split-markdown <project-id>",
	Short: "Generate split recommendations from a Markdown task tree",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectSplitMarkdown,
}

var projectConfirmCmd = &cobra.Command{
	Use:   "confirm <project-id>",
	Short: "Confirm the project and release tasks",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectConfirm,
}

var projectSyncCmd = &cobra.Command{
	Use:   "sync <project-id>",
	Short: "Synchronize specified project tasks to Provider",
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
		cmd.Flags().StringVarP(&projectFormat, "format", "f", "text", "Output format (text, json)")
	}

	projectCreateCmd.Flags().StringVar(&projectDescription, "description", "", "Project description")
	projectCreateCmd.Flags().StringVar(&projectParentID, "parent-id", "", "Parent project ID")
	projectCreateCmd.Flags().StringVar(&projectGoalText, "goal-text", "", "Natural language goal")
	projectCreateCmd.Flags().IntVar(&projectHorizonDays, "horizon-days", 14, "Planning cycle days")
	projectCreateCmd.Flags().StringVar(&projectListID, "list-id", "", "Default task list ID")
	projectCreateCmd.Flags().StringVar(&projectSource, "source", "", "Target source (abbreviation supported)")

	projectListCmd.Flags().StringVar(&projectStatusFilter, "status", "", "Filter by status")

	projectSplitCmd.Flags().StringVar(&projectAIHint, "ai-hint", "", "Split hint")
	projectSplitCmd.Flags().StringVar(&projectGoalText, "goal-text", "", "Temporarily overwrite target text")
	projectSplitCmd.Flags().IntVar(&projectHorizonDays, "horizon-days", 0, "Temporary coverage planning cycle")
	projectSplitCmd.Flags().IntVar(&projectMaxTasks, "max-tasks", 12, "Maximum number of split tasks")
	projectSplitCmd.Flags().BoolVar(&projectRequireDeliverable, "require-deliverable", false, "Enforce deliverables for each subtask")
	projectSplitCmd.Flags().IntVar(&projectMinEstimateMinutes, "min-estimate-minutes", 0, "Minimum duration")
	projectSplitCmd.Flags().IntVar(&projectMaxEstimateMinutes, "max-estimate-minutes", 0, "Maximum duration")
	projectSplitCmd.Flags().IntVar(&projectMinTasks, "min-tasks", 0, "Minimum number of tasks")
	projectSplitCmd.Flags().IntVar(&projectConstraintMaxTasks, "constraint-max-tasks", 0, "Constraint maximum task count")
	projectSplitCmd.Flags().IntVar(&projectMinPracticeTasks, "min-practice-tasks", 0, "Minimum number of actual combat tasks")

	projectSplitMarkdownCmd.Flags().StringVar(&projectMarkdownFile, "file", "", "Markdown file path")
	projectSplitMarkdownCmd.Flags().StringVar(&projectMarkdownInline, "markdown", "", "Inline Markdown text")
	projectSplitMarkdownCmd.Flags().IntVar(&projectHorizonDays, "horizon-days", 0, "Temporary coverage planning cycle")
	projectSplitMarkdownCmd.Flags().IntVar(&projectMaxTasks, "max-tasks", 200, "Maximum number of retained tasks")

	projectConfirmCmd.Flags().StringVar(&projectPlanID, "plan-id", "", "Specify plan ID")
	projectConfirmCmd.Flags().BoolVar(&projectWriteTasks, "write-tasks", true, "Whether to write local tasks")

	projectSyncCmd.Flags().StringVar(&projectProvider, "provider", "", "Target Provider (abbreviation supported)")
	_ = projectSyncCmd.MarkFlagRequired("provider")
}

func runProjectCreate(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return commandError("Failed to initialize project storage", err)
	}
	source := strings.TrimSpace(projectSource)
	if source != "" {
		source = provider.ResolveProviderName(source)
		if !provider.IsValidProvider(source) {
			return usageError("Invalid provider:" + projectSource)
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
		return commandError("Failed to save project", err)
	}
	projection := buildProjectCreateProjection(item)
	return printProjectionWithLegacyJSON(projectFormat, item, projection, func() { fmt.Print(renderProjectProjection("Project created", projection)) })
}

func runProjectList(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return commandError("Failed to initialize project storage", err)
	}
	svc := &projectservice.Service{ProjectStore: projectStore}
	items, err := svc.ListProjects(ctx, projectStatusFilter)
	if err != nil {
		return commandError("List projects failed", err)
	}
	projection := buildProjectListProjection(ctx, projectStore, items)
	return printProjectionWithLegacyJSON(projectFormat, items, projection, func() { fmt.Print(renderProjectList(items, projection)) })
}

func runProjectSplit(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return commandError("Failed to initialize project storage", err)
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
		return commandError("Failed to generate project plan", err)
	}
	projection := buildProjectMapProjection("project.split", "Project split recommendation generated.", result)
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project split", projection)) })
}

func runProjectSplitMarkdown(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	projectStore, err := project.NewFileStore(cfg.Storage.Path)
	if err != nil {
		return commandError("Failed to initialize project storage", err)
	}
	markdown, err := projectservice.ReadMarkdownInput(projectMarkdownFile, projectMarkdownInline)
	if err != nil {
		return commandError("Failed to read Markdown file", err)
	}
	result, err := (&projectservice.Service{ProjectStore: projectStore}).SplitProjectMarkdown(ctx, projectservice.SplitMarkdownInput{
		ProjectID:   args[0],
		Markdown:    markdown,
		HorizonDays: projectHorizonDays,
		MaxTasks:    projectMaxTasks,
	})
	if err != nil {
		if err.Error() == "Please pass in --file or --markdown" {
			return usageError(err.Error())
		}
		return commandError("Parsing Markdown project plan failed", err)
	}
	projection := buildProjectMapProjection("project.split_markdown", "Project split recommendation generated from Markdown.", result)
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project split", projection)) })
}

func runProjectConfirm(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	taskStore, projectStore, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	result, err := (&projectservice.Service{TaskStore: taskStore, ProjectStore: projectStore}).ConfirmProject(ctx, projectservice.ConfirmInput{
		ProjectID:  args[0],
		PlanID:     projectPlanID,
		WriteTasks: projectWriteTasks,
	})
	if err != nil {
		return commandError("Confirm project failed", err)
	}
	projection := buildProjectMapProjection("project.confirm", "Project plan confirmed.", result)
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project confirmation", projection)) })
}

func runProjectSync(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	taskStore, projectStore, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	projectID := strings.TrimSpace(args[0])
	if _, err := projectStore.GetProject(ctx, projectID); err != nil {
		return commandError("Failed to read item", err)
	}
	providers, err := loadAuthenticatedProviders(projectProvider)
	if err != nil {
		return commandError("Failed to initialize provider", err)
	}
	providerName := provider.ResolveProviderName(projectProvider)
	p := providers[providerName]

	result, err := (&projectservice.SyncService{
		TaskStore:    taskStore,
		ProjectStore: projectStore,
	}).SyncProject(ctx, projectID, p, providerName)
	if err != nil {
		return commandError("Sync project failed", err)
	}
	projection := buildProjectSyncProjection(result)
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project sync", projection)) })
}

func printProjectionWithLegacyJSON(format string, legacy interface{}, projection clioutput.Projection, renderText func()) error {
	if globalProjectionModeRequested() {
		return printProjection(format, projection, nil)
	}
	if renderText == nil {
		renderText = func() { fmt.Print(clioutput.RenderSummary(projection)) }
	}
	return printStructured(format, legacy, renderText)
}

func buildProjectCreateProjection(item *project.Project) clioutput.Projection {
	p := clioutput.New("project.create")
	p.Summary = "Project draft created."
	if item != nil {
		p.Facts["project_id"] = item.ID
		p.Facts["name"] = item.Name
		p.Facts["status"] = item.Status
		p.Facts["goal_type"] = item.GoalType
		p.Facts["horizon_days"] = item.HorizonDays
		if item.LatestPlanID != "" {
			p.Facts["plan_id"] = item.LatestPlanID
		}
		p.Actions = append(p.Actions, clioutput.Action{Name: "split", Command: "taskbridge project split " + item.ID})
	}
	p.Data = item
	return p
}

func buildProjectListProjection(ctx context.Context, store project.Store, items []project.Project) clioutput.Projection {
	p := clioutput.New("project.list")
	p.Summary = "Projects listed."
	p.Facts["count"] = len(items)
	if projectStatusFilter != "" {
		p.Facts["status_filter"] = projectStatusFilter
	}
	for _, line := range projectservice.ProjectListText(ctx, store, items) {
		p.Preview = append(p.Preview, clioutput.PreviewItem{Label: "project", Value: line})
	}
	p.Data = items
	return p
}

func buildProjectMapProjection(command, summary string, result map[string]interface{}) clioutput.Projection {
	p := clioutput.New(command)
	p.Summary = summary
	for _, key := range []string{"project_id", "plan_id", "status", "next_task_id", "mode", "provider", "message"} {
		if value, ok := result[key]; ok {
			p.Facts[key] = value
		}
	}
	for _, key := range []string{"count", "pulled", "pushed", "skipped", "updated"} {
		if value, ok := result[key]; ok {
			p.Facts[key] = value
		}
	}
	if warnings, ok := result["warnings"].([]string); ok {
		p.Risks = append(p.Risks, warnings...)
	}
	p.Data = result
	return p
}

func buildProjectSyncProjection(result *syncengine.ProjectSyncResult) clioutput.Projection {
	p := clioutput.New("project.sync")
	p.Summary = "Project sync completed."
	if result != nil {
		p.Status = cliStatusFromString(result.Status)
		p.Facts["project_id"] = result.ProjectID
		p.Facts["provider"] = result.Provider
		p.Facts["status"] = result.Status
		p.Facts["pushed"] = result.Pushed
		p.Facts["updated"] = result.Updated
		if result.Message != "" {
			p.Facts["message"] = result.Message
		}
		if len(result.Errors) > 0 {
			p.Status = clioutput.StatusPartial
			p.Risks = append(p.Risks, result.Errors...)
		}
	}
	p.Data = result
	return p
}

func renderProjectList(items []project.Project, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Projects\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if len(items) == 0 {
		b.WriteString("\nNo projects found.\n")
		return renderRecommendedAction(b.String(), projection)
	}

	table := ui.NewTable("ID", "Name", "Status", "Goal", "Horizon", "Source")
	for _, item := range items {
		table.AddRow(item.ID, item.Name, string(item.Status), string(item.GoalType), fmt.Sprint(item.HorizonDays), item.Source)
	}
	b.WriteString("\nProject table\n")
	b.WriteString(table.Render())
	b.WriteString("\n")
	return renderRecommendedAction(b.String(), projection)
}

func renderProjectProjection(title string, projection clioutput.Projection) string {
	var b strings.Builder
	if strings.TrimSpace(title) == "" {
		title = projection.Summary
	}
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(renderProjectionFacts(projection))
	return renderRecommendedAction(b.String(), projection)
}

func renderProjectionFacts(projection clioutput.Projection) string {
	var b strings.Builder
	if projection.Summary != "" {
		b.WriteString(projection.Summary)
		b.WriteString("\n")
	}
	if len(projection.Facts) > 0 {
		b.WriteString("\nSummary\n")
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
			if strings.TrimSpace(risk) == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(risk)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderRecommendedAction(prefix string, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString(prefix)
	for _, action := range projection.Actions {
		if strings.TrimSpace(action.Command) == "" {
			continue
		}
		b.WriteString("\nRecommended next step\n")
		b.WriteString(action.Command)
		b.WriteString("\n")
		break
	}
	return b.String()
}

func cliStatusFromString(status string) clioutput.Status {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "error", "failed", "failure":
		return clioutput.StatusFailed
	case "partial", "warning", "degraded":
		return clioutput.StatusPartial
	default:
		return clioutput.StatusSuccess
	}
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
			return commandError("Serialized output failed", err)
		}
		fmt.Println(string(bytes))
		return nil
	}
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return commandError("Serialized output failed", err)
	}
	fmt.Println(string(bytes))
	return nil
}
