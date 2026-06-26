package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/loader"
	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
	"github.com/yeisme/taskbridge/internal/sync"
	"github.com/yeisme/taskbridge/internal/syncaudit"
	"github.com/yeisme/taskbridge/pkg/config"
)

func TestRenderSyncResultUsesSummaryAndMetricsTable(t *testing.T) {
	oldDryRun := syncDryRun
	syncDryRun = false
	defer func() { syncDryRun = oldDryRun }()

	projection := buildSyncResultProjection(&sync.Result{
		Provider:     "google",
		Direction:    sync.DirectionPull,
		Pulled:       2,
		Pushed:       0,
		Updated:      1,
		Deleted:      0,
		Skipped:      3,
		Duration:     2 * time.Second,
		LastSyncTime: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	})

	output := renderSyncResult(projection)
	for _, want := range []string{"Status", "Sync pull completed for google.", "Sync result", "Metric", "Value", "Pulled", "2", "Written", "3"} {
		if !strings.Contains(output, want) {
			t.Fatalf("sync result output missing %q:\n%s", want, output)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("sync result default output should be human text, not JSON:\n%s", output)
	}
}

func TestRenderSyncResultDryRunSeparatesPlannedAndActualWrites(t *testing.T) {
	oldDryRun := syncDryRun
	syncDryRun = true
	defer func() { syncDryRun = oldDryRun }()

	projection := buildSyncResultProjection(&sync.Result{
		Provider:  "google",
		Direction: sync.DirectionPush,
		Pushed:    1,
		Updated:   1,
		Deleted:   1,
		Duration:  time.Second,
	})

	if projection.Facts["planned_writes"] != 3 {
		t.Fatalf("expected planned_writes=3, got %#v", projection.Facts["planned_writes"])
	}
	if projection.Facts["written"] != 0 {
		t.Fatalf("expected written=0 in dry-run, got %#v", projection.Facts["written"])
	}
	output := renderSyncResult(projection)
	for _, want := range []string{"preview completed", "Planned writes", "3", "Written", "0"} {
		if !strings.Contains(output, want) {
			t.Fatalf("dry-run sync output missing %q:\n%s", want, output)
		}
	}
}

func TestRenderSyncStatusUsesProviderTable(t *testing.T) {
	projection := buildSyncStatusProjection([]*sync.Status{{
		Provider:       "google",
		Authenticated:  true,
		LastSyncTime:   time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
		PendingChanges: 4,
	}})

	output := renderSyncStatus(projection)
	for _, want := range []string{"Sync providers", "Provider", "Name", "Authenticated", "Pending", "google", "Google Tasks", "yes", "4"} {
		if !strings.Contains(output, want) {
			t.Fatalf("sync status output missing %q:\n%s", want, output)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("sync status default output should be human text, not JSON:\n%s", output)
	}
}

func TestRenderSyncControlProjectionUsesAuditTable(t *testing.T) {
	projection := buildSyncDiffProjection(&syncaudit.Session{
		ID:     "diff-123",
		Source: "local",
		Target: "google",
		DryRun: true,
		Stats:  syncaudit.Stats{Created: 1, Updated: 2, Deleted: 0, Skipped: 3, Conflicts: 1, Errors: 0},
		Operations: []syncaudit.Operation{
			{Type: "create", Title: "Write report", Reason: "target snapshot has no matching task"},
		},
	})

	output := renderSyncControlProjection(projection)
	for _, want := range []string{"Sync audit", "Metric", "Value", "Created", "1", "Planned writes", "3", "Written", "0"} {
		if !strings.Contains(output, want) {
			t.Fatalf("sync control output missing %q:\n%s", want, output)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Fatalf("sync control default output should be human text, not JSON:\n%s", output)
	}
}

func TestSyncPullAllAggregatesSuccessFailureAndSkippedProviders(t *testing.T) {
	oldDryRun := syncDryRun
	syncDryRun = true
	defer func() { syncDryRun = oldDryRun }()

	store, err := filestore.New(t.TempDir(), "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	success := &syncAllMockProvider{name: "google", lists: []model.TaskList{{ID: "inbox", Name: "Inbox", Source: model.SourceGoogle}}, tasks: []model.Task{{ID: "g1", Title: "Google task", Status: model.StatusTodo, Source: model.SourceGoogle}}}
	failing := &syncAllMockProvider{name: "todoist", listErr: errors.New("remote unavailable")}
	loadResult := &loader.ProviderLoadResult{
		Providers: map[string]provider.Provider{"google": success, "todoist": failing},
		Statuses: map[string]*loader.ProviderLoadStatus{
			"google":    {Name: "google", Loaded: true, Authenticated: true},
			"todoist":   {Name: "todoist", Loaded: true, Authenticated: true},
			"microsoft": {Name: "microsoft", Error: "missing token"},
		},
	}

	receipt := aggregateSyncPullAll(context.Background(), loadResult, store)
	if receipt.ProvidersAttempted != 2 || receipt.ProvidersSucceeded != 1 || receipt.ProvidersFailed != 1 || receipt.ProvidersSkipped == 0 {
		t.Fatalf("unexpected aggregate counts: %+v", receipt)
	}
	if success.remoteWrites != 0 || failing.remoteWrites != 0 {
		t.Fatalf("pull all must not call remote write APIs: success=%d failing=%d", success.remoteWrites, failing.remoteWrites)
	}
	projection := buildSyncPullAllProjection(receipt)
	if projection.Status != "partial" {
		t.Fatalf("projection status = %q, want partial", projection.Status)
	}
	data, err := json.Marshal(projection.Data)
	if err != nil || !strings.Contains(string(data), "providers_attempted") {
		t.Fatalf("projection data should marshal with aggregate fields: %s err=%v", data, err)
	}
}

func TestSyncPullAllNoAuthenticatedProvidersReturnsSkippedSummary(t *testing.T) {
	store, err := filestore.New(t.TempDir(), "json")
	if err != nil {
		t.Fatalf("filestore.New: %v", err)
	}
	receipt := aggregateSyncPullAll(context.Background(), &loader.ProviderLoadResult{Providers: map[string]provider.Provider{}, Statuses: map[string]*loader.ProviderLoadStatus{}}, store)
	if receipt.ProvidersAttempted != 0 || receipt.ProvidersSucceeded != 0 || receipt.ProvidersFailed != 0 || receipt.ProvidersSkipped == 0 {
		t.Fatalf("no-auth fallback should skip providers without failing: %+v", receipt)
	}
	projection := buildSyncPullAllProjection(receipt)
	if projection.Status != "success" {
		t.Fatalf("no-auth fallback projection status = %q, want success", projection.Status)
	}
}

func TestSyncPullAllRespectsEnabledProviders(t *testing.T) {
	oldCfg := cfg
	cfg = config.DefaultConfig()
	cfg.Providers.Google.Enabled = true
	cfg.Providers.Todoist.Enabled = true
	defer func() { cfg = oldCfg }()

	loadResult := &loader.ProviderLoadResult{Providers: map[string]provider.Provider{}, Statuses: map[string]*loader.ProviderLoadStatus{
		"google":  {Name: "google"},
		"todoist": {Name: "todoist"},
		"feishu":  {Name: "feishu"},
	}}
	names := syncPullAllProviderNames(loadResult)
	if strings.Join(names, ",") != "google,todoist" {
		t.Fatalf("enabled provider names = %v, want google,todoist", names)
	}
}

type syncAllMockProvider struct {
	name         string
	lists        []model.TaskList
	tasks        []model.Task
	listErr      error
	remoteWrites int
}

func (p *syncAllMockProvider) Name() string                                               { return p.name }
func (p *syncAllMockProvider) DisplayName() string                                        { return p.name }
func (p *syncAllMockProvider) Authenticate(context.Context, map[string]interface{}) error { return nil }
func (p *syncAllMockProvider) IsAuthenticated() bool                                      { return true }
func (p *syncAllMockProvider) RefreshToken(context.Context) error                         { return nil }
func (p *syncAllMockProvider) ListTaskLists(context.Context) ([]model.TaskList, error) {
	return p.lists, p.listErr
}
func (p *syncAllMockProvider) CreateTaskList(context.Context, string) (*model.TaskList, error) {
	p.remoteWrites++
	return nil, nil
}
func (p *syncAllMockProvider) DeleteTaskList(context.Context, string) error {
	p.remoteWrites++
	return nil
}
func (p *syncAllMockProvider) ListTasks(context.Context, string, provider.ListOptions) ([]model.Task, error) {
	return p.tasks, nil
}
func (p *syncAllMockProvider) GetTask(context.Context, string, string) (*model.Task, error) {
	return nil, nil
}
func (p *syncAllMockProvider) SearchTasks(context.Context, string) ([]model.Task, error) {
	return nil, nil
}
func (p *syncAllMockProvider) CreateTask(context.Context, string, *model.Task) (*model.Task, error) {
	p.remoteWrites++
	return nil, nil
}
func (p *syncAllMockProvider) UpdateTask(context.Context, string, *model.Task) (*model.Task, error) {
	p.remoteWrites++
	return nil, nil
}
func (p *syncAllMockProvider) DeleteTask(context.Context, string, string) error {
	p.remoteWrites++
	return nil
}
func (p *syncAllMockProvider) BatchCreate(context.Context, string, []*model.Task) ([]model.Task, error) {
	p.remoteWrites++
	return nil, nil
}
func (p *syncAllMockProvider) BatchUpdate(context.Context, string, []*model.Task) ([]model.Task, error) {
	p.remoteWrites++
	return nil, nil
}
func (p *syncAllMockProvider) GetChanges(context.Context, time.Time) (*provider.SyncChanges, error) {
	return nil, nil
}
func (p *syncAllMockProvider) Capabilities() provider.Capabilities { return provider.Capabilities{} }
func (p *syncAllMockProvider) GetTokenInfo() *provider.TokenInfo {
	return &provider.TokenInfo{Provider: p.name, HasToken: true, IsValid: true}
}
