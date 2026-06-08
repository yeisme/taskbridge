package filestore

import (
	"context"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage"
)

func TestStorageReadResultsDoNotExposeMutableTaskState(t *testing.T) {
	for name, newStore := range map[string]func(t *testing.T) ownershipStore{
		"file": func(t *testing.T) ownershipStore {
			store, err := New(t.TempDir(), "json")
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			return store
		},
		"provider": func(t *testing.T) ownershipStore {
			store, err := NewProviderStorage("local", t.TempDir())
			if err != nil {
				t.Fatalf("NewProviderStorage: %v", err)
			}
			return store
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := newStore(t)
			due := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			completed := due.Add(time.Hour)
			parentID := "parent"
			original := &model.Task{
				ID:          "task-1",
				Title:       "original",
				Status:      model.StatusTodo,
				CreatedAt:   due,
				UpdatedAt:   due,
				CompletedAt: &completed,
				DueDate:     &due,
				ParentID:    &parentID,
				Tags:        []string{"alpha"},
				SubtaskIDs:  []string{"child-1"},
				Metadata: &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{
					"k": "v",
				}},
				Source: model.SourceLocal,
			}
			if err := store.SaveTask(ctx, original); err != nil {
				t.Fatalf("SaveTask: %v", err)
			}

			original.Title = "mutated-after-save"
			original.Tags[0] = "changed-after-save"
			original.Metadata.CustomFields["k"] = "changed-after-save"

			loaded, err := store.GetTask(ctx, "task-1")
			if err != nil {
				t.Fatalf("GetTask: %v", err)
			}
			loaded.Title = "mutated-get"
			loaded.Tags[0] = "mutated-tag"
			loaded.SubtaskIDs[0] = "mutated-child"
			loaded.Metadata.CustomFields["k"] = "mutated-map"
			loaded.DueDate = ptrTime(loaded.DueDate.AddDate(1, 0, 0))

			listed, err := store.ListTasks(ctx, storage.ListOptions{})
			if err != nil {
				t.Fatalf("ListTasks: %v", err)
			}
			listed[0].Tags[0] = "mutated-list"
			listed[0].Metadata.CustomFields["k"] = "mutated-list-map"

			queried, err := store.QueryTasks(ctx, storage.Query{})
			if err != nil {
				t.Fatalf("QueryTasks: %v", err)
			}
			queried[0].Tags[0] = "mutated-query"
			queried[0].Metadata.CustomFields["k"] = "mutated-query-map"

			again, err := store.GetTask(ctx, "task-1")
			if err != nil {
				t.Fatalf("GetTask again: %v", err)
			}
			if again.Title != "original" || again.Tags[0] != "alpha" || again.SubtaskIDs[0] != "child-1" || again.Metadata.CustomFields["k"] != "v" {
				t.Fatalf("store state was mutated through read result: %+v", again)
			}
			if again.DueDate == nil || !again.DueDate.Equal(due) {
				t.Fatalf("due date mutated through read result: %+v", again.DueDate)
			}
		})
	}
}

func TestStorageReadResultsDoNotExposeMutableTaskListState(t *testing.T) {
	for name, newStore := range map[string]func(t *testing.T) ownershipStore{
		"file": func(t *testing.T) ownershipStore {
			store, err := New(t.TempDir(), "json")
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			return store
		},
		"provider": func(t *testing.T) ownershipStore {
			store, err := NewProviderStorage("local", t.TempDir())
			if err != nil {
				t.Fatalf("NewProviderStorage: %v", err)
			}
			return store
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := newStore(t)
			list := &model.TaskList{ID: "list-1", Name: "original", Source: model.SourceLocal}
			if err := store.SaveTaskList(ctx, list); err != nil {
				t.Fatalf("SaveTaskList: %v", err)
			}
			list.Name = "mutated-after-save"

			loaded, err := store.GetTaskList(ctx, "list-1")
			if err != nil {
				t.Fatalf("GetTaskList: %v", err)
			}
			loaded.Name = "mutated-get"

			listed, err := store.ListTaskLists(ctx)
			if err != nil {
				t.Fatalf("ListTaskLists: %v", err)
			}
			listed[0].Name = "mutated-list"

			again, err := store.GetTaskList(ctx, "list-1")
			if err != nil {
				t.Fatalf("GetTaskList again: %v", err)
			}
			if again.Name != "original" {
				t.Fatalf("store list state was mutated through read result: %+v", again)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

type ownershipStore interface {
	SaveTask(context.Context, *model.Task) error
	GetTask(context.Context, string) (*model.Task, error)
	ListTasks(context.Context, storage.ListOptions) ([]model.Task, error)
	QueryTasks(context.Context, storage.Query) ([]model.Task, error)
	SaveTaskList(context.Context, *model.TaskList) error
	GetTaskList(context.Context, string) (*model.TaskList, error)
	ListTaskLists(context.Context) ([]model.TaskList, error)
}
