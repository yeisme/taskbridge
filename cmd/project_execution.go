package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/actionfile"
	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/projectservice"
	"github.com/yeisme/taskbridge/pkg/ui"
)

var (
	projectAdjustReason  string
	projectAdjustDryRun  bool
	projectAdjustConfirm bool
)

var projectReviewCmd = &cobra.Command{
	Use:   "review <project-id>",
	Short: "Review project execution status",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectReview,
}

var projectNextCmd = &cobra.Command{
	Use:   "next <project-id>",
	Short: "Export project next step",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectNext,
}

var projectAdjustCmd = &cobra.Command{
	Use:   "adjust <project-id>",
	Short: "Generate or apply project adjustments based on execution status",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectAdjust,
}

var projectDoneCmd = &cobra.Command{
	Use:   "done <project-id>",
	Short: "Mark project complete",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectDone,
}

var projectArchiveCmd = &cobra.Command{
	Use:   "archive <project-id>",
	Short: "Archive items",
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
		cmd.Flags().StringVarP(&projectFormat, "format", "f", "text", "Output format (text, json)")
	}
	projectAdjustCmd.Flags().StringVar(&projectAdjustReason, "reason", "", "Reason for adjustment")
	projectAdjustCmd.Flags().BoolVar(&projectAdjustDryRun, "dry-run", true, "Simulate execution")
	projectAdjustCmd.Flags().BoolVar(&projectAdjustConfirm, "confirm", false, "Confirm application adjustments")
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
	if err != nil {
		return commandError("Failed to initialize project service", err)
	}
	defer cleanup()
	result, err := service.Review(context.Background(), args[0])
	if err != nil {
		return commandError("Project review failed", err)
	}
	projection := buildProjectReviewProjection(result)
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectReview(result, projection)) })
}

func runProjectNext(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	if err != nil {
		return commandError("Failed to initialize project service", err)
	}
	defer cleanup()
	result, err := service.Next(context.Background(), args[0])
	if err != nil {
		return commandError("The next step of the project failed", err)
	}
	projection := buildProjectMapProjection("project.next", "Project next step selected.", result)
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project next step", projection)) })
}

func runProjectAdjust(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	if err != nil {
		return commandError("Failed to initialize project service", err)
	}
	defer cleanup()
	actions, err := service.Adjust(context.Background(), args[0], projectAdjustReason)
	if err != nil {
		return commandError("Build project adjustment failed", err)
	}
	if projectAdjustConfirm {
		result := actionfile.Executor{TaskStore: service.TaskStore}.Execute(context.Background(), actions, actionfile.ExecuteOptions{DryRun: false, Confirm: true})
		projection := buildActionExecuteProjection("project.adjust", "Project adjustment applied.", result)
		return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project adjustment", projection)) })
	}
	result := map[string]interface{}{"schema": "taskbridge.project-adjust.v1", "dry_run": projectAdjustDryRun, "requires_confirmation": len(actions.Actions) > 0, "actions": actions.Actions}
	projection := buildProjectAdjustPreviewProjection(result, len(actions.Actions))
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project adjustment preview", projection)) })
}

func runProjectDone(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	if err != nil {
		return commandError("Failed to initialize project service", err)
	}
	defer cleanup()
	result, err := service.Done(context.Background(), args[0])
	if err != nil {
		return commandError("Mark project complete failed", err)
	}
	projection := buildProjectMapProjection("project.done", "Project marked complete.", result)
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project completion", projection)) })
}

func runProjectArchive(_ *cobra.Command, args []string) error {
	service, cleanup, err := projectExecutionService()
	if err != nil {
		return commandError("Failed to initialize project service", err)
	}
	defer cleanup()
	result, err := service.Archive(context.Background(), args[0])
	if err != nil {
		return commandError("Archive project failed", err)
	}
	projection := buildProjectMapProjection("project.archive", "Project archived.", result)
	return printProjectionWithLegacyJSON(projectFormat, result, projection, func() { fmt.Print(renderProjectProjection("Project archive", projection)) })
}

func buildProjectReviewProjection(result *projectservice.ProjectReview) clioutput.Projection {
	p := clioutput.New("project.review")
	p.Summary = "Project review completed."
	if result != nil {
		p.Facts["project_id"] = result.ProjectID
		p.Facts["project_name"] = result.ProjectName
		p.Facts["status"] = result.Status
		p.Facts["next_task_id"] = result.NextTaskID
		for key, value := range result.Progress {
			p.Facts[key] = value
		}
		if risk, ok := result.Risk["level"]; ok {
			p.Facts["risk_level"] = risk
		}
		if reasons, ok := result.Risk["reasons"].([]string); ok {
			p.Risks = append(p.Risks, reasons...)
		}
		if result.NextTaskID != "" {
			p.Actions = append(p.Actions, clioutput.Action{Name: "next", Command: "taskbridge task show " + result.NextTaskID})
		}
	}
	p.Data = result
	return p
}

func buildProjectAdjustPreviewProjection(result map[string]interface{}, actionCount int) clioutput.Projection {
	p := buildProjectMapProjection("project.adjust", "Project adjustment preview generated.", result)
	p.Facts["actions"] = actionCount
	if actionCount > 0 {
		p.Actions = append(p.Actions, clioutput.Action{Name: "apply", Command: "taskbridge project adjust <project-id> --confirm"})
	}
	return p
}

func buildActionExecuteProjection(command, summary string, result actionfile.ExecuteResult) clioutput.Projection {
	p := clioutput.New(command)
	p.Summary = summary
	p.Status = cliStatusFromString(result.Status)
	p.Facts["dry_run"] = result.DryRun
	p.Facts["requires_confirmation"] = result.RequiresConfirmation
	p.Facts["total"] = result.Total
	p.Facts["updated"] = result.Updated
	p.Facts["skipped"] = result.Skipped
	if len(result.Errors) > 0 {
		p.Status = clioutput.StatusPartial
		p.Risks = append(p.Risks, result.Errors...)
	}
	if result.RequiresConfirmation {
		p.Actions = append(p.Actions, clioutput.Action{Name: "confirm", Command: "taskbridge project adjust <project-id> --confirm"})
	}
	p.Data = result
	return p
}

func renderProjectReview(result *projectservice.ProjectReview, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Project review\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if result != nil && len(result.SuggestedActions) > 0 {
		b.WriteString("\nSuggested actions\n")
		table := ui.NewTable("Action", "Task", "Reason", "Confirm")
		for _, action := range result.SuggestedActions {
			table.AddRow(action.Type, action.TaskID, action.Reason, fmt.Sprint(action.RequiresConfirmation))
		}
		b.WriteString(table.Render())
		b.WriteString("\n")
	}
	return renderRecommendedAction(b.String(), projection)
}
