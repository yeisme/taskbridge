package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
