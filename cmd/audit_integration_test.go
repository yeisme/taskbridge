package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
	"github.com/yeisme/taskbridge/pkg/config"
)

func setupTestConfig(t *testing.T, storagePath string) {
	t.Helper()
	cfg = &config.Config{
		Storage: config.StorageConfig{Path: storagePath},
	}
}

// writes an audit receipt even when actions fail.
func TestAgentExecuteWritesAuditReceipt(t *testing.T) {
	tmp := t.TempDir()
	storagePath := filepath.Join(tmp, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatal(err)
	}

	actionsFile := filepath.Join(tmp, "actions.json")
	actions := `{"schema":"taskbridge.actions.v1","actions":[{"id":"a1","type":"defer_task","task_id":"nonexistent","reason":"test"}]}`
	if err := os.WriteFile(actionsFile, []byte(actions), 0o644); err != nil {
		t.Fatal(err)
	}

	setupTestConfig(t, storagePath)

	agentActionFile = actionsFile
	agentDryRun = true
	agentConfirm = false
	defer func() {
		agentActionFile = ""
		agentDryRun = false
	}()

	output := captureStdout(t, func() {
		_ = runAgentExecute(nil, nil)
	})

	var envelope struct {
		Schema    string `json:"schema"`
		Status    string `json:"status"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("agent execute should output valid JSON:\n%s\nerror: %v", output, err)
	}
	if envelope.RequestID == "" {
		t.Fatal("agent execute should include request_id")
	}

	// Verify receipt file exists
	receiptPath := filepath.Join(storagePath, "audit", "actions", envelope.RequestID+".json")
	data, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("audit receipt should exist at %s: %v", receiptPath, err)
	}

	var receipt struct {
		SchemaVersion string `json:"schema_version"`
		SessionID     string `json:"session_id"`
		Command       string `json:"command"`
		DryRun        bool   `json:"dry_run"`
		Status        string `json:"status"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("receipt should be valid JSON: %v", err)
	}
	if receipt.SchemaVersion != "taskbridge.action-audit.v1" {
		t.Fatalf("schema_version = %q, want taskbridge.action-audit.v1", receipt.SchemaVersion)
	}
	if receipt.SessionID != envelope.RequestID {
		t.Fatalf("receipt session_id = %q, want %q", receipt.SessionID, envelope.RequestID)
	}
	if receipt.Command != "agent execute" {
		t.Fatalf("command = %q, want 'agent execute'", receipt.Command)
	}
	if !receipt.DryRun {
		t.Fatal("dry_run should be true")
	}
}

// TestAgentExecuteErrorReturnsNonZeroExit verifies that agent execute returns
// a non-zero exit code when execution fails, while stdout remains valid JSON.
func TestAgentExecuteErrorReturnsNonZeroExit(t *testing.T) {
	tmp := t.TempDir()
	storagePath := filepath.Join(tmp, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatal(err)
	}

	actionsFile := filepath.Join(tmp, "actions.json")
	actions := `{"schema":"taskbridge.actions.v1","actions":[{"id":"a1","type":"complete_task","task_id":"doesnotexist","reason":"test"}]}`
	if err := os.WriteFile(actionsFile, []byte(actions), 0o644); err != nil {
		t.Fatal(err)
	}

	setupTestConfig(t, storagePath)

	agentActionFile = actionsFile
	agentDryRun = true
	agentConfirm = false
	defer func() {
		agentActionFile = ""
		agentDryRun = false
	}()

	_ = captureStdout(t, func() {})

	err := runAgentExecute(nil, nil)
	if err == nil {
		t.Fatal("agent execute with failed actions should return non-nil error")
	}

	var cliErr *CLIError
	if !errorAs(err, &cliErr) {
		t.Fatalf("error should be *CLIError, got %T", err)
	}
	if cliErr.ExitCode == 0 {
		t.Fatal("exit code should be non-zero on execution failure")
	}
}

