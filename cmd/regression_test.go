package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/yeisme/taskbridge/pkg/config"
)

// TestAnalyzeStorageInitFailureDoesNotPanic verifies analyze commands don't panic on bad storage.
func TestAnalyzeStorageInitFailureDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	// Create a file (not a directory) at storage path to force init failure
	filePath := dir + "/bad-storage"
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg = &config.Config{
		Storage: config.StorageConfig{
			Path: filePath,
			File: config.FileStorageConfig{
				Format: "json",
			},
		},
	}

	// Test quadrant subcommand
	analyzeFormat = "json"
	cmd := testCmd()
	cmd.SetArgs([]string{"analyze", "quadrant"})
	// Should not panic; should return error
	_ = cmd.Execute()
}

// TestListsStorageInitFailureDoesNotPanic verifies lists command doesn't panic on bad storage.
func TestListsStorageInitFailureDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/bad-storage"
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg = &config.Config{
		Storage: config.StorageConfig{
			Path: filePath,
			File: config.FileStorageConfig{
				Format: "json",
			},
		},
	}

	cmd := testCmd()
	cmd.SetArgs([]string{"lists"})
	_ = cmd.Execute()
}

// TestTaskAddStorageInitFailureDoesNotPanic verifies task add doesn't panic on bad storage.
func TestTaskAddStorageInitFailureDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/bad-storage"
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg = &config.Config{
		Storage: config.StorageConfig{
			Path: filePath,
			File: config.FileStorageConfig{
				Format: "json",
			},
		},
	}

	cmd := testCmd()
	cmd.SetArgs([]string{"task", "add", "test task"})
	_ = cmd.Execute()
}

// TestVersionOutput verifies version command produces output.
func TestVersionOutput(t *testing.T) {
	cfg = &config.Config{}

	stdout := captureStdout(t, func() {
		cmd := testCmd()
		cmd.SetArgs([]string{"version"})
		_ = cmd.Execute()
	})

	if stdout == "" {
		t.Error("version command produced no output")
	}
}

// TestRootVersionFlag verifies release smoke can use the conventional --version flag.
func TestRootVersionFlag(t *testing.T) {
	cfg = &config.Config{}

	var stdout bytes.Buffer
	cmd := testCmd()
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute --version: %v", err)
	}

	if stdout.String() == "" {
		t.Fatal("--version produced no output")
	}
}

// TestVersionJSONOutput verifies version --json produces valid JSON.
func TestVersionJSONOutput(t *testing.T) {
	cfg = &config.Config{}

	stdout := captureStdout(t, func() {
		cmd := testCmd()
		cmd.SetArgs([]string{"version", "--json"})
		_ = cmd.Execute()
	})

	if stdout == "" {
		t.Error("version --json produced no output")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("version JSON not parseable: %v\noutput: %q", err, stdout)
	}
	if payload["mode"] != "json" || payload["command"] != "version.show" {
		t.Errorf("expected version JSON envelope: %+v", payload)
	}
	data, ok := payload["data"].(map[string]interface{})
	if !ok || data["version"] == nil {
		t.Errorf("expected version in data payload: %+v", payload)
	}
}

func testCmd() *cobra.Command {
	rootCmd.SetOut(new(bytes.Buffer))
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(nil)
	return rootCmd
}
