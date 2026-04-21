package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRootHelpIsCLIFirstAndNoProtocolService(t *testing.T) {
	output := helpOutput(t, rootCmd)
	legacyBranding := "TaskBridge" + " " + "M" + "C" + "P"
	if !strings.Contains(output, "以 CLI 为主的任务工作流工具") {
		t.Fatalf("expected CLI-first wording, got: %s", output)
	}
	if strings.Contains(output, "\n"+"m"+"cp ") || strings.Contains(output, " "+"m"+"cp ") {
		t.Fatalf("expected protocol command to be removed, got: %s", output)
	}
	if !strings.Contains(output, "project") || !strings.Contains(output, "governance") {
		t.Fatalf("expected new CLI-only commands, got: %s", output)
	}
	if strings.Contains(output, legacyBranding) {
		t.Fatalf("expected legacy protocol-first branding to be removed, got: %s", output)
	}
}

func TestVersionOutputUsesTaskBridgeBranding(t *testing.T) {
	previous := versionJSON
	versionJSON = false
	defer func() { versionJSON = previous }()

	output := captureStdout(t, func() {
		runVersion(nil, nil)
	})
	legacyBranding := "TaskBridge" + " " + "M" + "C" + "P"

	if !strings.Contains(output, "TaskBridge - 面向 AI 与多 Todo 平台的 CLI 工作流工具") {
		t.Fatalf("expected CLI-first version banner, got: %s", output)
	}
	if strings.Contains(output, legacyBranding) {
		t.Fatalf("expected legacy protocol-first branding to be removed, got: %s", output)
	}
}

func TestServeHelpNoLongerMentionsProtocolService(t *testing.T) {
	output := helpOutput(t, serveCmd)
	if strings.Contains(output, "M"+"C"+"P") {
		t.Fatalf("expected serve help to be CLI-only, got: %s", output)
	}
}

func helpOutput(t *testing.T, cmd interface {
	Help() error
	SetOut(io.Writer)
	SetErr(io.Writer)
}) string {
	t.Helper()

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Help(); err != nil {
		t.Fatalf("help: %v", err)
	}
	return buf.String()
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(out)
}
