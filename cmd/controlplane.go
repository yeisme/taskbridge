package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/actionaudit"
	"github.com/yeisme/taskbridge/internal/actionexecution"
	"github.com/yeisme/taskbridge/internal/actionfile"
	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/controlplane"
	"github.com/yeisme/taskbridge/pkg/ui"
)

var (
	controlFormat   string
	controlSource   string
	controlLimit    int
	controlMock     bool
	reviewApplyFile string
	reviewDryRun    bool
	reviewConfirm   bool
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Daily task workbench",
	RunE:  runToday,
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Recommend current next steps",
	RunE:  runNext,
}

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List tasks to be organized",
	RunE:  runInbox,
}

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Task health review",
	RunE:  runReview,
}

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Explore TaskBridge with built-in demo data (no provider authentication required)",
}

var demoTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "Preview the daily workbench using demo data",
	RunE:  runDemoToday,
}

func init() {
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(inboxCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(demoCmd)

	demoCmd.AddCommand(demoTodayCmd)
	demoTodayCmd.Flags().StringVarP(&controlFormat, "format", "f", "text", "Output format (text, json)")

	for _, cmd := range []*cobra.Command{todayCmd, nextCmd, inboxCmd, reviewCmd} {
		cmd.Flags().StringVarP(&controlFormat, "format", "f", "text", "Output format (text, json)")
		cmd.Flags().StringVar(&controlSource, "source", "", "Filter by source (abbreviations supported)")
		cmd.Flags().BoolVar(&controlMock, "mock", false, "Use built-in mock data (no provider authentication required)")
	}
	for _, cmd := range []*cobra.Command{nextCmd, inboxCmd} {
		cmd.Flags().IntVar(&controlLimit, "limit", 0, "Maximum number of returned tasks")
	}
	reviewCmd.Flags().StringVar(&reviewApplyFile, "apply-file", "", "Execute structured action file")
	reviewCmd.Flags().BoolVar(&reviewDryRun, "dry-run", false, "Simulate execution of action file")
	reviewCmd.Flags().BoolVar(&reviewConfirm, "confirm", false, "Confirm execution of action file")
}

func runDemoToday(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	service := mockControlService()
	result, err := service.Today(ctx, controlplane.Options{Source: controlSource})
	if err != nil {
		return commandError("Failed to generate demo workbench", err)
	}
	return printTodayResult(controlFormat, result)
}

func controlService() (*controlplane.Service, func(), error) {
	taskStore, projectStore, cleanup, err := getCLIStores()
	if err != nil {
		return nil, cleanup, err
	}
	return &controlplane.Service{TaskStore: taskStore, ProjectStore: projectStore}, cleanup, nil
}

func mockControlService() *controlplane.Service {
	return controlplane.NewMockService(time.Now())
}

func controlServiceForCommand() (*controlplane.Service, func(), error) {
	if controlMock {
		return mockControlService(), func() {}, nil
	}
	return controlService()
}

func runToday(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	service, cleanup, err := controlServiceForCommand()
	if err != nil {
		return commandError("Failed to initialize control plane", err)
	}
	defer cleanup()
	result, err := service.Today(ctx, controlplane.Options{Source: controlSource})
	if err != nil {
		return commandError("Failed to generate today's workbench", err)
	}
	return printTodayResult(controlFormat, result)
}

func runNext(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	service, cleanup, err := controlServiceForCommand()
	if err != nil {
		return commandError("Failed to initialize control plane", err)
	}
	defer cleanup()
	result, err := service.Next(ctx, controlplane.Options{Source: controlSource, Limit: controlLimit})
	if err != nil {
		return commandError("Failed to generate next step", err)
	}
	return printTaskListResult(controlFormat, "Suggest next steps", result)
}

func runInbox(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	service, cleanup, err := controlServiceForCommand()
	if err != nil {
		return commandError("Failed to initialize control plane", err)
	}
	defer cleanup()
	result, err := service.Inbox(ctx, controlplane.Options{Source: controlSource, Limit: controlLimit})
	if err != nil {
		return commandError("Failed to generate inbox", err)
	}
	return printTaskListResult(controlFormat, "Tasks to be organized", result)
}

func runReview(_ *cobra.Command, _ []string) error {
	if reviewApplyFile != "" {
		return runReviewApplyFile()
	}
	ctx := context.Background()

	service, cleanup, err := controlServiceForCommand()
	if err != nil {
		return commandError("Failed to initialize control plane", err)
	}
	defer cleanup()
	result, err := service.Review(ctx, controlplane.Options{Source: controlSource})
	if err != nil {
		return commandError("Failed to generate review disk", err)
	}
	projection := buildReviewProjection(result)
	return printProjectionWithLegacyJSON(controlFormat, result, projection, func() { fmt.Print(renderControlReview(result, projection)) })
}

func runReviewApplyFile() error {
	if err := validateReviewApplyMode(reviewDryRun, reviewConfirm); err != nil {
		return err
	}
	taskStore, _, cleanup, err := getCLIStores()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()

	sessionID := fmt.Sprintf("review_%s", time.Now().Format("20060102_150405"))
	service := actionexecution.Service{TaskStore: taskStore, AuditStore: actionaudit.NewStore(cfg.Storage.Path)}
	execResult, err := service.ExecuteFile(context.Background(), actionexecution.Options{
		SessionID:      sessionID,
		Command:        "review --apply-file",
		ActionFilePath: reviewApplyFile,
		DryRun:         reviewDryRun,
		Confirm:        reviewConfirm,
	})
	if err != nil {
		return commandError("Failed to read action file", err)
	}
	result := execResult.Execution
	projection := buildReviewApplyProjection(result)
	projection.Facts["audit_receipt_id"] = sessionID
	if err := printProjectionWithLegacyJSON(controlFormat, result, projection, func() { fmt.Print(renderProjectProjection("Review action file", projection)) }); err != nil {
		return err
	}
	if result.Status == "error" {
		return &CLIError{Message: "review --apply-file completed with errors", ExitCode: 1}
	}
	return nil
}

func validateReviewApplyMode(dryRun, confirm bool) error {
	if dryRun || confirm {
		return nil
	}
	return usageError("You must explicitly add --dry-run or --confirm when executing --apply-file")
}

func printTodayResult(format string, result *controlplane.TodayResult) error {
	projection := buildTodayProjection(result)
	return printProjectionWithLegacyJSON(format, result, projection, func() { fmt.Print(renderControlToday(result, projection)) })
}

func printTaskListResult(format, title string, result *controlplane.ListResult) error {
	projection := buildTaskListProjection(title, result)
	return printProjectionWithLegacyJSON(format, result, projection, func() { fmt.Print(renderControlTaskList(result, projection)) })
}

func buildTodayProjection(result *controlplane.TodayResult) clioutput.Projection {
	p := clioutput.New("task.today")
	p.Summary = "Daily workbench generated."
	if result != nil {
		p.Status = cliStatusFromString(result.Status)
		p.Facts["date"] = result.Date
		for key, value := range result.Summary {
			p.Facts[key] = value
		}
		p.Facts["sections"] = len(result.Sections)
		p.Facts["suggested_actions"] = len(result.SuggestedActions)
		p.Risks = append(p.Risks, result.Warnings...)
		p.Actions = append(p.Actions, clioutput.Action{Name: "next", Command: "taskbridge next"})
		if len(result.SuggestedActions) > 0 {
			p.Actions = append(p.Actions, clioutput.Action{Name: "review", Command: "taskbridge review"})
		}
	}
	p.Data = result
	return p
}

func buildTaskListProjection(title string, result *controlplane.ListResult) clioutput.Projection {
	command := "task.list"
	summary := title + "."
	if result != nil && result.Schema == controlplane.SchemaNext {
		command = "task.next"
		summary = "Next steps listed."
	} else if result != nil && result.Schema == controlplane.SchemaInbox {
		command = "task.inbox"
		summary = "Inbox tasks listed."
	}
	p := clioutput.New(command)
	p.Summary = summary
	if result != nil {
		p.Status = cliStatusFromString(result.Status)
		p.Facts["count"] = result.Count
		p.Risks = append(p.Risks, result.Warnings...)
		if result.Schema == controlplane.SchemaNext {
			p.Facts["recommendations"] = result.Count
			for i, task := range result.Tasks {
				if i >= 5 {
					break
				}
				prefix := fmt.Sprintf("recommendation.%d", i+1)
				p.Facts[prefix+".id"] = task.ID
				p.Facts[prefix+".source"] = task.Source
				p.Facts[prefix+".domain"] = task.Domain
				p.Facts[prefix+".action"] = task.NextAction
			}
		}
	}
	if result != nil && result.Schema == controlplane.SchemaNext {
		p.Actions = append(p.Actions, clioutput.Action{Name: "review", Command: "taskbridge review"})
	}
	p.Data = result
	return p
}

func buildReviewProjection(result *controlplane.ReviewResult) clioutput.Projection {
	p := clioutput.New("task.review")
	p.Summary = "Task health review generated."
	if result != nil {
		p.Status = cliStatusFromString(result.Status)
		for key, value := range result.Summary {
			p.Facts[key] = value
		}
		p.Facts["suggested_actions"] = len(result.SuggestedActions)
		p.Risks = append(p.Risks, result.Warnings...)
		if len(result.SuggestedActions) > 0 {
			p.Actions = append(p.Actions, clioutput.Action{Name: "apply", Command: "taskbridge review --apply-file <path> --dry-run"})
		}
	}
	p.Data = result
	return p
}

func buildReviewApplyProjection(result actionfile.ExecuteResult) clioutput.Projection {
	p := buildActionExecuteProjection("task.review_apply", "Review action file processed.", result)
	if reviewApplyFile != "" {
		p.Facts["action_file"] = reviewApplyFile
	}
	if result.RequiresConfirmation {
		p.Actions = []clioutput.Action{{Name: "confirm", Command: "taskbridge review --apply-file " + reviewApplyFile + " --confirm"}}
	}
	return p
}

func renderControlToday(result *controlplane.TodayResult, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Daily workbench\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if result == nil {
		return renderRecommendedAction(b.String(), projection)
	}
	for _, section := range result.Sections {
		b.WriteString("\n")
		b.WriteString(section.Title)
		b.WriteString("\n")
		appendTaskRefsTable(&b, section.Tasks)
	}
	if len(result.ProjectNext) > 0 {
		b.WriteString("\nProject next steps\n")
		table := ui.NewTable("Project", "Name", "Next task", "Risk")
		for _, item := range result.ProjectNext {
			table.AddRow(item.ProjectID, item.ProjectName, item.NextTaskID, item.RiskLevel)
		}
		b.WriteString(table.Render())
		b.WriteString("\n")
	}
	return renderRecommendedAction(b.String(), projection)
}

func renderControlTaskList(result *controlplane.ListResult, projection clioutput.Projection) string {
	var b strings.Builder
	title := "Tasks"
	if result != nil && result.Schema == controlplane.SchemaNext {
		title = "Next steps"
	} else if result != nil && result.Schema == controlplane.SchemaInbox {
		title = "Inbox tasks"
	}
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if result == nil || len(result.Tasks) == 0 {
		b.WriteString("\nNo tasks found.\n")
		return renderRecommendedAction(b.String(), projection)
	}
	b.WriteString("\nTask table\n")
	appendTaskRefsTable(&b, result.Tasks)
	return renderRecommendedAction(b.String(), projection)
}

func renderControlReview(result *controlplane.ReviewResult, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Task health review\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if result != nil && len(result.SuggestedActions) > 0 {
		b.WriteString("\nSuggested actions\n")
		table := ui.NewTable("Action", "Task", "Project", "Reason", "Confirm")
		for _, action := range result.SuggestedActions {
			table.AddRow(action.Type, action.TaskID, action.ProjectID, action.Reason, fmt.Sprint(action.RequiresConfirmation))
		}
		b.WriteString(table.Render())
		b.WriteString("\n")
	}
	return renderRecommendedAction(b.String(), projection)
}

func appendTaskRefsTable(b *strings.Builder, tasks []controlplane.TaskRef) {
	if len(tasks) == 0 {
		b.WriteString("No tasks found.\n")
		return
	}
	table := ui.NewTable("Task", "Title", "Status", "Priority", "Source", "Domain", "Reason")
	for _, task := range tasks {
		table.AddRow(task.ID, task.Title, task.Status, fmt.Sprint(task.Priority), task.Source, task.Domain, task.Reason)
	}
	b.WriteString(table.Render())
	b.WriteString("\n")
}
