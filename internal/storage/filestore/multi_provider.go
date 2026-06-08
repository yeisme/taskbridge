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
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
)

const (
	manifestSchema     = "taskbridge.storage.manifest"
	syncStateSchema    = "taskbridge.storage.sync-state"
	mappingsSchema     = "taskbridge.storage.mappings"
	providerMetaSchema = "taskbridge.storage.provider-meta"
)

// MultiProviderStorage 多 Provider 存储实现
type MultiProviderStorage struct {
	mu sync.RWMutex

	// basePath 存储基础路径
	basePath string
	// format 存储格式
	format string

	// 全局文件路径
	manifestFile  string
	syncStateFile string
	mappingsFile  string

	// 全局数据
	manifest  *model.Manifest
	syncState *model.SyncState
	mappings  *model.MappingDatabase

	// Provider 数据缓存
	providerData map[string]*ProviderStorage
}

// ProviderStorage 单个 Provider 的存储
type ProviderStorage struct {
	mu sync.RWMutex

	provider string
	basePath string

	tasksFile string
	listsFile string
	metaFile  string

	dirty     bool
	tasks     map[string]*model.Task
	taskLists map[string]*model.TaskList
	meta      *model.ProviderData
}

// NewMultiProviderStorage 创建多 Provider 存储实例
func NewMultiProviderStorage(basePath, format string) (*MultiProviderStorage, error) {
	mps := &MultiProviderStorage{
		basePath:      basePath,
		format:        format,
		manifestFile:  filepath.Join(basePath, "manifest.json"),
		syncStateFile: filepath.Join(basePath, "sync-state.json"),
		mappingsFile:  filepath.Join(basePath, "mappings.json"),
		providerData:  make(map[string]*ProviderStorage),
	}

	// 确保目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 确保providers目录存在
	providersDir := filepath.Join(basePath, "providers")
	if err := os.MkdirAll(providersDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create providers directory: %w", err)
	}

	// 加载全局数据
	if err := mps.loadGlobalData(); err != nil {
		return nil, fmt.Errorf("failed to load global data: %w", err)
	}

	return mps, nil
}

