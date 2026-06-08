// Package filestore 提供基于文件的存储实现
package filestore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/persistence"
	"github.com/yeisme/taskbridge/internal/storage"
)

// FileStorage 文件存储实现
type FileStorage struct {
	// mu 读写锁
	mu sync.RWMutex
	// basePath 存储基础路径
	basePath string
	// format 存储格式
	format string
	// tasksFile 任务文件路径
	tasksFile string
	// listsFile 列表文件路径
	listsFile string
	// syncFile 同步状态文件路径
	syncFile string
	// dirty 是否有未写盘的变更
	dirty bool

	// tasks 任务映射表
	tasks map[string]*model.Task
	// taskLists 任务列表映射表
	taskLists map[string]*model.TaskList
	// syncTimes 同步时间记录
	syncTimes map[model.TaskSource]time.Time
}

type tasksPersistData struct {
	Tasks []*model.Task `json:"tasks"`
}

type listsPersistData struct {
	Lists []*model.TaskList `json:"lists"`
}

type syncPersistData struct {
	SyncTimes map[string]string `json:"sync_times"`
}

const (
	tasksSchema = "taskbridge.storage.tasks"
	listsSchema = "taskbridge.storage.lists"
	syncSchema  = "taskbridge.storage.sync"
)

// New 创建文件存储实例
func New(basePath, format string) (*FileStorage, error) {
	fs := &FileStorage{
		basePath:  basePath,
		format:    format,
		tasksFile: filepath.Join(basePath, "tasks.json"),
		listsFile: filepath.Join(basePath, "lists.json"),
		syncFile:  filepath.Join(basePath, "sync.json"),
		tasks:     make(map[string]*model.Task),
		taskLists: make(map[string]*model.TaskList),
		syncTimes: make(map[model.TaskSource]time.Time),
	}

	// 确保目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 加载现有数据
	if err := fs.load(); err != nil {
		// 如果文件不存在，忽略错误
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load storage: %w", err)
		}
	}

	return fs, nil
}

// load 从文件加载数据
func (fs *FileStorage) load() error {
	// 加载任务
	if data, err := os.ReadFile(fs.tasksFile); err == nil {
		var payload tasksPersistData
		if _, legacy, err := persistence.ReadEnvelopeOrLegacy(data, tasksSchema, &payload); err != nil {
			var legacyTasks []*model.Task
			if _, _, legacyErr := persistence.ReadEnvelopeOrLegacy(data, "", &legacyTasks); legacyErr != nil {
				return fmt.Errorf("failed to unmarshal tasks: %w", err)
			}
			payload.Tasks = legacyTasks
		} else if legacy && len(payload.Tasks) == 0 {
			// Backward compatibility: legacy payload was a raw []*model.Task array.
			var legacyTasks []*model.Task
			if _, _, err := persistence.ReadEnvelopeOrLegacy(data, "", &legacyTasks); err != nil {
				return fmt.Errorf("failed to decode legacy tasks: %w", err)
			}
			payload.Tasks = legacyTasks
		}
		for _, task := range payload.Tasks {
			fs.tasks[task.ID] = task
		}
	}

	// 加载任务列表
	if data, err := os.ReadFile(fs.listsFile); err == nil {
		var payload listsPersistData
		if _, legacy, err := persistence.ReadEnvelopeOrLegacy(data, listsSchema, &payload); err != nil {
			var legacyLists []*model.TaskList
			if _, _, legacyErr := persistence.ReadEnvelopeOrLegacy(data, "", &legacyLists); legacyErr != nil {
				return fmt.Errorf("failed to unmarshal lists: %w", err)
			}
			payload.Lists = legacyLists
		} else if legacy && len(payload.Lists) == 0 {
			var legacyLists []*model.TaskList
			if _, _, err := persistence.ReadEnvelopeOrLegacy(data, "", &legacyLists); err != nil {
				return fmt.Errorf("failed to decode legacy lists: %w", err)
			}
			payload.Lists = legacyLists
		}
		for _, list := range payload.Lists {
			fs.taskLists[list.ID] = list
		}
	}

	// 加载同步时间
	if data, err := os.ReadFile(fs.syncFile); err == nil {
		var payload syncPersistData
		if _, legacy, err := persistence.ReadEnvelopeOrLegacy(data, syncSchema, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal sync times: %w", err)
		} else if legacy && len(payload.SyncTimes) == 0 {
			var legacySync map[string]string
			if _, _, err := persistence.ReadEnvelopeOrLegacy(data, "", &legacySync); err != nil {
				return fmt.Errorf("failed to decode legacy sync times: %w", err)
			}
			payload.SyncTimes = legacySync
		}
		for source, timeStr := range payload.SyncTimes {
			t, err := time.Parse(time.RFC3339, timeStr)
			if err == nil {
				fs.syncTimes[model.TaskSource(source)] = t
			}
		}
	}

	return nil
}

