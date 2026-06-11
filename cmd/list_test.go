package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
	"github.com/yeisme/taskbridge/pkg/config"
)

func TestListJSONEmptyResultIsParseableObject(t *testing.T) {
	withListTestConfig(t, t.TempDir())

	stdout := captureStdout(t, func() {
		if err := runList(testListCommand(), nil); err != nil {
			t.Fatalf("runList: %v", err)
		}
	})

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected JSON object, got err=%v stdout=%q", err, stdout)
	}
	if _, ok := payload["tasks"].([]interface{}); !ok {
		t.Fatalf("expected tasks array in payload: %+v", payload)
	}
	if payload["total"].(float64) != 0 || payload["has_more"].(bool) {
		t.Fatalf("unexpected pagination metadata: %+v", payload)
	}
}

func TestListJSONPaginationDoesNotAppendHumanHint(t *testing.T) {
	dir := t.TempDir()
	withListTestConfig(t, dir)
	store, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Now().UTC()
	for _, task := range []*model.Task{
		{ID: "task-1", Title: "One", Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: now, UpdatedAt: now},
		{ID: "task-2", Title: "Two", Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: now, UpdatedAt: now.Add(time.Second)},
	} {
		if err := store.SaveTask(context.Background(), task); err != nil {
			t.Fatalf("SaveTask(%s): %v", task.ID, err)
		}
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	listLimit = 1

	stdout := captureStdout(t, func() {
		if err := runList(testListCommand(), nil); err != nil {
			t.Fatalf("runList: %v", err)
		}
	})

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected pure JSON object, got err=%v stdout=%q", err, stdout)
	}
	if payload["total"].(float64) != 2 || payload["limit"].(float64) != 1 || !payload["has_more"].(bool) {
		t.Fatalf("unexpected pagination payload: %+v", payload)
	}
}

func TestListDefaultUsesTableNotJSON(t *testing.T) {
	dir := t.TempDir()
	withListTestConfig(t, dir)
	listFormat = "table"
	store, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Now().UTC()
	if err := store.SaveTask(context.Background(), &model.Task{ID: "task-1", Title: "One", Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := runList(testListCommand(), nil); err != nil {
			t.Fatalf("runList: %v", err)
		}
	})

	if strings.HasPrefix(strings.TrimSpace(stdout), "{") || strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		t.Fatalf("list default should not emit raw JSON:\n%s", stdout)
	}
	if !strings.Contains(stdout, "One") {
		t.Fatalf("list default should include a human task table, got:\n%s", stdout)
	}
}

func TestListEventsRoutesThroughProjectionDispatcher(t *testing.T) {
	withListTestConfig(t, t.TempDir())
	oldEvents := outputEvents
	outputEvents = true
	t.Cleanup(func() { outputEvents = oldEvents })

	err := runList(testListCommand(), nil)
	if err == nil || !strings.Contains(err.Error(), "--events") {
		t.Fatalf("list --events should be rejected by projection dispatcher, got %v", err)
	}
}

func TestListExplainUsesProjectionRenderer(t *testing.T) {
	withListTestConfig(t, t.TempDir())
	oldExplain := outputExplain
	outputExplain = true
	t.Cleanup(func() { outputExplain = oldExplain })

	stdout := captureStdout(t, func() {
		if err := runList(testListCommand(), nil); err != nil {
			t.Fatalf("runList: %v", err)
		}
	})
	if !strings.Contains(stdout, "Conclusion") || strings.Contains(stdout, "Total 0 tasks") {
		t.Fatalf("list --explain should render explain sections, got:\n%s", stdout)
	}
}

func TestListsEventsRoutesThroughProjectionDispatcher(t *testing.T) {
	withListTestConfig(t, t.TempDir())
	oldEvents := outputEvents
	outputEvents = true
	t.Cleanup(func() { outputEvents = oldEvents })

	err := runLists(&cobra.Command{}, nil)
	if err == nil || !strings.Contains(err.Error(), "--events") {
		t.Fatalf("lists --events should be rejected by projection dispatcher, got %v", err)
	}
}

func TestListsExplainUsesProjectionRenderer(t *testing.T) {
	withListTestConfig(t, t.TempDir())
	oldExplain := outputExplain
	outputExplain = true
	t.Cleanup(func() { outputExplain = oldExplain })

	stdout := captureStdout(t, func() {
		if err := runLists(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runLists: %v", err)
		}
	})
	if !strings.Contains(stdout, "Conclusion") || strings.Contains(stdout, "Provider") {
		t.Fatalf("lists --explain should render explain sections, got:\n%s", stdout)
	}
}

func TestListsDefaultUsesTableNotJSON(t *testing.T) {
	dir := t.TempDir()
	withListTestConfig(t, dir)
	listsFormat = "table"
	store, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Now().UTC()
	if err := store.SaveTaskList(context.Background(), &model.TaskList{ID: "list-1", Name: "Inbox", Source: model.SourceLocal, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("SaveTaskList: %v", err)
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := runLists(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runLists: %v", err)
		}
	})

	if strings.HasPrefix(strings.TrimSpace(stdout), "{") || strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		t.Fatalf("lists default should not emit raw JSON:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Inbox") || !strings.Contains(stdout, "Provider") {
		t.Fatalf("lists default should include a human table, got:\n%s", stdout)
	}
}

func TestListsJSONUsesParseableLegacyPayload(t *testing.T) {
	dir := t.TempDir()
	withListTestConfig(t, dir)
	listsFormat = "json"
	store, err := filestore.New(dir, "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	now := time.Now().UTC()
	if err := store.SaveTaskList(context.Background(), &model.TaskList{ID: "list-1", Name: "Inbox", Source: model.SourceLocal, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("SaveTaskList: %v", err)
	}
	if err := store.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := runLists(&cobra.Command{}, nil); err != nil {
			t.Fatalf("runLists: %v", err)
		}
	})

	var payload []map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("lists --format json should remain parseable, got %v in %q", err, stdout)
	}
	if len(payload) != 1 || payload[0]["list_name"] != "Inbox" {
		t.Fatalf("unexpected lists json payload: %+v", payload)
	}
}

func TestListStorageInitFailureDoesNotPanic(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	withListTestConfig(t, filePath)

	stdout := captureStdout(t, func() {
		err := runList(testListCommand(), nil)
		if err == nil {
			t.Fatalf("expected storage init error")
		}
	})
	if stdout != "" {
		t.Fatalf("expected empty stdout on storage init failure, got %q", stdout)
	}
}

func withListTestConfig(t *testing.T, storagePath string) {
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
	})

	cfg = config.DefaultConfig()
	cfg.Storage.Path = storagePath
	cfg.Storage.File.Format = "json"
	listSource = ""
	listStatus = ""
	listFormat = "json"
	listQuadrant = 0
	listPriority = 0
	listTag = ""
	listNames = nil
	listIDs = nil
	listTaskIDs = nil
	listQuery = ""
	listAll = true
	listSyncNow = false
	listLimit = 0
	listOffset = 0
	listFields = ""
	listsSource = ""
	listsFormat = "table"
	listsSyncNow = false
}

func testListCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("status", "", "")
	cmd.Flags().Int("limit", 0, "")
	if listLimit != 0 {
		_ = cmd.Flags().Set("limit", "1")
	}
	return cmd
}

var _ storage.Storage
