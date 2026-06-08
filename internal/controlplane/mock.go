package controlplane

import (
	"context"
	"sync"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage"
)

// mockStore implements storage.Storage backed by an in-memory slice of tasks.
// Only the methods needed by controlplane.Service are fully implemented.
type mockStore struct {
	mu    sync.RWMutex
	tasks []*model.Task
}

func newMockStore(tasks []model.Task) *mockStore {
	ptrs := make([]*model.Task, len(tasks))
	for i := range tasks {
		ptrs[i] = &tasks[i]
	}
	return &mockStore{tasks: ptrs}
}

func (m *mockStore) SaveTask(_ context.Context, task *model.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.tasks {
		if t.ID == task.ID {
			m.tasks[i] = task
			return nil
		}
	}
	m.tasks = append(m.tasks, task)
	return nil
}

func (m *mockStore) GetTask(_ context.Context, id string) (*model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockStore) ListTasks(_ context.Context, _ storage.ListOptions) ([]model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]model.Task, len(m.tasks))
	for i, t := range m.tasks {
		result[i] = *t
	}
	return result, nil
}

func (m *mockStore) DeleteTask(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.tasks {
		if t.ID == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockStore) SaveTasks(_ context.Context, tasks []*model.Task) error {
	for _, t := range tasks {
		_ = m.SaveTask(context.Background(), t)
	}
	return nil
}

func (m *mockStore) QueryTasks(_ context.Context, query storage.Query) ([]model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []model.Task
	for _, t := range m.tasks {
		if !matchSource(t, query.Sources) {
			continue
		}
		if !matchStatus(t, query.Statuses) {
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}

func matchSource(t *model.Task, sources []model.TaskSource) bool {
	if len(sources) == 0 {
		return true
	}
	for _, s := range sources {
		if t.Source == s {
			return true
		}
	}
	return false
}

func matchStatus(t *model.Task, statuses []model.TaskStatus) bool {
	if len(statuses) == 0 {
		return true
	}
	for _, s := range statuses {
		if t.Status == s {
			return true
		}
	}
	return false
}

// Unimplemented methods — controlplane.Service does not call these.

func (m *mockStore) SaveTaskList(_ context.Context, _ *model.TaskList) error {
	return nil
}
func (m *mockStore) GetTaskList(_ context.Context, _ string) (*model.TaskList, error) {
	return nil, nil
}
func (m *mockStore) ListTaskLists(_ context.Context) ([]model.TaskList, error) {
	return nil, nil
}
func (m *mockStore) DeleteTaskList(_ context.Context, _ string) error {
	return nil
}
func (m *mockStore) ExportToJSON(_ context.Context, _ storage.ExportOptions) ([]byte, error) {
	return nil, nil
}
func (m *mockStore) ExportToMarkdown(_ context.Context, _ storage.ExportOptions) ([]byte, error) {
	return nil, nil
}
func (m *mockStore) GetLastSyncTime(_ context.Context, _ model.TaskSource) (*time.Time, error) {
	return nil, nil
}
func (m *mockStore) SetLastSyncTime(_ context.Context, _ model.TaskSource, _ time.Time) error {
	return nil
}

// NewMockService returns a Service pre-loaded with MockTasks.
// Used by --mock flag on today/next/inbox/review commands.
func NewMockService(now time.Time) *Service {
	tasks := MockTasks(now)
	store := newMockStore(tasks)
	return &Service{TaskStore: store}
}
