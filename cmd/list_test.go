package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		if err := store.SaveTask(nil, task); err != nil {
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
