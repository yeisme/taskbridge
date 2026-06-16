package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/actionaudit"
	"github.com/yeisme/taskbridge/internal/actionexecution"
	"github.com/yeisme/taskbridge/internal/agentcontract"
	"github.com/yeisme/taskbridge/internal/controlplane"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/projectplanner"
	"github.com/yeisme/taskbridge/internal/projectservice"
)

var (
	agentRequestID   string
	agentDryRun      bool
	agentConfirm     bool
	agentActionFile  string
	agentHorizonDays int
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent secure execution entry",
}

var agentCapabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Output Agent callability",
	RunE:  runAgentCapabilities,
}

var agentTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "Output Agent friendly today",
	RunE:  runAgentToday,
}

var agentPlanCmd = &cobra.Command{
	Use:   "plan <goal>",
	Short: "Generate target plan preview",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentPlan,
}

var agentExecuteCmd = &cobra.Command{
	Use:   "execute",
	Short: "Execute action file",
	RunE:  runAgentExecute,
}

var agentSchemasCmd = &cobra.Command{
	Use:   "schemas",
	Short: "Output Agent JSON schema name",
	RunE:  runAgentSchemas,
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentCapabilitiesCmd)
	agentCmd.AddCommand(agentTodayCmd)
	agentCmd.AddCommand(agentPlanCmd)
	agentCmd.AddCommand(agentExecuteCmd)
	agentCmd.AddCommand(agentSchemasCmd)

	for _, cmd := range []*cobra.Command{agentCapabilitiesCmd, agentTodayCmd, agentPlanCmd, agentExecuteCmd, agentSchemasCmd} {
		cmd.Flags().StringVar(&agentRequestID, "request-id", "", "Request ID")
	}
	agentPlanCmd.Flags().BoolVar(&agentDryRun, "dry-run", true, "Simulate execution")
	agentPlanCmd.Flags().IntVar(&agentHorizonDays, "horizon-days", 14, "Planning cycle days")
	agentExecuteCmd.Flags().StringVar(&agentActionFile, "action-file", "", "action file path")
	agentExecuteCmd.Flags().BoolVar(&agentDryRun, "dry-run", true, "Simulate execution")
	agentExecuteCmd.Flags().BoolVar(&agentConfirm, "confirm", false, "Confirm execution of dangerous actions")
	_ = agentExecuteCmd.MarkFlagRequired("action-file")
}

func runAgentCapabilities(_ *cobra.Command, _ []string) error {
	payload := map[string]interface{}{
		"schema":    "taskbridge.agent-capabilities.v1",
		"version":   "1.0",
		"providers": getAuthProviderOrder(),
		"commands": []string{
			"agent today",
			"agent plan",
			"agent execute",
			"agent schemas",
			"agent capabilities",
			"demo today",
			"today",
			"next",
			"review",
			"audit show",
			"audit list",
			"sync diff",
		},
		"schema_versions": []string{
			"taskbridge.agent-result.v1",
			"taskbridge.agent-capabilities.v1",
			"taskbridge.today.v1",
			"taskbridge.actions.v1",
			"taskbridge.action-result.v1",
			"taskbridge.action-audit.v1",
			"taskbridge.sync-session.v1",
			"taskbridge.project-review.v1",
		},
		"dangerous_actions": []string{
			"delete_task",
			"complete_task",
			"defer_task",
			"reschedule_task",
			"conflict_discard",
		},
		"audit": map[string]interface{}{
			"supported":        true,
			"receipt_schema":   "taskbridge.action-audit.v1",
			"read_commands":    []string{"taskbridge audit show <session-id>", "taskbridge audit list"},
			"write_triggers":   []string{"review --apply-file --confirm", "agent execute --confirm"},
			"dry_run_no_write": true,
		},
		"output_behavior": map[string]interface{}{
			"json_envelope":     "taskbridge.agent-result.v1 on stdout",
			"agent_keyvalue":    "--agent produces stable key=value on stdout",
			"error_exit_code":   "non-zero on failure",
			"stdout_json":       "stdout is valid JSON even on error paths",
			"stderr_diagnostic": "human-readable diagnostics only, no tokens or payloads",
		},
		"confirmation": map[string]interface{}{
			"required_for":    []string{"complete_task", "delete_task", "defer_task", "reschedule_task"},
			"dry_run_safe":    true,
			"no_silent_write": true,
		},
		"not_implemented": []string{"mcp_adapter", "remote_provider_write", "bidirectional_sync_auto_resolve"},
	}
	return printAgent(agentcontract.OK(requestID(), false, payload))
}

