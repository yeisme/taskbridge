package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/actionaudit"
	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/pkg/ui"
)

var auditFormat string

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Inspect action execution audit receipts",
}

var auditShowCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show a single audit receipt",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuditShow,
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent audit receipts",
	RunE:  runAuditList,
}

func init() {
	rootCmd.AddCommand(auditCmd)
	auditCmd.AddCommand(auditShowCmd)
	auditCmd.AddCommand(auditListCmd)

	auditCmd.PersistentFlags().StringVarP(&auditFormat, "format", "f", "text", "Output format (text, json)")
}

func runAuditShow(_ *cobra.Command, args []string) error {
	store := actionaudit.NewStore(cfg.Storage.Path)
	receipt, err := store.Load(args[0])
	if err != nil {
		return commandError("Failed to load audit receipt", err)
	}

	projection := buildAuditReceiptProjection(receipt)
	return printProjectionWithLegacyJSON(auditFormat, receipt, projection, func() {
		fmt.Print(renderAuditReceipt(receipt, projection))
	})
}

func runAuditList(_ *cobra.Command, _ []string) error {
	store := actionaudit.NewStore(cfg.Storage.Path)
	summaries, err := store.List()
	if err != nil {
		return commandError("Failed to list audit receipts", err)
	}

	projection := buildAuditListProjection(summaries)
	return printProjectionWithLegacyJSON(auditFormat, summaries, projection, func() {
		fmt.Print(renderAuditList(summaries, projection))
	})
}

func buildAuditReceiptProjection(receipt *actionaudit.Receipt) clioutput.Projection {
	p := clioutput.New("audit.show")
	p.Summary = "Audit receipt loaded."
	if receipt != nil {
		p.Status = cliStatusFromString(receipt.Status)
		p.Facts["session_id"] = receipt.SessionID
		p.Facts["command"] = receipt.Command
		p.Facts["dry_run"] = receipt.DryRun
		p.Facts["confirm"] = receipt.Confirm
		p.Facts["total"] = receipt.Stats.Total
		p.Facts["updated"] = receipt.Stats.Updated
		p.Facts["errors"] = receipt.Stats.Errors
		if len(receipt.Errors) > 0 {
			p.Risks = append(p.Risks, receipt.Errors...)
		}
	}
	p.Data = receipt
	return p
}

func buildAuditListProjection(summaries []actionaudit.ReceiptSummary) clioutput.Projection {
	p := clioutput.New("audit.list")
	p.Summary = "Audit receipts listed."
	p.Facts["count"] = len(summaries)
	p.Data = summaries
	return p
}

func renderAuditReceipt(receipt *actionaudit.Receipt, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Audit receipt\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if receipt != nil && len(receipt.Operations) > 0 {
		b.WriteString("\nOperations\n")
		table := ui.NewTable("Action", "Type", "Task", "Status", "Dry run", "Confirmed")
		for _, op := range receipt.Operations {
			table.AddRow(op.ActionID, op.Type, op.TaskID, op.Status, fmt.Sprint(op.DryRun), fmt.Sprint(op.Confirmed))
		}
		b.WriteString(table.Render())
		b.WriteString("\n")
	}
	if receipt != nil && len(receipt.Errors) > 0 {
		b.WriteString("\nErrors\n")
		for _, e := range receipt.Errors {
			b.WriteString("  " + e + "\n")
		}
	}
	return renderRecommendedAction(b.String(), projection)
}

func renderAuditList(summaries []actionaudit.ReceiptSummary, projection clioutput.Projection) string {
	var b strings.Builder
	b.WriteString("Audit receipts\n\n")
	b.WriteString(renderProjectionFacts(projection))
	if len(summaries) == 0 {
		b.WriteString("\nNo audit receipts found.\n")
		return renderRecommendedAction(b.String(), projection)
	}
	b.WriteString("\nReceipts\n")
	table := ui.NewTable("Session", "Command", "Status", "Dry run", "Confirm", "Total", "Updated")
	for _, s := range summaries {
		table.AddRow(s.SessionID, s.Command, s.Status, fmt.Sprint(s.DryRun), fmt.Sprint(s.Confirm), fmt.Sprint(s.Total), fmt.Sprint(s.Updated))
	}
	b.WriteString(table.Render())
	b.WriteString("\n")
	return renderRecommendedAction(b.String(), projection)
}

// suppress unused import when no direct json use
var _ = json.Marshal
