package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDemoTodayJSONParses verifies demo today --json (global flag) produces a
// single parseable JSON object on stdout with the AI-native envelope fields.
func TestDemoTodayJSONParses(t *testing.T) {
	controlFormat = ""
	outputJSON = true
	defer func() { outputJSON = false }()

	output := captureStdout(t, func() {
		if err := runDemoToday(nil, nil); err != nil {
			t.Fatalf("runDemoToday: %v", err)
		}
	})

	var envelope struct {
		SpecVersion string `json:"spec_version"`
		Mode        string `json:"mode"`
		Command     string `json:"command"`
		Status      string `json:"status"`
		Data        struct {
			Schema   string `json:"schema"`
			Sections []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"sections"`
			Summary map[string]int `json:"summary"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("demo today --json should produce parseable JSON, got error: %v\noutput:\n%s", err, output)
	}
	if envelope.SpecVersion != "1.0" {
		t.Fatalf("spec_version = %q, want 1.0", envelope.SpecVersion)
	}
	if envelope.Mode != "json" {
		t.Fatalf("mode = %q, want json", envelope.Mode)
	}
	if envelope.Command != "task.today" {
		t.Fatalf("command = %q, want task.today", envelope.Command)
	}
	if envelope.Data.Schema != "taskbridge.today.v1" {
		t.Fatalf("data.schema = %q, want taskbridge.today.v1", envelope.Data.Schema)
	}
	if len(envelope.Data.Sections) == 0 {
		t.Fatal("demo today should include at least one section")
	}
	if envelope.Data.Summary["must_do"] == 0 {
		t.Fatal("demo today should include must_do summary count")
	}
}

// TestDemoTodayHumanOutput verifies demo today default output is human-readable
// English text, not JSON.
func TestDemoTodayHumanOutput(t *testing.T) {
	controlFormat = ""
	outputJSON = false
	outputAgent = false

	output := captureStdout(t, func() {
		if err := runDemoToday(nil, nil); err != nil {
			t.Fatalf("runDemoToday: %v", err)
		}
	})

	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("demo today default should be human text, not JSON:\n%s", output)
	}
	if !strings.Contains(output, "Daily workbench") {
		t.Fatalf("demo today should show Daily workbench title:\n%s", output)
	}
}

// TestDemoTodayAgentOutput verifies demo today --agent produces stable key=value.
func TestDemoTodayAgentOutput(t *testing.T) {
	controlFormat = ""
	outputAgent = true
	defer func() { outputAgent = false }()

	output := captureStdout(t, func() {
		if err := runDemoToday(nil, nil); err != nil {
			t.Fatalf("runDemoToday: %v", err)
		}
	})

	for _, want := range []string{"spec_version=", "mode=agent", "command=task.today", "status="} {
		if !strings.Contains(output, want) {
			t.Fatalf("demo today --agent should contain %q:\n%s", want, output)
		}
	}
}
