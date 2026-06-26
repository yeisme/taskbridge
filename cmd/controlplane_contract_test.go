package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// runControlplaneHelper sets up flags and runs a command function, returning stdout.
func runControlplaneHelper(t *testing.T, jsonFlag, agentFlag bool, runFn func() error) string {
	t.Helper()
	controlFormat = ""
	controlMock = true
	outputJSON = jsonFlag
	outputAgent = agentFlag
	defer func() {
		controlMock = false
		outputJSON = false
		outputAgent = false
	}()

	return captureStdout(t, func() {
		if err := runFn(); err != nil {
			t.Fatalf("command failed: %v", err)
		}
	})
}

// TestTodayJSONContract verifies today --json produces a parseable AI-native envelope.
func TestTodayJSONContract(t *testing.T) {
	output := runControlplaneHelper(t, true, false, func() error {
		return runToday(nil, nil)
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("today --json must produce a single parseable JSON object:\n%s\nerror: %v", output, err)
	}
	for _, field := range []string{"spec_version", "mode", "command", "status", "summary", "facts", "data"} {
		if _, ok := envelope[field]; !ok {
			t.Fatalf("today --json envelope missing field %q:\n%s", field, output)
		}
	}
	if envelope["mode"] != "json" {
		t.Fatalf("mode = %v, want json", envelope["mode"])
	}
	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("today --json data missing or wrong type: %#v", envelope["data"])
	}
	sections, ok := data["sections"].([]interface{})
	if !ok {
		t.Fatalf("today --json sections missing: %#v", data)
	}
	seen := map[string]bool{}
	for _, raw := range sections {
		section, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := section["id"].(string)
		seen[id] = true
		if tasks, ok := section["tasks"].([]interface{}); ok && len(tasks) > 0 {
			task, _ := tasks[0].(map[string]interface{})
			if _, ok := task["source"]; !ok {
				t.Fatalf("task entry missing source metadata: %#v", task)
			}
			if _, ok := task["domain"]; !ok {
				t.Fatalf("task entry missing domain metadata: %#v", task)
			}
		}
	}
	for _, id := range []string{"work", "life", "inbox", "overdue", "recommended_next", "sync_warnings"} {
		if !seen[id] {
			t.Fatalf("today --json missing section %q in %#v", id, seen)
		}
	}
}

// TestTodayAgentContract verifies today --agent produces stable key=value.
func TestTodayAgentContract(t *testing.T) {
	output := runControlplaneHelper(t, false, true, func() error {
		return runToday(nil, nil)
	})

	for _, want := range []string{"spec_version=", "mode=agent", "command=task.today", "status="} {
		if !strings.Contains(output, want) {
			t.Fatalf("today --agent must contain %q:\n%s", want, output)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("today --agent must be key=value, not JSON:\n%s", output)
	}
}

// TestTodayHumanContract verifies today default output is human-readable English.
func TestTodayHumanContract(t *testing.T) {
	output := runControlplaneHelper(t, false, false, func() error {
		return runToday(nil, nil)
	})

	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("today default must be human text, not JSON:\n%s", output)
	}
	for _, disallowed := range []string{"今日", "即将", "建议", "当前", "推进"} {
		if strings.Contains(output, disallowed) {
			t.Fatalf("today default must be English, found %q:\n%s", disallowed, output)
		}
	}
}

// TestNextJSONContract verifies next --json produces a parseable AI-native envelope.
func TestNextJSONContract(t *testing.T) {
	output := runControlplaneHelper(t, true, false, func() error {
		return runNext(nil, nil)
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("next --json must produce a single parseable JSON object:\n%s\nerror: %v", output, err)
	}
	for _, field := range []string{"spec_version", "mode", "command", "status"} {
		if _, ok := envelope[field]; !ok {
			t.Fatalf("next --json envelope missing field %q:\n%s", field, output)
		}
	}
}

// TestNextAgentContract verifies next --agent produces stable key=value.
func TestNextAgentContract(t *testing.T) {
	output := runControlplaneHelper(t, false, true, func() error {
		return runNext(nil, nil)
	})

	for _, want := range []string{"spec_version=", "mode=agent", "status=", "fact.recommendation.1.source=", "fact.recommendation.1.domain="} {
		if !strings.Contains(output, want) {
			t.Fatalf("next --agent must contain %q:\n%s", want, output)
		}
	}
}

// TestReviewJSONContract verifies review --json produces a parseable AI-native envelope.
func TestReviewJSONContract(t *testing.T) {
	output := runControlplaneHelper(t, true, false, func() error {
		return runReview(nil, nil)
	})

	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("review --json must produce a single parseable JSON object:\n%s\nerror: %v", output, err)
	}
	for _, field := range []string{"spec_version", "mode", "command", "status"} {
		if _, ok := envelope[field]; !ok {
			t.Fatalf("review --json envelope missing field %q:\n%s", field, output)
		}
	}
}

// TestReviewAgentContract verifies review --agent produces stable key=value.
func TestReviewAgentContract(t *testing.T) {
	output := runControlplaneHelper(t, false, true, func() error {
		return runReview(nil, nil)
	})

	for _, want := range []string{"spec_version=", "mode=agent", "status="} {
		if !strings.Contains(output, want) {
			t.Fatalf("review --agent must contain %q:\n%s", want, output)
		}
	}
}

// TestControlplaneNoANSIInMachineOutput verifies JSON and agent modes produce no ANSI codes.
func TestControlplaneNoANSIInMachineOutput(t *testing.T) {
	jsonOut := runControlplaneHelper(t, true, false, func() error {
		return runToday(nil, nil)
	})
	if strings.Contains(jsonOut, "\x1b[") {
		t.Fatalf("today --json must not contain ANSI escape codes:\n%s", jsonOut)
	}

	agentOut := runControlplaneHelper(t, false, true, func() error {
		return runToday(nil, nil)
	})
	if strings.Contains(agentOut, "\x1b[") {
		t.Fatalf("today --agent must not contain ANSI escape codes:\n%s", agentOut)
	}
}
