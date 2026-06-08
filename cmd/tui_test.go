package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/storage"
)

func TestTUIToggleCompleteReportsSaveFailure(t *testing.T) {
	task := model.Task{ID: "task-1", Title: "Task", Status: model.StatusTodo, Source: model.SourceLocal}
	store := &failingTUIStore{tasks: []model.Task{task}, saveErr: errors.New("disk full")}
	m := Model{currentView: ViewTasks, tasks: store.tasks, filtered: store.tasks, selected: 0, store: store}

	next, _ := m.toggleComplete()
	got := next.(Model)

	if got.err == nil {
		t.Fatalf("expected visible save error")
	}
	if got.filtered[0].Status != model.StatusTodo {
		t.Fatalf("expected filtered task status to remain todo, got %s", got.filtered[0].Status)
	}
}

func TestTUIDeleteReportsDeleteFailure(t *testing.T) {
	task := model.Task{ID: "task-1", Title: "Task", Status: model.StatusTodo, Source: model.SourceLocal}
	store := &failingTUIStore{tasks: []model.Task{task}, deleteErr: errors.New("disk full")}
	m := Model{inputMode: ModeConfirmDelete, expandedTask: &task, filtered: store.tasks, store: store}

	next, _ := m.handleConfirmDeleteInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got := next.(Model)

	if got.err == nil {
		t.Fatalf("expected visible delete error")
	}
	if len(got.filtered) != 1 {
		t.Fatalf("expected task to remain visible when delete fails")
	}
}

type failingTUIStore struct {
	tasks     []model.Task
	saveErr   error
	deleteErr error
	listErr   error
}

func (s *failingTUIStore) SaveTask(_ context.Context, task *model.Task) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	for i := range s.tasks {
		if s.tasks[i].ID == task.ID {
			s.tasks[i] = *task
			return nil
		}
	}
	s.tasks = append(s.tasks, *task)
	return nil
}

func (s *failingTUIStore) GetTask(_ context.Context, id string) (*model.Task, error) {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			return &s.tasks[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (s *failingTUIStore) ListTasks(context.Context, storage.ListOptions) ([]model.Task, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]model.Task(nil), s.tasks...), nil
}

func (s *failingTUIStore) DeleteTask(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *failingTUIStore) SaveTasks(ctx context.Context, tasks []*model.Task) error {
	for _, task := range tasks {
		if err := s.SaveTask(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func (s *failingTUIStore) QueryTasks(ctx context.Context, _ storage.Query) ([]model.Task, error) {
	return s.ListTasks(ctx, storage.ListOptions{})
}

func (s *failingTUIStore) SaveTaskList(context.Context, *model.TaskList) error { return nil }
func (s *failingTUIStore) GetTaskList(context.Context, string) (*model.TaskList, error) {
	return nil, errors.New("not found")
}
func (s *failingTUIStore) ListTaskLists(context.Context) ([]model.TaskList, error) {
	return nil, nil
}
func (s *failingTUIStore) DeleteTaskList(context.Context, string) error { return nil }
func (s *failingTUIStore) ExportToJSON(context.Context, storage.ExportOptions) ([]byte, error) {
	return []byte("[]"), nil
}
func (s *failingTUIStore) ExportToMarkdown(context.Context, storage.ExportOptions) ([]byte, error) {
	return []byte(""), nil
}
func (s *failingTUIStore) GetLastSyncTime(context.Context, model.TaskSource) (*time.Time, error) {
	return nil, nil
}
func (s *failingTUIStore) SetLastSyncTime(context.Context, model.TaskSource, time.Time) error {
	return nil
}

var _ storage.Storage = (*failingTUIStore)(nil)