// save 保存数据到文件
func (fs *FileStorage) save() error {
	// 保存任务
	tasks := make([]*model.Task, 0, len(fs.tasks))
	for _, task := range fs.tasks {
		tasks = append(tasks, task)
	}
	if err := persistence.WriteEnvelopeAtomic(fs.tasksFile, tasksSchema, 1, tasksPersistData{Tasks: tasks}); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	// 保存任务列表
	lists := make([]*model.TaskList, 0, len(fs.taskLists))
	for _, list := range fs.taskLists {
		lists = append(lists, list)
	}
	if err := persistence.WriteEnvelopeAtomic(fs.listsFile, listsSchema, 1, listsPersistData{Lists: lists}); err != nil {
		return fmt.Errorf("failed to write lists file: %w", err)
	}

	// 保存同步时间
	syncData := make(map[string]string)
	for source, t := range fs.syncTimes {
		syncData[string(source)] = t.Format(time.RFC3339)
	}
	if err := persistence.WriteEnvelopeAtomic(fs.syncFile, syncSchema, 1, syncPersistData{SyncTimes: syncData}); err != nil {
		return fmt.Errorf("failed to write sync file: %w", err)
	}

	return nil
}

// SaveTask 保存任务
func (fs *FileStorage) SaveTask(_ context.Context, task *model.Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()

	stored := model.CloneTask(task)
	stored.UpdatedAt = time.Now()
	fs.tasks[stored.ID] = stored
	fs.dirty = true

	return nil
}

// GetTask 获取任务
func (fs *FileStorage) GetTask(_ context.Context, id string) (*model.Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	task, ok := fs.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	return model.CloneTask(task), nil
}

// ListTasks 列出任务
func (fs *FileStorage) ListTasks(_ context.Context, opts storage.ListOptions) ([]model.Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var result []model.Task
	for _, task := range fs.tasks {
		// 应用过滤条件
		if opts.Source != "" && task.Source != opts.Source {
			continue
		}
		if opts.ListID != "" && task.ListID != opts.ListID {
			continue
		}
		result = append(result, *model.CloneTask(task))
	}

	return result, nil
}

// DeleteTask 删除任务
func (fs *FileStorage) DeleteTask(ctx context.Context, id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	delete(fs.tasks, id)
	fs.dirty = true
	return nil
}

// SaveTasks 批量保存任务
func (fs *FileStorage) SaveTasks(ctx context.Context, tasks []*model.Task) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	now := time.Now()
	for _, task := range tasks {
		if task == nil {
			continue
		}
		stored := model.CloneTask(task)
		stored.UpdatedAt = now
		fs.tasks[stored.ID] = stored
	}

	// SaveTasks: immediate write + clear dirty
	if err := fs.save(); err != nil {
		return err
	}
	fs.dirty = false
	return nil
}

// QueryTasks 查询任务
func (fs *FileStorage) QueryTasks(ctx context.Context, query storage.Query) ([]model.Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return queryTasksFromMap(fs.tasks, query), nil
}

