package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/sync"
	"github.com/yeisme/taskbridge/internal/syncaudit"
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