func runAgentToday(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	service, cleanup, err := controlService()
	if err != nil {
		return printAgent(agentcontract.Error(requestID(), "store_init_failed", err.Error(), "taskbridge doctor"))
	}
	defer cleanup()
	result, err := service.Today(ctx, controlplane.Options{})
	if err != nil {
		return printAgent(agentcontract.Error(requestID(), "today_failed", err.Error(), "taskbridge today"))
	}
	return printAgent(agentcontract.OK(requestID(), false, result))
}

func runAgentPlan(_ *cobra.Command, args []string) error {
	if !agentDryRun {
		projectStore, err := project.NewFileStore(cfg.Storage.Path)
		if err != nil {
			return printAgent(agentcontract.Error(requestID(), "project_store_init_failed", err.Error(), "taskbridge doctor"))
		}
		result, err := (&projectservice.Service{ProjectStore: projectStore}).CreateProjectDraftPlan(context.Background(), projectservice.DraftPlanInput{
			Name:        args[0],
			GoalText:    args[0],
			HorizonDays: agentHorizonDays,
			MaxTasks:    10,
		})
		if err != nil {
			return printAgent(agentcontract.Error(requestID(), "agent_plan_failed", err.Error(), "Check the target text and try again"))
		}
		payload := map[string]interface{}{
			"schema":          "taskbridge.agent-plan.v1",
			"goal":            args[0],
			"project_id":      result.Project.ID,
			"plan_id":         result.Plan.PlanID,
			"project_status":  result.Project.Status,
			"created_project": true,
			"created_plan":    true,
			"plan": map[string]interface{}{
				"status":        result.Plan.Status,
				"confidence":    result.Plan.Confidence,
				"constraints":   result.Plan.Constraints,
				"tasks_preview": result.Plan.TasksPreview,
				"phases":        result.Plan.Phases,
				"warnings":      result.Plan.Warnings,
			},
		}
		return printAgent(agentcontract.OK(requestID(), false, payload))
	}
	plan := projectplanner.Decompose(projectplanner.DecomposeInput{
		ProjectID:   "agent_preview",
		ProjectName: args[0],
		GoalText:    args[0],
		GoalType:    projectplanner.DetectGoalType(args[0]),
		HorizonDays: agentHorizonDays,
		MaxTasks:    10,
	})
	result := map[string]interface{}{"schema": "taskbridge.agent-plan.v1", "goal": args[0], "would_create_project": true, "plan": plan}
	return printAgent(agentcontract.OK(requestID(), agentDryRun, result))
}

