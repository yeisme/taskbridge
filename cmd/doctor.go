package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/controlplane"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/pkg/ui"
)

var (
	doctorFormat     string
	quickstartFormat string
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the TaskBridge local environment",
	RunE:  runDoctor,
}

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Gives the next suggested command based on the current status",
	RunE:  runQuickstart,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(quickstartCmd)

	doctorCmd.Flags().StringVarP(&doctorFormat, "format", "f", "text", "Output format (text, json)")
	quickstartCmd.Flags().StringVarP(&quickstartFormat, "format", "f", "text", "Output format (text, json)")
}

func runDoctor(_ *cobra.Command, _ []string) error {
	result := buildDoctorResult()
	projection := buildDoctorProjection(result)
	return printProjectionWithLegacyJSON(doctorFormat, result, projection, func() { fmt.Print(renderDoctorResult(result, projection)) })
}

func runQuickstart(_ *cobra.Command, _ []string) error {
	result := buildDoctorResult()
	payload := map[string]interface{}{
		"schema":      "taskbridge.quickstart.v1",
		"status":      result.Status,
		"next_action": result.NextAction,
		"checks":      result.Checks,
	}
	projection := buildQuickstartProjection(result, payload)
	return printProjectionWithLegacyJSON(quickstartFormat, payload, projection, func() { fmt.Print(renderQuickstartResult(result, projection)) })
}

func buildDoctorProjection(result controlplane.DoctorResult) clioutput.Projection {
	p := clioutput.New("doctor.check")
	p.Summary = "TaskBridge doctor completed."
	p.Status = cliStatusFromString(result.Status)
	p.Facts["checks"] = len(result.Checks)
	ok, warnings, errors := 0, 0, 0
	for _, check := range result.Checks {
		switch check.Status {
		case "ok":
			ok++
		case "warning":
			warnings++
		case "error":
			errors++
		}
		if check.Status != "ok" {
			p.Risks = append(p.Risks, check.ID+": "+check.Message)
		}
	}
	p.Facts["ok"] = ok
	p.Facts["warnings"] = warnings
	p.Facts["errors"] = errors
	if result.NextAction != "" {
		p.Actions = append(p.Actions, clioutput.Action{Name: "next", Command: result.NextAction})
	}
	p.Data = result
	return p
}

func buildQuickstartProjection(result controlplane.DoctorResult, payload map[string]interface{}) clioutput.Projection {
	p := clioutput.New("doctor.quickstart")
	p.Summary = "Quickstart recommendation generated."
	p.Status = cliStatusFromString(result.Status)
	p.Facts["checks"] = len(result.Checks)
	if result.NextAction != "" {
		p.Actions = append(p.Actions, clioutput.Action{Name: "next", Command: result.NextAction})
	}
	p.Data = payload
	return p
}

func renderDoctorResult(result controlplane.DoctorResult, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Doctor checks\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if len(result.Checks) == 0 {
		b.WriteString("\nNo checks were run.\n")
		return renderRecommendedAction(b.String(), projection)
	}
	b.WriteString("\nCheck table\n")
	table := ui.NewTable("Check", "Status", "Message", "Next action")
	for _, check := range result.Checks {
		table.AddRow(check.ID, check.Status, check.Message, check.NextAction)
	}
	b.WriteString(table.Render())
	b.WriteString("\n")
	return renderRecommendedAction(b.String(), projection)
}

func renderQuickstartResult(result controlplane.DoctorResult, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Quickstart recommendation\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if len(result.Checks) > 0 {
		b.WriteString("\nChecks\n")
		table := ui.NewTable("Check", "Status", "Next action")
		for _, check := range result.Checks {
			table.AddRow(check.ID, check.Status, check.NextAction)
		}
		b.WriteString(table.Render())
		b.WriteString("\n")
	}
	return renderRecommendedAction(b.String(), projection)
}

func buildDoctorResult() controlplane.DoctorResult {
	checks := make([]controlplane.DoctorCheck, 0)
	status := "ok"
	add := func(check controlplane.DoctorCheck) {
		checks = append(checks, check)
		if check.Status == "error" {
			status = "error"
		} else if check.Status == "warning" && status == "ok" {
			status = "warning"
		}
	}

	if cfg == nil {
		add(controlplane.DoctorCheck{ID: "config", Status: "error", Message: "Configuration not initialized", NextAction: "Rerun taskbridge"})
		return controlplane.DoctorResult{Schema: controlplane.SchemaDoctor, Status: "error", Checks: checks, NextAction: "Rerun taskbridge"}
	}

	if cfg.Storage.Path == "" {
		add(controlplane.DoctorCheck{ID: "storage_path", Status: "error", Message: "storage path is empty", NextAction: "Set TASKBRIDGE_STORAGE_PATH"})
	} else if err := os.MkdirAll(cfg.Storage.Path, 0o755); err != nil {
		add(controlplane.DoctorCheck{ID: "storage_path", Status: "error", Message: err.Error(), NextAction: "Check storage path permissions"})
	} else if file, err := os.CreateTemp(cfg.Storage.Path, ".taskbridge-write-check-*"); err != nil {
		add(controlplane.DoctorCheck{ID: "storage_path", Status: "error", Message: err.Error(), NextAction: "Check storage path write permission"})
	} else {
		name := file.Name()
		_ = file.Close()
		_ = os.Remove(name)
		add(controlplane.DoctorCheck{ID: "storage_path", Status: "ok", Message: fmt.Sprintf("storage path is writable: %s", cfg.Storage.Path)})
	}

	if _, err := project.NewFileStore(cfg.Storage.Path); err != nil {
		add(controlplane.DoctorCheck{ID: "project_store", Status: "warning", Message: err.Error(), NextAction: "Check projects.json or storage path"})
	} else {
		add(controlplane.DoctorCheck{ID: "project_store", Status: "ok", Message: "project store is readable"})
	}

	authenticated := 0
	for _, name := range getAuthProviderOrder() {
		snapshot := getProviderAuthSnapshot(name)
		if snapshot.Authenticated {
			authenticated++
		}
	}
	if authenticated == 0 {
		add(controlplane.DoctorCheck{ID: "provider_auth", Status: "warning", Message: "no provider authenticated", NextAction: "taskbridge demo today"})
	} else {
		add(controlplane.DoctorCheck{ID: "provider_auth", Status: "ok", Message: fmt.Sprintf("%d provider(s) authenticated", authenticated)})
	}

	next := "taskbridge today"
	if status != "ok" || authenticated == 0 {
		next = "taskbridge demo today"
	}
	return controlplane.DoctorResult{Schema: controlplane.SchemaDoctor, Status: status, Checks: checks, NextAction: next}
}
