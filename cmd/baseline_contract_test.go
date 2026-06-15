package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBaselineTodayJSONStructure verifies the current baseline structure of
// today --json. This is a RED baseline: if the structure changes, this test
// must be updated to reflect the new contract.
func TestBaselineTodayJSONStructure(t *testing.T) {
	controlFormat = ""
	controlMock = true
	outputJSON = true
	defer func() {
		controlFormat = ""
		controlMock = false
		outputJSON = false
	}()

	output := captureStdout(t, func() {
		if err := runToday(nil, nil); err != nil {
			t.Fatalf("runToday: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("today --json baseline must be parseable JSON:\n%s", output)
	}
	for _, field := range []string{"spec_version", "mode", "command", "status"} {
		if _, ok := envelope[field]; !ok {
			t.Fatalf("baseline today --json missing %q", field)
		}
	}
	if envelope["spec_version"] != "1.0" {
		t.Fatalf("baseline spec_version = %v, want 1.0", envelope["spec_version"])
	}
}

// TestBaselineNextAgentStructure verifies the current baseline structure of
// next --agent.
func TestBaselineNextAgentStructure(t *testing.T) {
	controlFormat = ""
	controlMock = true
	outputAgent = true
	defer func() {
		controlFormat = ""
		controlMock = false
		outputAgent = false
	}()

	output := captureStdout(t, func() {
		if err := runNext(nil, nil); err != nil {
			t.Fatalf("runNext: %v", err)
		}
	})

	for _, want := range []string{"spec_version=1.0", "mode=agent"} {
		if !strings.Contains(output, want) {
			t.Fatalf("baseline next --agent missing %q:\n%s", want, output)
		}
	}
}

// TestBaselineReviewJSONStructure verifies the current baseline structure of
// review --json.
func TestBaselineReviewJSONStructure(t *testing.T) {
	controlFormat = ""
	controlMock = true
	outputJSON = true
	defer func() {
		controlFormat = ""
		controlMock = false
		outputJSON = false
	}()

	output := captureStdout(t, func() {
		if err := runReview(nil, nil); err != nil {
			t.Fatalf("runReview: %v", err)
		}
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("review --json baseline must be parseable JSON:\n%s", output)
	}
	for _, field := range []string{"spec_version", "mode", "command", "status"} {
		if _, ok := envelope[field]; !ok {
			t.Fatalf("baseline review --json missing %q", field)
		}
	}
}

// TestBaselineAgentExecuteDryRun verifies agent execute --dry-run baseline
// returns a parseable JSON envelope even when actions fail.
func TestBaselineAgentExecuteDryRun(t *testing.T) {
	tmp := t.TempDir()
	storagePath := filepath.Join(tmp, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatal(err)
	}

	actionsFile := filepath.Join(tmp, "actions.json")
	body := `{"schema":"taskbridge.actions.v1","actions":[{"id":"b1","type":"defer_task","task_id":"baseline-missing","reason":"RED baseline"}]}`
	if err := os.WriteFile(actionsFile, []byte(body), 0o644); err != nil {
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
		Schema string `json:"schema"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("baseline agent execute --dry-run must output valid JSON:\n%s", output)
	}
	if envelope.Schema != "taskbridge.agent-result.v1" {
		t.Fatalf("baseline schema = %q, want taskbridge.agent-result.v1", envelope.Schema)
	}
}

// TestBaselineMachineOutputNoANSI verifies all machine-mode outputs are ANSI-free.
func TestBaselineMachineOutputNoANSI(t *testing.T) {
	controlFormat = ""
	controlMock = true
	outputJSON = true
	jsonOut := captureStdout(t, func() {
		_ = runToday(nil, nil)
	})
	if strings.Contains(jsonOut, "\x1b[") {
		t.Fatalf("today --json must not contain ANSI:\n%s", jsonOut)
	}
	outputJSON = false

	outputAgent = true
	agentOut := captureStdout(t, func() {
		_ = runToday(nil, nil)
	})
	if strings.Contains(agentOut, "\x1b[") {
		t.Fatalf("today --agent must not contain ANSI:\n%s", agentOut)
	}
	outputAgent = false
	controlMock = false
}
