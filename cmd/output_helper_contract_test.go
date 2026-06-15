package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

type outputHelperPayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestOutputHelperWantsJSONContract(t *testing.T) {
	oldJSON, oldQuiet := outputJSON, quiet
	outputJSON, quiet = false, false
	t.Cleanup(func() { outputJSON, quiet = oldJSON, oldQuiet })

	if !wantsJSON("json") {
		t.Fatal(`wantsJSON("json") = false, want true`)
	}
	for _, format := range []string{"text", "table"} {
		if wantsJSON(format) {
			t.Fatalf("wantsJSON(%q) = true, want false", format)
		}
	}
}

func TestOutputHelperPrintStructuredJSONModesWriteJSONToStdoutOnly(t *testing.T) {
	for _, tc := range []struct {
		name       string
		format     string
		globalJSON bool
	}{
		{name: "format json", format: "json"},
		{name: "global json", format: "table", globalJSON: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldJSON, oldQuiet := outputJSON, quiet
			outputJSON, quiet = tc.globalJSON, false
			t.Cleanup(func() { outputJSON, quiet = oldJSON, oldQuiet })

			var err error
			var stdout string
			stderr := captureStderr(t, func() {
				stdout = captureStdout(t, func() {
					err = printStructured(tc.format, outputHelperPayload{Name: "alpha", Count: 2}, func() {
						fmt.Fprintln(os.Stdout, "Human summary should not be used in JSON mode")
					})
				})
			})

			if err != nil {
				t.Fatalf("printStructured returned error: %v", err)
			}
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("JSON mode wrote diagnostics to stderr, got %q", stderr)
			}
			var payload outputHelperPayload
			if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
				t.Fatalf("JSON mode stdout is not valid JSON: %v\nstdout: %q", err, stdout)
			}
			if payload.Name != "alpha" || payload.Count != 2 {
				t.Fatalf("unexpected JSON payload: %+v", payload)
			}
		})
	}
}

func TestOutputHelperPrintStructuredHumanModeWritesTextNotJSON(t *testing.T) {
	oldJSON, oldQuiet := outputJSON, quiet
	outputJSON, quiet = false, false
	t.Cleanup(func() { outputJSON, quiet = oldJSON, oldQuiet })

	var err error
	stdout := captureStdout(t, func() {
		err = printStructured("table", outputHelperPayload{Name: "alpha", Count: 2}, func() {
			fmt.Fprintln(os.Stdout, "Human summary")
		})
	})
	if err != nil {
		t.Fatalf("printStructured returned error: %v", err)
	}
	if strings.TrimSpace(stdout) != "Human summary" {
		t.Fatalf("human mode stdout = %q, want human summary", stdout)
	}
	if json.Valid([]byte(strings.TrimSpace(stdout))) {
		t.Fatalf("human mode stdout should not be JSON: %q", stdout)
	}
}

func TestOutputHelperMarshalErrorsReturnCommandError(t *testing.T) {
	oldJSON, oldQuiet := outputJSON, quiet
	outputJSON, quiet = false, false
	t.Cleanup(func() { outputJSON, quiet = oldJSON, oldQuiet })

	bad := map[string]any{"bad": func() {}}
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{name: "printStructured", call: func() error { return printStructured("json", bad, nil) }},
		{name: "printResult", call: func() error { return printResult(bad) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			stdout := captureStdout(t, func() { err = tc.call() })
			if stdout != "" {
				t.Fatalf("marshal failure should leave stdout empty, got %q", stdout)
			}
			if err == nil {
				t.Fatal("expected non-nil command error")
			}
			if cliExitCode(err) != 1 {
				t.Fatalf("cliExitCode = %d, want 1 for command error", cliExitCode(err))
			}
			if !strings.Contains(err.Error(), "Serialized output failed") {
				t.Fatalf("error should describe serialization failure, got %v", err)
			}
		})
	}
}

func TestOutputHelperQuietModeStreamlinesPrintResult(t *testing.T) {
	oldQuiet := quiet
	quiet = true
	t.Cleanup(func() { quiet = oldQuiet })

	var err error
	stdout := captureStdout(t, func() {
		err = printResult(outputHelperPayload{Name: "alpha", Count: 2})
	})
	if err != nil {
		t.Fatalf("printResult returned error: %v", err)
	}
	if strings.Count(stdout, "\n") != 1 || strings.Contains(stdout, "\n  ") {
		t.Fatalf("quiet mode should emit compact one-line JSON, got %q", stdout)
	}
	var payload outputHelperPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		t.Fatalf("quiet mode stdout is not valid JSON: %v\nstdout: %q", err, stdout)
	}
	if payload.Name != "alpha" || payload.Count != 2 {
		t.Fatalf("unexpected quiet payload: %+v", payload)
	}
}

func TestOutputHelperPrintProjectionWithLegacyJSONGlobalFlagUsesEnvelope(t *testing.T) {
	oldJSON, oldQuiet := outputJSON, quiet
	outputJSON, quiet = true, false
	t.Cleanup(func() { outputJSON, quiet = oldJSON, oldQuiet })

	projection := clioutput.New("project.list")
	projection.Summary = "Loaded projects."
	legacy := map[string]any{"legacy_only": true}

	var err error
	var stdout string
	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			err = printProjectionWithLegacyJSON("table", legacy, projection, func() {
				fmt.Fprintln(os.Stdout, "Human project list")
			})
		})
	})
	if err != nil {
		t.Fatalf("printProjectionWithLegacyJSON returned error: %v", err)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("global JSON mode wrote diagnostics to stderr, got %q", stderr)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		t.Fatalf("global JSON mode stdout is not valid JSON: %v\nstdout: %q", err, stdout)
	}
	if payload["mode"] != "json" || payload["command"] != "project.list" || payload["status"] != "success" {
		t.Fatalf("unexpected projection envelope: %#v", payload)
	}
	if _, ok := payload["legacy_only"]; ok {
		t.Fatalf("global --json should render projection envelope, not legacy payload: %#v", payload)
	}
}

func TestOutputHelperPipeDetectionStreamlinesListLimit(t *testing.T) {
	dir := t.TempDir()
	withListTestConfig(t, dir)
	listFormat = "table"

	store, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Now().UTC()
	for i := 0; i < 55; i++ {
		task := &model.Task{
			ID:        fmt.Sprintf("task-%03d", i),
			Title:     fmt.Sprintf("Pipe task %03d", i),
			Status:    model.StatusTodo,
			Source:    model.SourceLocal,
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			UpdatedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := store.SaveTask(context.Background(), task); err != nil {
			t.Fatalf("SaveTask(%s): %v", task.ID, err)
		}
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := runList(testListCommand(), nil); err != nil {
			t.Fatalf("runList: %v", err)
		}
	})

	if !strings.Contains(stdout, "Total 55 tasks (display first 50") {
		t.Fatalf("pipe-detected stdout should be limited with pagination hint, got:\n%s", stdout)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return string(out)
}