// loadGlobalData 加载全局数据
func (mps *MultiProviderStorage) loadGlobalData() error {
	// 加载清单
	if data, err := os.ReadFile(mps.manifestFile); err == nil {
		var manifest model.Manifest
		if _, _, err := persistence.ReadEnvelopeOrLegacy(data, manifestSchema, &manifest); err != nil {
			return fmt.Errorf("failed to unmarshal manifest: %w", err)
		}
		mps.manifest = &manifest
	} else if os.IsNotExist(err) {
		mps.manifest = model.NewManifest()
	} else {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// 加载同步状态
	if data, err := os.ReadFile(mps.syncStateFile); err == nil {
		var syncState model.SyncState
		if _, _, err := persistence.ReadEnvelopeOrLegacy(data, syncStateSchema, &syncState); err != nil {
			return fmt.Errorf("failed to unmarshal sync state: %w", err)
		}
		mps.syncState = &syncState
	} else if os.IsNotExist(err) {
		mps.syncState = model.NewSyncState()
	} else {
		return fmt.Errorf("failed to read sync state: %w", err)
	}

	// 加载映射
	if data, err := os.ReadFile(mps.mappingsFile); err == nil {
		var mappings model.MappingDatabase
		if _, _, err := persistence.ReadEnvelopeOrLegacy(data, mappingsSchema, &mappings); err != nil {
			return fmt.Errorf("failed to unmarshal mappings: %w", err)
		}
		mps.mappings = &mappings
	} else if os.IsNotExist(err) {
		mps.mappings = model.NewMappingDatabase()
	} else {
		return fmt.Errorf("failed to read mappings: %w", err)
	}

	return nil
}

// saveGlobalData 保存全局数据
func (mps *MultiProviderStorage) saveGlobalData() error {
	// 保存清单
	mps.manifest.UpdatedAt = time.Now()
	if err := persistence.WriteEnvelopeAtomic(mps.manifestFile, manifestSchema, 1, mps.manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// 保存同步状态
	if err := persistence.WriteEnvelopeAtomic(mps.syncStateFile, syncStateSchema, 1, mps.syncState); err != nil {
		return fmt.Errorf("failed to write sync state: %w", err)
	}

	// 保存映射
	if err := persistence.WriteEnvelopeAtomic(mps.mappingsFile, mappingsSchema, 1, mps.mappings); err != nil {
		return fmt.Errorf("failed to write mappings: %w", err)
	}

	return nil
}

// GetProviderStorage 获取指定 Provider 的存储
func (mps *MultiProviderStorage) GetProviderStorage(provider string) (*ProviderStorage, error) {
	mps.mu.Lock()
	defer mps.mu.Unlock()

	// 检查缓存
	if ps, ok := mps.providerData[provider]; ok {
		return ps, nil
	}

	// 创建新的 Provider 存储
	ps, err := NewProviderStorage(provider, filepath.Join(mps.basePath, "providers", provider))
	if err != nil {
		return nil, err
	}

	mps.providerData[provider] = ps
	return ps, nil
}

// GetManifest 获取清单
func (mps *MultiProviderStorage) GetManifest() *model.Manifest {
	mps.mu.RLock()
	defer mps.mu.RUnlock()
	return mps.manifest
}

// UpdateManifest 更新清单
func (mps *MultiProviderStorage) UpdateManifest(fn func(*model.Manifest)) error {
	mps.mu.Lock()
	defer mps.mu.Unlock()

	fn(mps.manifest)
	return mps.saveGlobalData()
}

// GetSyncState 获取同步状态
func (mps *MultiProviderStorage) GetSyncState() *model.SyncState {
	mps.mu.RLock()
	defer mps.mu.RUnlock()
	return mps.syncState
}

// AddSyncSession 添加同步会话
func (mps *MultiProviderStorage) AddSyncSession(session model.SyncSession) error {
	mps.mu.Lock()
	defer mps.mu.Unlock()

	mps.syncState.SyncSessions = append(mps.syncState.SyncSessions, session)
	// 只保留最近100个会话
	if len(mps.syncState.SyncSessions) > 100 {
		mps.syncState.SyncSessions = mps.syncState.SyncSessions[len(mps.syncState.SyncSessions)-100:]
	}
	return mps.saveGlobalData()
}

// AddPendingOperation 添加待处理操作
func (mps *MultiProviderStorage) AddPendingOperation(op model.PendingOp) error {
	mps.mu.Lock()
	defer mps.mu.Unlock()

	mps.syncState.PendingOperations = append(mps.syncState.PendingOperations, op)
	return mps.saveGlobalData()
}

// GetMappings 获取映射数据库
func (mps *MultiProviderStorage) GetMappings() *model.MappingDatabase {
	mps.mu.RLock()
	defer mps.mu.RUnlock()
	return mps.mappings
}

// FindMappingByProviderID 根据 Provider 任务ID 查找映射
func (mps *MultiProviderStorage) FindMappingByProviderID(provider, taskID string) *model.TaskMapping {
	mps.mu.RLock()
	defer mps.mu.RUnlock()

	for i := range mps.mappings.Mappings {
		if ref, ok := mps.mappings.Mappings[i].Providers[provider]; ok {
			if ref.ID == taskID {
				return &mps.mappings.Mappings[i]
			}
		}
	}
	return nil
}

// UpdateMapping 更新任务映射
func (mps *MultiProviderStorage) UpdateMapping(mapping model.TaskMapping) error {
	mps.mu.Lock()
	defer mps.mu.Unlock()

	// 查找现有映射
	for i, m := range mps.mappings.Mappings {
		if m.LocalID == mapping.LocalID {
			mps.mappings.Mappings[i] = mapping
			return mps.saveGlobalData()
		}
	}

	// 添加新映射
	mps.mappings.Mappings = append(mps.mappings.Mappings, mapping)
	return mps.saveGlobalData()
}

// RemoveMapping 删除任务映射
func (mps *MultiProviderStorage) RemoveMapping(localID string) error {
	mps.mu.Lock()
	defer mps.mu.Unlock()

	for i, m := range mps.mappings.Mappings {
		if m.LocalID == localID {
			mps.mappings.Mappings = append(
				mps.mappings.Mappings[:i],
				mps.mappings.Mappings[i+1:]...,
			)
			return mps.saveGlobalData()
		}
	}
	return nil
}

// --- 实现 storage.Storage 接口 ---

// SaveTask 保存任务
func (mps *MultiProviderStorage) SaveTask(ctx context.Context, task *model.Task) error {
	ps, err := mps.GetProviderStorage(string(task.Source))
	if err != nil {
		return err
	}
	return ps.SaveTask(ctx, task)
}

// GetTask 获取任务
func (mps *MultiProviderStorage) GetTask(ctx context.Context, id string) (*model.Task, error) {
	// 从映射中查找任务所属的 Provider
	mps.mu.RLock()
	for _, m := range mps.mappings.Mappings {
		if m.LocalID == id {
			for provider := range m.Providers {
				mps.mu.RUnlock()
				ps, err := mps.GetProviderStorage(provider)
				if err != nil {
					return nil, err
				}
				return ps.GetTask(ctx, id)
			}
		}
	}
	mps.mu.RUnlock()
	return nil, fmt.Errorf("task not found: %s", id)
}

// ListTasks 列出任务
func (mps *MultiProviderStorage) ListTasks(ctx context.Context, opts storage.ListOptions) ([]model.Task, error) {
	if opts.Source != "" {
		ps, err := mps.GetProviderStorage(string(opts.Source))
		if err != nil {
			return nil, err
		}
		return ps.ListTasks(ctx, opts)
	}

	// 汇总所有 Provider 的任务
	var result []model.Task
	for _, p := range provider.GetAllProviderNames() {
		ps, err := mps.GetProviderStorage(p)
		if err != nil {
			continue
		}
		tasks, err := ps.ListTasks(ctx, opts)
		if err != nil {
			continue
		}
		result = append(result, tasks...)
	}
	return result, nil
}

// DeleteTask 删除任务
func (mps *MultiProviderStorage) DeleteTask(ctx context.Context, id string) error {
	// 从映射中查找任务所属的 Provider
	mps.mu.RLock()
	for _, m := range mps.mappings.Mappings {
		if m.LocalID == id {
			for provider := range m.Providers {
				mps.mu.RUnlock()
				ps, err := mps.GetProviderStorage(provider)
				if err != nil {
					return err
				}
				return ps.DeleteTask(ctx, id)
			}
		}
	}
	mps.mu.RUnlock()
	return fmt.Errorf("task not found: %s", id)
}

// SaveTasks 批量保存任务
func (mps *MultiProviderStorage) SaveTasks(ctx context.Context, tasks []*model.Task) error {
	// 按 Provider 分组
	byProvider := make(map[model.TaskSource][]*model.Task)
	for _, task := range tasks {
		byProvider[task.Source] = append(byProvider[task.Source], task)
	}

	// 分别保存
	for source, providerTasks := range byProvider {
		ps, err := mps.GetProviderStorage(string(source))
		if err != nil {
			return err
		}
		if err := ps.SaveTasks(ctx, providerTasks); err != nil {
			return err
		}
	}
	return nil
}

// QueryTasks 查询任务
func (mps *MultiProviderStorage) QueryTasks(ctx context.Context, query storage.Query) ([]model.Task, error) {
	var result []model.Task

	providers := provider.GetAllProviderNames()
	if len(query.Sources) > 0 {
		providers = make([]string, len(query.Sources))
		for i, s := range query.Sources {
			providers[i] = string(s)
		}
	}

	for _, p := range providers {
		ps, err := mps.GetProviderStorage(p)
		if err != nil {
			continue
		}
		// Provider 级查询时取消分页和排序，最后做统一排序/分页。
		providerQuery := query
		providerQuery.Limit = 0
		providerQuery.Offset = 0
		providerQuery.OrderBy = ""
		providerQuery.OrderDesc = false

		tasks, err := ps.QueryTasks(ctx, providerQuery)
		if err != nil {
			continue
		}
		result = append(result, tasks...)
	}

	if query.OrderBy != "" {
		sortTasks(result, query.OrderBy, query.OrderDesc)
	}

	return applyPagination(result, query.Offset, query.Limit), nil
}

// SaveTaskList 保存任务列表
func (mps *MultiProviderStorage) SaveTaskList(ctx context.Context, list *model.TaskList) error {
	ps, err := mps.GetProviderStorage(string(list.Source))
	if err != nil {
		return err
	}
	return ps.SaveTaskList(ctx, list)
}

// GetTaskList 获取任务列表
func (mps *MultiProviderStorage) GetTaskList(ctx context.Context, id string) (*model.TaskList, error) {
	for _, p := range provider.GetAllProviderNames() {
		ps, err := mps.GetProviderStorage(p)
		if err != nil {
			continue
		}
		list, err := ps.GetTaskList(ctx, id)
		if err == nil {
			return list, nil
		}
	}
	return nil, fmt.Errorf("task list not found: %s", id)
}

// ListTaskLists 列出任务列表
func (mps *MultiProviderStorage) ListTaskLists(ctx context.Context) ([]model.TaskList, error) {
	var result []model.TaskList
	for _, p := range provider.GetAllProviderNames() {
		ps, err := mps.GetProviderStorage(p)
		if err != nil {
			continue
		}
		lists, err := ps.ListTaskLists(ctx)
		if err != nil {
			continue
		}
		result = append(result, lists...)
	}
	return result, nil
}

// DeleteTaskList 删除任务列表
func (mps *MultiProviderStorage) DeleteTaskList(ctx context.Context, id string) error {
	for _, p := range provider.GetAllProviderNames() {
		ps, err := mps.GetProviderStorage(p)
		if err != nil {
			continue
		}
		if err := ps.DeleteTaskList(ctx, id); err == nil {
			return nil
		}
	}
	return fmt.Errorf("task list not found: %s", id)
}

// ExportToJSON 导出为 JSON
func (mps *MultiProviderStorage) ExportToJSON(ctx context.Context, opts storage.ExportOptions) ([]byte, error) {
	tasks, err := mps.QueryTasks(ctx, opts.Query)
	if err != nil {
		return nil, err
	}

	if opts.Pretty {
		return json.MarshalIndent(tasks, "", "  ")
	}
	return json.Marshal(tasks)
}

// ExportToMarkdown 导出为 Markdown
func (mps *MultiProviderStorage) ExportToMarkdown(ctx context.Context, opts storage.ExportOptions) ([]byte, error) {
	// 复用 FileStorage 的实现
	fs := &FileStorage{}
	return fs.ExportToMarkdown(ctx, opts)
}

// GetLastSyncTime 获取上次同步时间
func (mps *MultiProviderStorage) GetLastSyncTime(ctx context.Context, source model.TaskSource) (*time.Time, error) {
	mps.mu.RLock()
	defer mps.mu.RUnlock()

	if meta, ok := mps.manifest.Providers[string(source)]; ok {
		if !meta.LastSync.IsZero() {
			return &meta.LastSync, nil
		}
	}
	return nil, nil
}

// SetLastSyncTime 设置上次同步时间
func (mps *MultiProviderStorage) SetLastSyncTime(ctx context.Context, source model.TaskSource, t time.Time) error {
	return mps.UpdateManifest(func(m *model.Manifest) {
		meta := m.Providers[string(source)]
		meta.LastSync = t
		m.Providers[string(source)] = meta
	})
}

// Flush flushes all loaded provider stores.
func (mps *MultiProviderStorage) Flush() error {
	mps.mu.RLock()
	defer mps.mu.RUnlock()
	var errs []string
	for name, ps := range mps.providerData {
		if err := ps.Flush(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("flush errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Close flushes and closes all loaded provider stores.
func (mps *MultiProviderStorage) Close() error {
	mps.mu.RLock()
	defer mps.mu.RUnlock()
	var errs []string
	for name, ps := range mps.providerData {
		if err := ps.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// --- ProviderStorage 方法 ---

// NewProviderStorage 创建 Provider 存储
func NewProviderStorage(provider, basePath string) (*ProviderStorage, error) {
	ps := &ProviderStorage{
		provider:  provider,
		basePath:  basePath,
		tasksFile: filepath.Join(basePath, "tasks.json"),
		listsFile: filepath.Join(basePath, "lists.json"),
		metaFile:  filepath.Join(basePath, "meta.json"),
		tasks:     make(map[string]*model.Task),
		taskLists: make(map[string]*model.TaskList),
	}

	// 确保目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create provider directory: %w", err)
	}

	// 加载数据
	if err := ps.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load provider data: %w", err)
	}

	return ps, nil
}

// load 加载 Provider 数据
func (ps *ProviderStorage) load() error {
	// 加载任务
	if data, err := os.ReadFile(ps.tasksFile); err == nil {
		var payload tasksPersistData
		if _, legacy, err := persistence.ReadEnvelopeOrLegacy(data, tasksSchema, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal tasks: %w", err)
		} else if legacy && len(payload.Tasks) == 0 {
			var legacyTasks []*model.Task
			if _, _, err := persistence.ReadEnvelopeOrLegacy(data, "", &legacyTasks); err != nil {
				return fmt.Errorf("failed to decode legacy tasks: %w", err)
			}
			payload.Tasks = legacyTasks
		}
		for _, task := range payload.Tasks {
			ps.tasks[task.ID] = task
		}
	}

	// 加载列表
	if data, err := os.ReadFile(ps.listsFile); err == nil {
		var payload listsPersistData
		if _, legacy, err := persistence.ReadEnvelopeOrLegacy(data, listsSchema, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal lists: %w", err)
		} else if legacy && len(payload.Lists) == 0 {
			var legacyLists []*model.TaskList
			if _, _, err := persistence.ReadEnvelopeOrLegacy(data, "", &legacyLists); err != nil {
				return fmt.Errorf("failed to decode legacy lists: %w", err)
			}
			payload.Lists = legacyLists
		}
		for _, list := range payload.Lists {
			ps.taskLists[list.ID] = list
		}
	}

	// 加载元数据
	if data, err := os.ReadFile(ps.metaFile); err == nil {
		var meta model.ProviderData
		if _, _, err := persistence.ReadEnvelopeOrLegacy(data, providerMetaSchema, &meta); err != nil {
			return fmt.Errorf("failed to unmarshal meta: %w", err)
		}
		ps.meta = &meta
	} else if os.IsNotExist(err) {
		ps.meta = &model.ProviderData{
			Provider:     ps.provider,
			Capabilities: model.Capabilities{},
		}
	}

	return nil
}

// save 保存 Provider 数据
func (ps *ProviderStorage) save() error {
	// 保存任务
	tasks := make([]*model.Task, 0, len(ps.tasks))
	for _, task := range ps.tasks {
		tasks = append(tasks, task)
	}
	if err := persistence.WriteEnvelopeAtomic(ps.tasksFile, tasksSchema, 1, tasksPersistData{Tasks: tasks}); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	// 保存列表
	lists := make([]*model.TaskList, 0, len(ps.taskLists))
	for _, list := range ps.taskLists {
		lists = append(lists, list)
	}
	if err := persistence.WriteEnvelopeAtomic(ps.listsFile, listsSchema, 1, listsPersistData{Lists: lists}); err != nil {
		return fmt.Errorf("failed to write lists file: %w", err)
	}

	// 保存元数据
	if err := persistence.WriteEnvelopeAtomic(ps.metaFile, providerMetaSchema, 1, ps.meta); err != nil {
		return fmt.Errorf("failed to write meta file: %w", err)
	}

	return nil
}

// SaveTask 保存任务
func (ps *ProviderStorage) SaveTask(_ context.Context, task *model.Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	stored := model.CloneTask(task)
	stored.UpdatedAt = time.Now()
	ps.tasks[stored.ID] = stored
	ps.dirty = true

	return nil
}

// GetTask 获取任务
func (ps *ProviderStorage) GetTask(_ context.Context, id string) (*model.Task, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	task, ok := ps.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	return model.CloneTask(task), nil
}

// ListTasks 列出任务
func (ps *ProviderStorage) ListTasks(_ context.Context, opts storage.ListOptions) ([]model.Task, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var result []model.Task
	for _, task := range ps.tasks {
		if opts.ListID != "" && task.ListID != opts.ListID {
			continue
		}
		result = append(result, *model.CloneTask(task))
	}
	return result, nil
}

// DeleteTask 删除任务
func (ps *ProviderStorage) DeleteTask(_ context.Context, id string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.tasks, id)
	ps.dirty = true
	return nil
}

// SaveTasks 批量保存任务
func (ps *ProviderStorage) SaveTasks(_ context.Context, tasks []*model.Task) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	for _, task := range tasks {
		if task == nil {
			continue
		}
		stored := model.CloneTask(task)
		stored.UpdatedAt = now
		ps.tasks[stored.ID] = stored
	}

	// SaveTasks: immediate write + clear dirty
	if err := ps.save(); err != nil {
		return err
	}
	ps.dirty = false
	return nil
}

// QueryTasks 查询任务
func (ps *ProviderStorage) QueryTasks(_ context.Context, query storage.Query) ([]model.Task, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return queryTasksFromMap(ps.tasks, query), nil
}

// SaveTaskList 保存任务列表
func (ps *ProviderStorage) SaveTaskList(_ context.Context, list *model.TaskList) error {
	if list == nil {
		return fmt.Errorf("task list is nil")
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	stored := model.CloneTaskList(list)
	stored.UpdatedAt = time.Now()
	ps.taskLists[stored.ID] = stored
	ps.dirty = true

	return nil
}

// GetTaskList 获取任务列表
func (ps *ProviderStorage) GetTaskList(_ context.Context, id string) (*model.TaskList, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	list, ok := ps.taskLists[id]
	if !ok {
		return nil, fmt.Errorf("task list not found: %s", id)
	}
	return model.CloneTaskList(list), nil
}

// ListTaskLists 列出任务列表
func (ps *ProviderStorage) ListTaskLists(_ context.Context) ([]model.TaskList, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var result []model.TaskList
	for _, list := range ps.taskLists {
		result = append(result, *model.CloneTaskList(list))
	}
	return result, nil
}

// DeleteTaskList 删除任务列表
func (ps *ProviderStorage) DeleteTaskList(_ context.Context, id string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.taskLists, id)
	ps.dirty = true
	return nil
}

// GetMeta 获取元数据
func (ps *ProviderStorage) GetMeta() *model.ProviderData {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.meta
}

// UpdateMeta 更新元数据
func (ps *ProviderStorage) UpdateMeta(fn func(*model.ProviderData)) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	fn(ps.meta)
	ps.dirty = true
	return nil
}

// Flush writes any pending changes to disk.
func (ps *ProviderStorage) Flush() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if !ps.dirty {
		return nil
	}
	if err := ps.save(); err != nil {
		return fmt.Errorf("flush failed: %w", err)
	}
	ps.dirty = false
	return nil
}

// Close flushes any pending changes and releases resources.
func (ps *ProviderStorage) Close() error {
	return ps.Flush()
}