// SaveTaskList 保存任务列表
func (fs *FileStorage) SaveTaskList(ctx context.Context, list *model.TaskList) error {
	if list == nil {
		return fmt.Errorf("task list is nil")
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()

	stored := model.CloneTaskList(list)
	stored.UpdatedAt = time.Now()
	fs.taskLists[stored.ID] = stored
	fs.dirty = true

	return nil
}

// GetTaskList 获取任务列表
func (fs *FileStorage) GetTaskList(ctx context.Context, id string) (*model.TaskList, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	list, ok := fs.taskLists[id]
	if !ok {
		return nil, fmt.Errorf("task list not found: %s", id)
	}
	return model.CloneTaskList(list), nil
}

// ListTaskLists 列出任务列表
func (fs *FileStorage) ListTaskLists(ctx context.Context) ([]model.TaskList, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var result []model.TaskList
	for _, list := range fs.taskLists {
		result = append(result, *model.CloneTaskList(list))
	}
	return result, nil
}

// DeleteTaskList 删除任务列表
func (fs *FileStorage) DeleteTaskList(ctx context.Context, id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	delete(fs.taskLists, id)
	fs.dirty = true
	return nil
}

// ExportToJSON 导出为 JSON
func (fs *FileStorage) ExportToJSON(ctx context.Context, opts storage.ExportOptions) ([]byte, error) {
	tasks, err := fs.QueryTasks(ctx, opts.Query)
	if err != nil {
		return nil, err
	}

	if opts.Pretty {
		return json.MarshalIndent(tasks, "", "  ")
	}
	return json.Marshal(tasks)
}

// ExportToMarkdown 导出为 Markdown
func (fs *FileStorage) ExportToMarkdown(ctx context.Context, opts storage.ExportOptions) ([]byte, error) {
	tasks, err := fs.QueryTasks(ctx, opts.Query)
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	sb.WriteString("# Tasks\n\n")

	// 按象限分组
	quadrants := map[model.Quadrant][]model.Task{
		model.QuadrantUrgentImportant:       {},
		model.QuadrantNotUrgentImportant:    {},
		model.QuadrantUrgentNotImportant:    {},
		model.QuadrantNotUrgentNotImportant: {},
	}

	for _, task := range tasks {
		quadrants[task.Quadrant] = append(quadrants[task.Quadrant], task)
	}

	quadrantNames := map[model.Quadrant]string{
		model.QuadrantUrgentImportant:       "🔥 紧急且重要 (Q1)",
		model.QuadrantNotUrgentImportant:    "📋 重要不紧急 (Q2)",
		model.QuadrantUrgentNotImportant:    "⚡ 紧急不重要 (Q3)",
		model.QuadrantNotUrgentNotImportant: "🗑️ 不紧急不重要 (Q4)",
	}

	for q := model.Quadrant(1); q <= 4; q++ {
		fmt.Fprintf(&sb, "## %s\n\n", quadrantNames[q])
		for _, task := range quadrants[q] {
			status := " "
			if task.IsCompleted() {
				status = "x"
			}
			fmt.Fprintf(&sb, "- [%s] %s\n", status, task.Title)
			if task.DueDate != nil {
				fmt.Fprintf(&sb, "  - 截止日期: %s\n", task.DueDate.Format("2006-01-02"))
			}
			if task.Priority != model.PriorityNone {
				fmt.Fprintf(&sb, "  - 优先级: %s\n", task.Priority.String())
			}
		}
		sb.WriteString("\n")
	}

	return []byte(sb.String()), nil
}

// GetLastSyncTime 获取上次同步时间
func (fs *FileStorage) GetLastSyncTime(ctx context.Context, source model.TaskSource) (*time.Time, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	t, ok := fs.syncTimes[source]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

// SetLastSyncTime 设置上次同步时间
func (fs *FileStorage) SetLastSyncTime(ctx context.Context, source model.TaskSource, t time.Time) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.syncTimes[source] = t
	fs.dirty = true
	return nil
}

// Flush writes any pending changes to disk.
func (fs *FileStorage) Flush() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if !fs.dirty {
		return nil
	}
	if err := fs.save(); err != nil {
		return fmt.Errorf("flush failed: %w", err)
	}
	fs.dirty = false
	return nil
}

// Close flushes any pending changes and releases resources.
func (fs *FileStorage) Close() error {
	return fs.Flush()
}