func runAgentExecute(_ *cobra.Command, _ []string) error {
	effectiveDryRun := effectiveAgentExecuteDryRun(agentDryRun, agentConfirm)
	sessionID := requestID()
	taskStore, _, cleanup, err := getCLIStores()
	if err != nil {
		return printAgent(agentcontract.Error(sessionID, "store_init_failed", err.Error(), "taskbridge doctor"))
	}
	defer cleanup()

	service := actionexecution.Service{TaskStore: taskStore, AuditStore: actionaudit.NewStore(cfg.Storage.Path)}
	execResult, err := service.ExecuteFile(context.Background(), actionexecution.Options{
		SessionID:      sessionID,
		Command:        "agent execute",
		ActionFilePath: agentActionFile,
		DryRun:         effectiveDryRun,
		Confirm:        agentConfirm,
	})
	if err != nil {
		return printAgent(agentcontract.Error(sessionID, "action_file_invalid", err.Error(), "Check action file"))
	}

	result := execResult.Execution
	envelope := agentcontract.OK(sessionID, effectiveDryRun, result)
	if result.RequiresConfirmation {
		envelope = agentcontract.Confirmation(sessionID, effectiveDryRun, result)
	}
	if result.Status == "error" {
		envelope.Status = "error"
	}
	if envelope.Result != nil {
		if data, ok := envelope.Result.(map[string]interface{}); ok {
			data["audit_receipt_id"] = sessionID
		}
	}
	printErr := printAgent(envelope)
	if printErr != nil {
		return printErr
	}
	if result.Status == "error" {
		return &CLIError{Message: "agent execute completed with errors", ExitCode: 1}
	}
	return nil
}

func runAgentSchemas(_ *cobra.Command, _ []string) error {
	payload := map[string]interface{}{
		"schema": "taskbridge.agent-schema-index.v1",
		"schemas": []map[string]interface{}{
			{
				"name":        "taskbridge.agent-result.v1",
				"description": "Agent command result envelope. Fields: schema, status, request_id, dry_run, requires_confirmation, result, warnings, errors. stdout is always valid JSON even on error.",
				"commands":    []string{"agent today", "agent plan", "agent execute", "agent schemas", "agent capabilities"},
			},
			{
				"name":        "taskbridge.agent-capabilities.v1",
				"description": "Agent capability declaration. Lists implemented commands, dangerous actions, schema versions, audit support, output behavior, and confirmation rules. Does not claim MCP or remote write.",
				"commands":    []string{"agent capabilities"},
			},
			{
				"name":        "taskbridge.today.v1",
				"description": "Daily workbench data model. Sections: must_do, at_risk, suggested_next, project_next. Summary counts per section.",
				"commands":    []string{"today", "demo today", "next"},
			},
			{
				"name":        "taskbridge.actions.v1",
				"description": "Action file input schema. Fields: schema, source, created_at, actions[]. Action types: defer_task, reschedule_task, complete_task, split_task.",
				"commands":    []string{"review --apply-file", "agent execute"},
			},
			{
				"name":        "taskbridge.action-result.v1",
				"description": "Action file execution result. Fields: schema, status, dry_run, requires_confirmation, total, updated, skipped, errors.",
				"commands":    []string{"review --apply-file", "agent execute"},
			},
			{
				"name":        "taskbridge.action-audit.v1",
				"description": "Audit receipt for action execution attempts. Fields: schema_version, session_id, command, action_file, dry_run, confirm, status, started_at, finished_at, duration_ms, stats, operations, errors, redaction. Written by review --apply-file and agent execute.",
				"commands":    []string{"audit show", "audit list"},
			},
			{
				"name":        "taskbridge.sync-session.v1",
				"description": "Sync session audit record. Fields: schema, id, mode, source, target, dry_run, started_at, completed_at, status, stats, operations.",
				"commands":    []string{"sync diff", "sync audit"},
			},
			{
				"name":        "taskbridge.project-review.v1",
				"description": "Project review result. Fields: schema, project_id, status, suggested_actions[], risks[].",
				"commands":    []string{"project review"},
			},
		},
	}
	return printAgent(agentcontract.OK(requestID(), false, payload))
}

func printAgent(result agentcontract.Result) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return commandError("Failed to serialize Agent output", err)
	}
	fmt.Println(string(data))
	return nil
}

func effectiveAgentExecuteDryRun(dryRun, confirm bool) bool {
	if confirm {
		return false
	}
	return dryRun
}

func requestID() string {
	if agentRequestID != "" {
		return agentRequestID
	}
	return fmt.Sprintf("req_%s", time.Now().Format("20060102_150405"))
}