func TestAgentExecuteAuditOperationsFollowPerActionOutcomes(t *testing.T) {
	tmp := t.TempDir()
	storagePath := filepath.Join(tmp, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := filestore.New(storagePath, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Now()
	if err := store.SaveTask(context.Background(), &model.Task{ID: "task-ok", Title: "ok", Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	actionsFile := filepath.Join(tmp, "actions.json")
	actions := `{"schema":"taskbridge.actions.v1","actions":[{"id":"act-ok","type":"complete_task","task_id":"task-ok","reason":"ok"},{"id":"act-missing","type":"complete_task","task_id":"missing","reason":"missing"}]}`
	if err := os.WriteFile(actionsFile, []byte(actions), 0o644); err != nil {
		t.Fatal(err)
	}

	setupTestConfig(t, storagePath)
	agentActionFile = actionsFile
	agentDryRun = true
	agentConfirm = true
	agentRequestID = "req_outcome_mapping"
	defer func() {
		agentActionFile = ""
		agentDryRun = false
		agentConfirm = false
		agentRequestID = ""
	}()

	_ = captureStdout(t, func() {
		_ = runAgentExecute(nil, nil)
	})

	receiptPath := filepath.Join(storagePath, "audit", "actions", "req_outcome_mapping.json")
	data, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("audit receipt should exist at %s: %v", receiptPath, err)
	}
	var receipt struct {
		Operations []struct {
			ActionID string `json:"action_id"`
			Status   string `json:"status"`
			Error    string `json:"error"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("receipt should be valid JSON: %v", err)
	}
	if len(receipt.Operations) != 2 {
		t.Fatalf("operations length = %d, want 2: %+v", len(receipt.Operations), receipt.Operations)
	}
	if receipt.Operations[0].ActionID != "act-ok" || receipt.Operations[0].Status != "applied" || receipt.Operations[0].Error != "" {
		t.Fatalf("first operation = %+v, want applied act-ok without error", receipt.Operations[0])
	}
	if receipt.Operations[1].ActionID != "act-missing" || receipt.Operations[1].Status != "failed" || receipt.Operations[1].Error == "" {
		t.Fatalf("second operation = %+v, want failed act-missing with error", receipt.Operations[1])
	}
}

func TestProjectAdjustConfirmUsesFacadeOutcomes(t *testing.T) {
	tmp := t.TempDir()
	storagePath := filepath.Join(tmp, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatal(err)
	}
	setupTestConfig(t, storagePath)

	projectStore, err := project.NewFileStore(storagePath)
	if err != nil {
		t.Fatalf("project.NewFileStore: %v", err)
	}
	ctx := context.Background()
	if err := projectStore.SaveProject(ctx, &project.Project{ID: "proj-1", Name: "Project", GoalText: "Project", GoalType: project.GoalTypeGeneric, Status: project.StatusActive}); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}
	taskStore, err := filestore.New(storagePath, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Now()
	if err := taskStore.SaveTask(ctx, &model.Task{ID: "task-large", Title: "large", Status: model.StatusTodo, Source: model.SourceLocal, EstimatedMinutes: 240, CreatedAt: now, UpdatedAt: now, Metadata: &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{"tb_project_id": "proj-1"}}}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	if err := taskStore.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	projectAdjustConfirm = true
	projectFormat = "json"
	defer func() {
		projectAdjustConfirm = false
		projectFormat = "text"
	}()

	output := captureStdout(t, func() {
		err = runProjectAdjust(nil, []string{"proj-1"})
	})
	if err != nil {
		t.Fatalf("project adjust --confirm: %v", err)
	}
	var result struct {
		Status  string `json:"status"`
		Actions []struct {
			ActionID string `json:"action_id"`
			Status   string `json:"status"`
			Error    string `json:"error"`
		} `json:"actions"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("project adjust should emit machine-readable JSON:\n%s\nerror: %v", output, err)
	}
	if result.Status != "ok" || len(result.Actions) != 1 || result.Actions[0].Status != "applied" || result.Actions[0].Error != "" {
		t.Fatalf("unexpected project adjust result: %+v", result)
	}
}

// TestAuditShowReadsReceipt verifies audit show reads a previously written receipt.
func TestAuditShowReadsReceipt(t *testing.T) {
	tmp := t.TempDir()
	storagePath := filepath.Join(tmp, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatal(err)
	}

	setupTestConfig(t, storagePath)

	// First write a receipt via agent execute
	actionsFile := filepath.Join(tmp, "actions.json")
	actions := `{"schema":"taskbridge.actions.v1","actions":[{"id":"a1","type":"defer_task","task_id":"t1","reason":"test"}]}`
	if err := os.WriteFile(actionsFile, []byte(actions), 0o644); err != nil {
		t.Fatal(err)
	}

	agentActionFile = actionsFile
	agentDryRun = true
	defer func() {
		agentActionFile = ""
		agentDryRun = false
	}()

	output := captureStdout(t, func() { _ = runAgentExecute(nil, nil) })

	var env struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatal(err)
	}

	// Now audit show should read it
	auditFormat = "json"
	defer func() { auditFormat = "" }()

	showOutput := captureStdout(t, func() {
		if err := runAuditShow(nil, []string{env.RequestID}); err != nil {
			t.Fatalf("audit show: %v", err)
		}
	})

	var showResult struct {
		SessionID string `json:"session_id"`
		Command   string `json:"command"`
	}
	if err := json.Unmarshal([]byte(showOutput), &showResult); err != nil {
		t.Fatalf("audit show --format json should produce legacy JSON:\n%s\nerror: %v", showOutput, err)
	}
	if showResult.SessionID != env.RequestID {
		t.Fatalf("session_id = %q, want %q", showResult.SessionID, env.RequestID)
	}
}

// TestAuditListShowsReceipts verifies audit list shows previously written receipts.
func TestAuditListShowsReceipts(t *testing.T) {
	tmp := t.TempDir()
	storagePath := filepath.Join(tmp, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatal(err)
	}

	setupTestConfig(t, storagePath)

	// Write a receipt
	actionsFile := filepath.Join(tmp, "actions.json")
	actions := `{"schema":"taskbridge.actions.v1","actions":[{"id":"a1","type":"defer_task","task_id":"t1","reason":"test"}]}`
	if err := os.WriteFile(actionsFile, []byte(actions), 0o644); err != nil {
		t.Fatal(err)
	}

	agentActionFile = actionsFile
	agentDryRun = true
	defer func() {
		agentActionFile = ""
		agentDryRun = false
	}()

	_ = captureStdout(t, func() { _ = runAgentExecute(nil, nil) })

	// List should show it
	auditFormat = "json"
	defer func() { auditFormat = "" }()

	listOutput := captureStdout(t, func() {
		if err := runAuditList(nil, nil); err != nil {
			t.Fatalf("audit list: %v", err)
		}
	})

	var summaries []struct {
		SessionID string `json:"session_id"`
		Command   string `json:"command"`
	}
	if err := json.Unmarshal([]byte(listOutput), &summaries); err != nil {
		t.Fatalf("audit list --format json should produce legacy JSON array:\n%s\nerror: %v", listOutput, err)
	}
	if len(summaries) == 0 {
		t.Fatal("audit list should show at least one receipt")
	}
	if summaries[0].Command != "agent execute" {
		t.Fatalf("command = %q, want 'agent execute'", summaries[0].Command)
	}
}

// errorAs wraps errors.As for test usage
func errorAs(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	if ce, ok := err.(*CLIError); ok {
		if ptr, ok := target.(**CLIError); ok {
			*ptr = ce
			return true
		}
	}
	return false
}
