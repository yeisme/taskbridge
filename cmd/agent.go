package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/actionfile"
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
	Short: "Agent 安全执行入口",
}

var agentCapabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "输出 Agent 可调用能力",
	RunE:  runAgentCapabilities,
}

var agentTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "输出 Agent 友好的 today",
	RunE:  runAgentToday,
}

var agentPlanCmd = &cobra.Command{
	Use:   "plan <goal>",
	Short: "生成目标计划预览",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentPlan,
}

var agentExecuteCmd = &cobra.Command{
	Use:   "execute",
	Short: "执行 action file",
	RunE:  runAgentExecute,
}

var agentSchemasCmd = &cobra.Command{
	Use:   "schemas",
	Short: "输出 Agent JSON schema 名称",
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
		cmd.Flags().StringVar(&agentRequestID, "request-id", "", "请求 ID")
	}
	agentPlanCmd.Flags().BoolVar(&agentDryRun, "dry-run", true, "模拟执行")
	agentPlanCmd.Flags().IntVar(&agentHorizonDays, "horizon-days", 14, "规划周期天数")
	agentExecuteCmd.Flags().StringVar(&agentActionFile, "action-file", "", "action file 路径")
	agentExecuteCmd.Flags().BoolVar(&agentDryRun, "dry-run", true, "模拟执行")
	agentExecuteCmd.Flags().BoolVar(&agentConfirm, "confirm", false, "确认执行危险动作")
	_ = agentExecuteCmd.MarkFlagRequired("action-file")
}

func runAgentCapabilities(_ *cobra.Command, _ []string) error {
	payload := map[string]interface{}{
		"schema":            "taskbridge.agent-capabilities.v1",
		"providers":         getAuthProviderOrder(),
		"commands":          []string{"agent today", "agent plan", "agent execute", "agent schemas", "today", "next", "review", "sync diff"},
		"schema_versions":   []string{"taskbridge.agent-result.v1", "taskbridge.today.v1", "taskbridge.actions.v1"},
		"dangerous_actions": []string{"delete_task", "complete_task", "defer_task", "reschedule_task", "remote_write", "conflict_discard"},
	}
	return printAgent(agentcontract.OK(requestID(), false, payload))
}

func runAgentToday(_ *cobra.Command, _ []string) error {
	ctx := context.Background()
	service, cleanup, err := controlService()
	defer cleanup()
	if err != nil {
		return printAgent(agentcontract.Error(requestID(), "store_init_failed", err.Error(), "taskbridge doctor"))
	}
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
			return printAgent(agentcontract.Error(requestID(), "agent_plan_failed", err.Error(), "检查目标文本后重试"))
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
	file, err := actionfile.Load(agentActionFile)
	if err != nil {
		return printAgent(agentcontract.Error(requestID(), "action_file_invalid", err.Error(), "检查 action file"))
	}
	taskStore, _, cleanup, err := getCLIStores()
	defer cleanup()
	if err != nil {
		return printAgent(agentcontract.Error(requestID(), "store_init_failed", err.Error(), "taskbridge doctor"))
	}
	effectiveDryRun := effectiveAgentExecuteDryRun(agentDryRun, agentConfirm)
	result := actionfile.Executor{TaskStore: taskStore}.Execute(context.Background(), file, actionfile.ExecuteOptions{DryRun: effectiveDryRun, Confirm: agentConfirm})
	envelope := agentcontract.OK(requestID(), effectiveDryRun, result)
	if result.RequiresConfirmation {
		envelope = agentcontract.Confirmation(requestID(), effectiveDryRun, result)
	}
	if result.Status == "error" {
		envelope.Status = "error"
	}
	return printAgent(envelope)
}

func runAgentSchemas(_ *cobra.Command, _ []string) error {
	payload := map[string]interface{}{
		"schemas": []string{
			"taskbridge.agent-result.v1",
			"taskbridge.agent-capabilities.v1",
			"taskbridge.today.v1",
			"taskbridge.actions.v1",
			"taskbridge.action-result.v1",
			"taskbridge.sync-session.v1",
			"taskbridge.project-review.v1",
		},
	}
	return printAgent(agentcontract.OK(requestID(), false, payload))
}

func printAgent(result agentcontract.Result) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return commandError("序列化 Agent 输出失败", err)
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
