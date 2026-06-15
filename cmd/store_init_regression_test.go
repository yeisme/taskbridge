package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestStoreInitRegressionListJSONDoesNotPanic(t *testing.T) {
	storagePath := regularFileStoragePath(t)
	withStoreInitRegressionConfig(t, storagePath)
	listFormat = "json"
	listAll = true

	stdout, stderr, err := captureStoreInitFailure(t, func() error {
		return runList(testListCommand(), nil)
	})

	assertStoreInitFailure(t, err, stdout, stderr)
}

func TestStoreInitRegressionListsJSONDoesNotPanic(t *testing.T) {
	storagePath := regularFileStoragePath(t)
	withStoreInitRegressionConfig(t, storagePath)
	listsFormat = "json"

	stdout, stderr, err := captureStoreInitFailure(t, func() error {
		return runLists(&cobra.Command{}, nil)
	})

	assertStoreInitFailure(t, err, stdout, stderr)
}

func TestStoreInitRegressionTaskAddDoesNotPanic(t *testing.T) {
	storagePath := regularFileStoragePath(t)
	withStoreInitRegressionConfig(t, storagePath)

	stdout, stderr, err := captureStoreInitFailure(t, func() error {
		return runTaskAdd(&cobra.Command{}, []string{"regression task"})
	})

	assertStoreInitFailure(t, err, stdout, stderr)
}

func regularFileStoragePath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func withStoreInitRegressionConfig(t *testing.T, storagePath string) {
	t.Helper()

	previousCfg := cfg
	previousListSource := listSource
	previousListStatus := listStatus
	previousListFormat := listFormat
	previousListQuadrant := listQuadrant
	previousListPriority := listPriority
	previousListTag := listTag
	previousListNames := listNames
	previousListIDs := listIDs
	previousListTaskIDs := listTaskIDs
	previousListQuery := listQuery
	previousListAll := listAll
	previousListSyncNow := listSyncNow
	previousListLimit := listLimit
	previousListOffset := listOffset
	previousListFields := listFields
	previousListsSource := listsSource
	previousListsFormat := listsFormat
	previousListsSyncNow := listsSyncNow
	previousTaskListID := taskListID
	previousTaskDueDate := taskDueDate
	previousTaskPriority := taskPriority
	previousTaskQuadrant := taskQuadrant
	previousTaskFormat := taskFormat
	previousOutputJSON := outputJSON
	previousOutputAgent := outputAgent
	previousOutputEvents := outputEvents
	previousOutputExplain := outputExplain
	previousQuiet := quiet

	t.Cleanup(func() {
		cfg = previousCfg
		listSource = previousListSource
		listStatus = previousListStatus
		listFormat = previousListFormat
		listQuadrant = previousListQuadrant
		listPriority = previousListPriority
		listTag = previousListTag
		listNames = previousListNames
		listIDs = previousListIDs
		listTaskIDs = previousListTaskIDs
		listQuery = previousListQuery
		listAll = previousListAll
		listSyncNow = previousListSyncNow
		listLimit = previousListLimit
		listOffset = previousListOffset
		listFields = previousListFields
		listsSource = previousListsSource
		listsFormat = previousListsFormat
		listsSyncNow = previousListsSyncNow
		taskListID = previousTaskListID
		taskDueDate = previousTaskDueDate
		taskPriority = previousTaskPriority
		taskQuadrant = previousTaskQuadrant
		taskFormat = previousTaskFormat
		outputJSON = previousOutputJSON
		outputAgent = previousOutputAgent
		outputEvents = previousOutputEvents
		outputExplain = previousOutputExplain
		quiet = previousQuiet
	})

	setupTestConfig(t, storagePath)
	cfg.Storage.File.Format = "json"
	listSource = ""
	listStatus = ""
	listFormat = "table"
	listQuadrant = 0
	listPriority = 0
	listTag = ""
	listNames = nil
	listIDs = nil
	listTaskIDs = nil
	listQuery = ""
	listAll = false
	listSyncNow = false
	listLimit = 0
	listOffset = 0
	listFields = ""
	listsSource = ""
	listsFormat = "table"
	listsSyncNow = false
	taskListID = ""
	taskDueDate = ""
	taskPriority = 0
	taskQuadrant = 0
	taskFormat = "text"
	outputJSON = false
	outputAgent = false
	outputEvents = false
	outputExplain = false
	quiet = false
}

func captureStoreInitFailure(t *testing.T, run func() error) (string, string, error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	runErr := run()
	if runErr != nil {
		fmt.Fprintln(os.Stderr, formatCLIError(runErr))
	}

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}

	stdoutBytes, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrBytes, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	return string(stdoutBytes), string(stderrBytes), runErr
}

func assertStoreInitFailure(t *testing.T, err error, stdout, stderr string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected storage initialization error")
	}
	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("error should be *CLIError, got %T", err)
	}
	if cliErr.ExitCode == 0 {
		t.Fatal("exit code should be non-zero on storage initialization failure")
	}
	if !strings.Contains(stderr, "Failed to create storage") {
		t.Fatalf("stderr should include command storage initialization message, got %q", stderr)
	}
	if !strings.Contains(stderr, "failed to create storage directory") {
		t.Fatalf("stderr should include low-level storage initialization message, got %q", stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout on storage initialization failure, got %q", stdout)
	}
	assertNoPanicStackTrace(t, stdout)
}

func assertNoPanicStackTrace(t *testing.T, output string) {
	t.Helper()

	if strings.Contains(output, "panic:") || strings.Contains(output, "goroutine ") {
		t.Fatalf("stdout should not contain a panic stack trace, got %q", output)
	}
}
