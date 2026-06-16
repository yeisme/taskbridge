package actionexecution

import (
	"context"
	"fmt"
	"time"

	"github.com/yeisme/taskbridge/internal/actionaudit"
	"github.com/yeisme/taskbridge/internal/actionfile"
	"github.com/yeisme/taskbridge/internal/storage"
)

// Options describes one action file execution attempt.
type Options struct {
	SessionID      string
	Command        string
	ActionFilePath string
	DryRun         bool
	Confirm        bool
}

// Result contains the loaded action file, execution result, and audit receipt.
type Result struct {
	ActionFile    *actionfile.File
	Execution     actionfile.ExecuteResult
	Receipt       actionaudit.Receipt
	ReceiptSaved  bool
	AuditWriteErr error
}

// Service owns the Load -> Execute -> per-action outcomes -> audit receipt flow.
type Service struct {
	TaskStore  storage.Storage
	AuditStore *actionaudit.Store
	Now        func() time.Time
}

// ExecuteFile loads the action file from disk, executes it, and records an audit receipt.
func (s Service) ExecuteFile(ctx context.Context, opts Options) (*Result, error) {
	file, err := actionfile.Load(opts.ActionFilePath)
	if err != nil {
		return nil, err
	}
	return s.Execute(ctx, file, opts), nil
}

// Execute executes an already-loaded action file and records an audit receipt.
func (s Service) Execute(ctx context.Context, file *actionfile.File, opts Options) *Result {
	sessionID := opts.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("action_%s", s.now().Format("20060102_150405"))
	}
	receipt := actionaudit.Start(sessionID, opts.Command, opts.ActionFilePath, opts.DryRun, opts.Confirm)
	execution := actionfile.Executor{TaskStore: s.TaskStore}.Execute(ctx, file, actionfile.ExecuteOptions{DryRun: opts.DryRun, Confirm: opts.Confirm})

	receipt.Finish(receiptStatus(execution), Stats(execution), Operations(execution), execution.Errors)
	result := &Result{ActionFile: file, Execution: execution, Receipt: receipt}
	if s.AuditStore == nil {
		return result
	}
	if err := s.AuditStore.Save(receipt); err != nil {
		result.AuditWriteErr = err
		result.Execution.Status = "error"
		result.Execution.Errors = append(result.Execution.Errors, "audit receipt write failed: "+err.Error())
		result.Receipt.Errors = result.Execution.Errors
		result.Receipt.Stats.Errors = len(result.Execution.Errors)
		return result
	}
	result.ReceiptSaved = true
	return result
}

// Operations converts per-action outcomes into audit receipt operations.
func Operations(result actionfile.ExecuteResult) []actionaudit.Operation {
	operations := make([]actionaudit.Operation, 0, len(result.Actions))
	for _, outcome := range result.Actions {
		operations = append(operations, actionaudit.Operation{
			ActionID:  outcome.ActionID,
			Type:      outcome.Type,
			TaskID:    outcome.TaskID,
			ProjectID: outcome.ProjectID,
			Status:    outcome.Status,
			Reason:    outcome.Reason,
			Error:     outcome.Error,
			DryRun:    result.DryRun,
			Confirmed: !result.DryRun && outcome.Status == "applied",
		})
	}
	return operations
}

// Stats summarises an execution result for audit receipts.
func Stats(result actionfile.ExecuteResult) actionaudit.Stats {
	confirmed := 0
	for _, outcome := range result.Actions {
		if !result.DryRun && outcome.Status == "applied" {
			confirmed++
		}
	}
	return actionaudit.Stats{
		Total:     result.Total,
		Updated:   result.Updated,
		Skipped:   result.Skipped,
		Errors:    len(result.Errors),
		Confirmed: confirmed,
	}
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func receiptStatus(result actionfile.ExecuteResult) string {
	if result.RequiresConfirmation && result.Status == "ok" {
		return "requires_confirmation"
	}
	return result.Status
}
