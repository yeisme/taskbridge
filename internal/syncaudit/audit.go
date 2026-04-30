package syncaudit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage"
)

const (
	SessionSchema  = "taskbridge.sync-session.v1"
	ConflictSchema = "taskbridge.conflict.v1"
)

type Session struct {
	Schema      string      `json:"schema"`
	ID          string      `json:"id"`
	Mode        string      `json:"mode"`
	Source      string      `json:"source"`
	Target      string      `json:"target"`
	DryRun      bool        `json:"dry_run"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt time.Time   `json:"completed_at"`
	Status      string      `json:"status"`
	Stats       Stats       `json:"stats"`
	Operations  []Operation `json:"operations"`
}

type Stats struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Deleted   int `json:"deleted"`
	Skipped   int `json:"skipped"`
	Conflicts int `json:"conflicts"`
	Errors    int `json:"errors"`
}

type Operation struct {
	OpID                 string   `json:"op_id"`
	Type                 string   `json:"type"`
	LocalID              string   `json:"local_id,omitempty"`
	ProviderID           string   `json:"provider_id,omitempty"`
	Title                string   `json:"title,omitempty"`
	BeforeHash           string   `json:"before_hash,omitempty"`
	AfterHash            string   `json:"after_hash,omitempty"`
	Reason               string   `json:"reason,omitempty"`
	Fields               []string `json:"fields,omitempty"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
}

type Conflict struct {
	Schema         string   `json:"schema"`
	ID             string   `json:"id"`
	LocalID        string   `json:"local_id"`
	Providers      []string `json:"providers"`
	FieldConflicts []string `json:"field_conflicts"`
	DetectedAt     string   `json:"detected_at"`
	Strategies     []string `json:"strategies"`
	Status         string   `json:"status"`
}

type Store struct {
	BasePath string
}

var idCounter uint64

func (s Store) Diff(ctx context.Context, taskStore storage.Storage, source, target string) (*Session, error) {
	started := time.Now()
	source = normalizeSource(source)
	target = normalizeSource(target)
	if err := validateDiffEndpoint(source, true); err != nil {
		return nil, err
	}
	if err := validateDiffEndpoint(target, false); err != nil {
		return nil, err
	}
	if target != "" && source == target {
		return nil, fmt.Errorf("source and target must be different: %s", source)
	}
	sourceTasks, err := queryTasksBySource(ctx, taskStore, source)
	if err != nil {
		return nil, err
	}
	targetTasks := []model.Task{}
	if target != "" {
		targetTasks, err = queryTasksBySource(ctx, taskStore, target)
		if err != nil {
			return nil, err
		}
	}
	ops, stats := compareSnapshots(sourceTasks, targetTasks)
	return &Session{
		Schema:      SessionSchema,
		ID:          newID("diff"),
		Mode:        "diff",
		Source:      source,
		Target:      target,
		DryRun:      true,
		StartedAt:   started,
		CompletedAt: time.Now(),
		Status:      "completed",
		Stats:       stats,
		Operations:  ops,
	}, nil
}

func queryTasksBySource(ctx context.Context, taskStore storage.Storage, source string) ([]model.Task, error) {
	tasks, err := taskStore.QueryTasks(ctx, storage.Query{Sources: []model.TaskSource{model.TaskSource(source)}})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Title == tasks[j].Title {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].Title < tasks[j].Title
	})
	return tasks, nil
}

func compareSnapshots(sourceTasks, targetTasks []model.Task) ([]Operation, Stats) {
	index := newTaskIndex(targetTasks)
	consumed := map[string]bool{}
	conflictConsumed := map[string]bool{}
	ops := make([]Operation, 0, len(sourceTasks)+len(targetTasks))
	stats := Stats{}

	for _, sourceTask := range sourceTasks {
		candidates := index.match(sourceTask, consumed)
		switch len(candidates) {
		case 0:
			op := operationFromTask("create", sourceTask, "target snapshot has no matching task")
			ops = append(ops, op)
			stats.Created++
		case 1:
			targetTask := candidates[0]
			consumed[targetTask.ID] = true
			fields := changedFields(sourceTask, targetTask)
			if len(fields) == 0 {
				op := operationFromTask("skip", sourceTask, "source and target task content match")
				op.ProviderID = targetTask.SourceRawID
				ops = append(ops, op)
				stats.Skipped++
				continue
			}
			op := operationFromTask("update", sourceTask, "target task differs from source snapshot")
			op.ProviderID = targetTask.SourceRawID
			op.Fields = fields
			ops = append(ops, op)
			stats.Updated++
		default:
			for _, candidate := range candidates {
				conflictConsumed[candidate.ID] = true
			}
			op := operationFromTask("conflict", sourceTask, "multiple target candidates match this source task")
			op.RequiresConfirmation = true
			ops = append(ops, op)
			stats.Conflicts++
		}
	}

	for _, targetTask := range targetTasks {
		if consumed[targetTask.ID] || conflictConsumed[targetTask.ID] {
			continue
		}
		op := operationFromTask("delete", targetTask, "target task has no matching source task")
		op.RequiresConfirmation = true
		ops = append(ops, op)
		stats.Deleted++
	}
	for i := range ops {
		ops[i].OpID = fmt.Sprintf("op_%03d", i+1)
	}
	return ops, stats
}

type taskIndex struct {
	byRawID   map[string][]model.Task
	byLocalID map[string][]model.Task
	byTitle   map[string][]model.Task
}

