package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/yeisme/taskbridge/internal/clioutput"
)

// TestWantsJSON_FormatJSON verifies wantsJSON returns true for "json" format.
// Note: In test environments, stdout may be a pipe, which also triggers JSON mode.
func TestWantsJSON_FormatJSON(t *testing.T) {
	// "json" explicitly should always be true
	if !wantsJSON("json") {
		t.Error(`wantsJSON("json") = false, want true`)
	}
	if !wantsJSON("JSON") {
		t.Error(`wantsJSON("JSON") = false, want true`)
	}
	if !wantsJSON("  json  ") {
		t.Error(`wantsJSON("  json  ") = false, want true`)
	}
}

// TestWantsJSON_NonJSONFormat verifies that default human formats stay human even
// when stdout is captured by tests; scripts should request --json explicitly.
func TestWantsJSON_NonJSONFormat(t *testing.T) {
	oldQuiet := quiet
	quiet = false
	defer func() { quiet = oldQuiet }()
	if wantsJSON("text") {
		t.Fatal(`wantsJSON("text") = true, want false`)
	}
}

func TestWantsJSON_QuietMode(t *testing.T) {
	oldQuiet := quiet
	quiet = true
	defer func() { quiet = oldQuiet }()
	if !wantsJSON("text") {
		t.Fatal(`wantsJSON("text") in quiet mode = false, want true`)
	}
}

// TestPrintStructured_JSONMode verifies JSON mode outputs valid JSON to stdout.
func TestPrintStructured_JSONMode(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	err := printStructured("json", data, func() {
		// This should not be called in JSON mode
		t.Error("renderText called in JSON mode")
	})

	// Restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	output := buf.String()

	if err != nil {
		t.Fatalf("printStructured error: %v", err)
	}

	// Output must be valid JSON
	var parsed map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, output)
	}
	if parsed["key"] != "value" {
		t.Errorf("parsed[\"key\"] = %q, want %q", parsed["key"], "value")
	}
}

func TestPrintProjection_JSONEnvelope(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p := clioutput.New("doctor.check")
	p.Summary = "ok"
	err := printProjection("json", p, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if err != nil {
		t.Fatalf("printProjection error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, buf.String())
	}
	if parsed["mode"] != "json" || parsed["command"] != "doctor.check" || parsed["status"] != "success" {
		t.Fatalf("unexpected projection envelope: %#v", parsed)
	}
}

func TestPrintProjection_AgentMode(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p := clioutput.New("doctor.check")
	err := printProjection("agent", p, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if err != nil {
		t.Fatalf("printProjection error: %v", err)
	}
	output := buf.String()
	for _, want := range []string{"spec_version=1.0", "mode=agent", "command=doctor.check", "status=success"} {
		if !strings.Contains(output, want) {
			t.Fatalf("agent output missing %q:\n%s", want, output)
		}
	}
}

func TestErrorProjectionUsesStableCode(t *testing.T) {
	p := errorProjection("provider.info", usageError("invalid provider"))
	if p.Status != clioutput.StatusFailed {
		t.Fatalf("Status = %q, want failed", p.Status)
	}
	if p.Error == nil || p.Error.Code != "usage_error" || p.Error.Message != "invalid provider" {
		t.Fatalf("unexpected error projection: %#v", p.Error)
	}
}

func TestErrorProjectionCommandError(t *testing.T) {
	p := errorProjection("storage.open", commandError("storage failed", os.ErrNotExist))
	if p.Error == nil || p.Error.Code != "command_error" {
		t.Fatalf("unexpected error projection: %#v", p.Error)
	}
	if !strings.Contains(p.Error.Message, "storage failed") {
		t.Fatalf("message should include command error: %#v", p.Error)
	}
}

func TestResolveOutputFormat_GlobalFlagsOverrideLegacyFormat(t *testing.T) {
	oldJSON, oldAgent, oldEvents, oldExplain := outputJSON, outputAgent, outputEvents, outputExplain
	defer func() {
		outputJSON, outputAgent, outputEvents, outputExplain = oldJSON, oldAgent, oldEvents, oldExplain
	}()

	outputJSON, outputAgent, outputEvents, outputExplain = true, false, false, false
	if got := resolveOutputFormat("text"); got != "json" {
		t.Fatalf("resolveOutputFormat with --json = %q, want json", got)
	}

	outputJSON, outputAgent, outputEvents, outputExplain = false, true, false, false
	if got := resolveOutputFormat("json"); got != "agent" {
		t.Fatalf("resolveOutputFormat with --agent = %q, want agent", got)
	}
}

func TestResolveOutputFormat_DefaultsToLegacyFormat(t *testing.T) {
	oldJSON, oldAgent, oldEvents, oldExplain := outputJSON, outputAgent, outputEvents, outputExplain
	defer func() {
		outputJSON, outputAgent, outputEvents, outputExplain = oldJSON, oldAgent, oldEvents, oldExplain
	}()
	outputJSON, outputAgent, outputEvents, outputExplain = false, false, false, false

	if got := resolveOutputFormat("table"); got != "table" {
		t.Fatalf("resolveOutputFormat = %q, want table", got)
	}
}

func TestPrintProjectionRejectsUnsupportedEventsMode(t *testing.T) {
	oldEvents := outputEvents
	outputEvents = true
	defer func() { outputEvents = oldEvents }()

	err := printProjection("text", clioutput.New("doctor.check"), nil)
	if err == nil {
		t.Fatal("expected unsupported events mode error")
	}
	if !strings.Contains(err.Error(), "--events") {
		t.Fatalf("events error should mention --events: %v", err)
	}
}

func TestPrintProjectionWithLegacyJSONRoutesAllGlobalModes(t *testing.T) {
	oldJSON, oldAgent, oldEvents, oldExplain := outputJSON, outputAgent, outputEvents, outputExplain
	defer func() {
		outputJSON, outputAgent, outputEvents, outputExplain = oldJSON, oldAgent, oldEvents, oldExplain
	}()
	outputEvents = true
	err := printProjectionWithLegacyJSON("table", map[string]any{"legacy": true}, clioutput.New("project.list"), func() {})
	if err == nil || !strings.Contains(err.Error(), "--events") {
		t.Fatalf("expected global --events to route through projection dispatcher, got %v", err)
	}
}

// TestCommandError_ExitCode verifies commandError returns non-zero exit code.
func TestCommandError_ExitCode(t *testing.T) {
	err := commandError("test", nil)
	if err == nil {
		t.Fatal("commandError returned nil")
	}
	code := cliExitCode(err)
	if code != 1 {
		t.Errorf("cliExitCode(commandError) = %d, want 1", code)
	}
}

// TestUsageError_ExitCode verifies usageError returns exit code 2.
func TestUsageError_ExitCode(t *testing.T) {
	err := usageError("bad argument")
	if err == nil {
		t.Fatal("usageError returned nil")
	}
	code := cliExitCode(err)
	if code != 2 {
		t.Errorf("cliExitCode(usageError) = %d, want 2", code)
	}
}

// TestCommandError_Message verifies error message formatting.
func TestCommandError_Message(t *testing.T) {
	err := commandError("storage failed", nil)
	if err.Error() != "storage failed" {
		t.Errorf("commandError without cause: %q, want %q", err.Error(), "storage failed")
	}

	wrapped := commandError("storage failed", os.ErrNotExist)
	if !strings.Contains(wrapped.Error(), "storage failed") {
		t.Errorf("commandError with cause should contain message: %q", wrapped.Error())
	}
	if !strings.Contains(wrapped.Error(), "file does not exist") {
		t.Errorf("commandError with cause should contain cause: %q", wrapped.Error())
	}
}
