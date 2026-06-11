package projectservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/project"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

func TestSyncProjectDoesNotMarkSyncedWhenProviderPushFails(t *testing.T) {
	ctx := context.Background()
	taskStore, err := filestore.New(t.TempDir(), "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	projectStore, err := project.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("project.NewFileStore: %v", err)
	}

	projectID := "proj-sync-fail"
	if err := projectStore.SaveProject(ctx, &project.Project{ID: projectID, Name: "项目", GoalText: "项目", GoalType: project.GoalTypeGeneric, Status: project.StatusConfirmed}); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}
	if err := taskStore.SaveTask(ctx, &model.Task{ID: "task-1", Title: "同步任务", Status: model.StatusTodo, Source: model.SourceLocal, CreatedAt: time.Now(), UpdatedAt: time.Now(), Metadata: &model.TaskMetadata{Version: "1.0", CustomFields: map[string]interface{}{"tb_project_id": projectID}}}); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}

	service := SyncService{TaskStore: taskStore, ProjectStore: projectStore}
	result, err := service.SyncProject(ctx, projectID, failingCreateProvider{}, "todoist")
	if err != nil {
		t.Fatalf("SyncProject: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected provider create error in result")
	}
	if result.Status == string(project.StatusSynced) {
		t.Fatalf("project sync status=%q with errors, want non-synced", result.Status)
	}

	stored, err := projectStore.GetProject(ctx, projectID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if stored.Status == project.StatusSynced {
		t.Fatalf("stored project status=%q with push errors, want non-synced", stored.Status)
	}
}

type failingCreateProvider struct{}

func (failingCreateProvider) Name() string                                               { return "todoist" }
func (failingCreateProvider) DisplayName() string                                        { return "Todoist" }
func (failingCreateProvider) Authenticate(context.Context, map[string]interface{}) error { return nil }
func (failingCreateProvider) IsAuthenticated() bool                                      { return true }
func (failingCreateProvider) RefreshToken(context.Context) error                         { return nil }
func (failingCreateProvider) ListTaskLists(context.Context) ([]model.TaskList, error) {
	return []model.TaskList{{ID: "list-1", Name: "Inbox", Source: model.SourceTodoist}}, nil
}
func (failingCreateProvider) CreateTaskList(context.Context, string) (*model.TaskList, error) {
	return nil, errors.New("not implemented")
}
func (failingCreateProvider) DeleteTaskList(context.Context, string) error { return nil }
func (failingCreateProvider) ListTasks(context.Context, string, provider.ListOptions) ([]model.Task, error) {
	return nil, nil
}
func (failingCreateProvider) GetTask(context.Context, string, string) (*model.Task, error) {
	return nil, nil
}
func (failingCreateProvider) SearchTasks(context.Context, string) ([]model.Task, error) {
	return nil, nil
}
func (failingCreateProvider) CreateTask(context.Context, string, *model.Task) (*model.Task, error) {
	return nil, errors.New("remote create failed")
}
func (failingCreateProvider) UpdateTask(context.Context, string, *model.Task) (*model.Task, error) {
	return nil, errors.New("remote update failed")
}
func (failingCreateProvider) DeleteTask(context.Context, string, string) error {
	return errors.New("remote delete failed")
}
func (failingCreateProvider) BatchCreate(context.Context, string, []*model.Task) ([]model.Task, error) {
	return nil, errors.New("remote batch create failed")
}
func (failingCreateProvider) BatchUpdate(context.Context, string, []*model.Task) ([]model.Task, error) {
	return nil, errors.New("remote batch update failed")
}
func (failingCreateProvider) GetChanges(context.Context, time.Time) (*provider.SyncChanges, error) {
	return nil, nil
}
func (failingCreateProvider) Capabilities() provider.Capabilities { return provider.Capabilities{} }
func (failingCreateProvider) GetTokenInfo() *provider.TokenInfo {
	return &provider.TokenInfo{Provider: "todoist", HasToken: true, IsValid: true}
}