func newTaskIndex(tasks []model.Task) taskIndex {
	index := taskIndex{
		byRawID:   map[string][]model.Task{},
		byLocalID: map[string][]model.Task{},
		byTitle:   map[string][]model.Task{},
	}
	for _, task := range tasks {
		if strings.TrimSpace(task.SourceRawID) != "" {
			index.byRawID[task.SourceRawID] = append(index.byRawID[task.SourceRawID], task)
		}
		if id := taskLocalID(task); id != "" {
			index.byLocalID[id] = append(index.byLocalID[id], task)
		}
		if title := normalizeTitle(task.Title); title != "" {
			index.byTitle[title] = append(index.byTitle[title], task)
		}
	}
	return index
}

func (i taskIndex) match(task model.Task, consumed map[string]bool) []model.Task {
	if strings.TrimSpace(task.SourceRawID) != "" {
		if candidates := unconsumed(i.byRawID[task.SourceRawID], consumed); len(candidates) > 0 {
			return candidates
		}
	}
	if id := taskLocalID(task); id != "" {
		if candidates := unconsumed(i.byLocalID[id], consumed); len(candidates) > 0 {
			return candidates
		}
	}
	return unconsumed(i.byTitle[normalizeTitle(task.Title)], consumed)
}

func unconsumed(tasks []model.Task, consumed map[string]bool) []model.Task {
	result := make([]model.Task, 0, len(tasks))
	for _, task := range tasks {
		if consumed[task.ID] {
			continue
		}
		result = append(result, task)
	}
	return result
}

func changedFields(sourceTask, targetTask model.Task) []string {
	fields := []string{}
	if strings.TrimSpace(sourceTask.Title) != strings.TrimSpace(targetTask.Title) {
		fields = append(fields, "title")
	}
	if sourceTask.Status != targetTask.Status {
		fields = append(fields, "status")
	}
	if !sameDate(sourceTask.DueDate, targetTask.DueDate) {
		fields = append(fields, "due_date")
	}
	if sourceTask.Priority != targetTask.Priority {
		fields = append(fields, "priority")
	}
	if sourceTask.EstimatedMinutes != targetTask.EstimatedMinutes {
		fields = append(fields, "estimated_minutes")
	}
	return fields
}

func sameDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func operationFromTask(opType string, task model.Task, reason string) Operation {
	return Operation{
		Type:       opType,
		LocalID:    task.ID,
		ProviderID: task.SourceRawID,
		Title:      task.Title,
		Reason:     reason,
	}
}

func taskLocalID(task model.Task) string {
	if task.Metadata == nil {
		return ""
	}
	return strings.TrimSpace(task.Metadata.LocalID)
}

func normalizeTitle(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

func (s Store) SaveSession(session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	dir := filepath.Join(s.BasePath, "audit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, session.ID+".json"), data, 0o644)
}

func (s Store) LoadSession(id string) (*Session, error) {
	data, err := os.ReadFile(filepath.Join(s.BasePath, "audit", id+".json"))
	if err != nil {
		return nil, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s Store) ListConflicts() ([]Conflict, error) {
	path := filepath.Join(s.BasePath, "conflicts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Conflict{}, nil
		}
		return nil, err
	}
	var conflicts []Conflict
	if err := json.Unmarshal(data, &conflicts); err != nil {
		return nil, err
	}
	return conflicts, nil
}

func (s Store) ResolveConflict(id, strategy string) (*Conflict, error) {
	conflicts, err := s.ListConflicts()
	if err != nil {
		return nil, err
	}
	for i := range conflicts {
		if conflicts[i].ID == id {
			conflicts[i].Status = "resolved:" + strategy
			if err := s.writeConflicts(conflicts); err != nil {
				return nil, err
			}
			return &conflicts[i], nil
		}
	}
	return nil, fmt.Errorf("conflict not found: %s", id)
}

func (s Store) CreateBackup() (map[string]interface{}, error) {
	id := newID("backup")
	dir := filepath.Join(s.BasePath, "backups", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	files := []string{"tasks.json", "lists.json", "sync.json", "projects.json", "conflicts.json"}
	copied := make([]string, 0)
	for _, name := range files {
		src := filepath.Join(s.BasePath, name)
		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return nil, err
		}
		copied = append(copied, name)
	}
	return map[string]interface{}{"schema": "taskbridge.backup.v1", "id": id, "path": dir, "files": copied}, nil
}

func (s Store) RestoreBackup(id string) (map[string]interface{}, error) {
	dir := filepath.Join(s.BasePath, "backups", id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	restored := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(s.BasePath, entry.Name()), data, 0o644); err != nil {
			return nil, err
		}
		restored = append(restored, entry.Name())
	}
	return map[string]interface{}{"schema": "taskbridge.restore.v1", "id": id, "restored": restored}, nil
}

func (s Store) writeConflicts(conflicts []Conflict) error {
	data, err := json.MarshalIndent(conflicts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.BasePath, "conflicts.json"), data, 0o644)
}

func normalizeSource(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return provider.ResolveProviderName(value)
}

func validateDiffEndpoint(value string, required bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("source provider is required")
		}
		return nil
	}
	if value == string(model.SourceLocal) || provider.IsValidProvider(value) {
		return nil
	}
	return fmt.Errorf("invalid provider: %s", value)
}

func newID(prefix string) string {
	seq := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s_%s_%06d", prefix, time.Now().Format("20060102_150405.000000000"), seq)
}
