package sync

import (
	"context"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/model"
	"github.com/yeisme/taskbridge/internal/provider"
)

// TestSchedulerConfigIntervalField verifies that SchedulerConfig accepts
// an Interval duration alongside CronExpression.
func TestSchedulerConfigIntervalField(t *testing.T) {
	cfg := SchedulerConfig{
		Interval:       90 * time.Second,
		CronExpression: "",
		Direction:      DirectionPull,
		MaxRetries:     3,
	}
	if cfg.Interval != 90*time.Second {
		t.Errorf("expected Interval=90s, got %v", cfg.Interval)
	}
}

// TestSchedulerIntervalStart verifies that the scheduler can start with an
// Interval-based configuration (no cron expression).
func TestSchedulerIntervalStart(t *testing.T) {
	mockProvider := &MockProvider{
		name:          "mock",
		authenticated: true,
		taskLists:     []model.TaskList{{ID: "list1", Name: "List 1", Source: "mock"}},
		tasks:         map[string][]model.Task{},
	}
	providers := map[string]provider.Provider{"mock": mockProvider}
	store := NewMockStorage()

	cfg := SchedulerConfig{
		Interval:   500 * time.Millisecond,
		Direction:  DirectionPull,
		MaxRetries: 1,
	}
	scheduler := NewScheduler(cfg, providers, store)

	ctx := context.Background()
	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("scheduler Start with Interval failed: %v", err)
	}
	if !scheduler.IsRunning() {
		t.Error("scheduler should be running after Start")
	}

	// Wait enough time for at least one tick
	time.Sleep(700 * time.Millisecond)

	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("scheduler Stop failed: %v", err)
	}
	if scheduler.IsRunning() {
		t.Error("scheduler should not be running after Stop")
	}
}

// TestSchedulerIntervalTriggersSync verifies that an interval-based
// scheduler actually triggers sync runs.
func TestSchedulerIntervalTriggersSync(t *testing.T) {
	mockProvider := &MockProvider{
		name:          "mock",
		authenticated: true,
		taskLists:     []model.TaskList{{ID: "list1", Name: "List 1", Source: "mock"}},
		tasks: map[string][]model.Task{
			"list1": {{ID: "task1", Title: "Task 1", Status: "todo", Source: "mock"}},
		},
	}
	providers := map[string]provider.Provider{"mock": mockProvider}
	store := NewMockStorage()

	cfg := SchedulerConfig{
		Interval:      200 * time.Millisecond,
		Direction:     DirectionPull,
		MaxRetries:    1,
		RetryInterval: 10 * time.Millisecond,
	}
	scheduler := NewScheduler(cfg, providers, store)

	ctx := context.Background()
	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("scheduler Start failed: %v", err)
	}

	// Wait for multiple intervals
	time.Sleep(700 * time.Millisecond)

	scheduler.Stop()

	stats := scheduler.GetStats()
	if stats.TotalRuns == 0 {
		t.Error("expected at least one sync run with interval scheduling")
	}
	if stats.LastRunStatus != "success" {
		t.Errorf("expected last run status 'success', got %q", stats.LastRunStatus)
	}
}

// TestSchedulerCronExpressionStillWorks verifies that the existing
// cron-based scheduling path is not broken.
func TestSchedulerCronExpressionStillWorks(t *testing.T) {
	mockProvider := &MockProvider{
		name:          "mock",
		authenticated: true,
		taskLists:     []model.TaskList{{ID: "list1", Name: "List 1", Source: "mock"}},
		tasks:         map[string][]model.Task{},
	}
	providers := map[string]provider.Provider{"mock": mockProvider}
	store := NewMockStorage()

	cfg := SchedulerConfig{
		CronExpression: "*/1 * * * * *",
		Direction:      DirectionPull,
		MaxRetries:     1,
	}
	scheduler := NewScheduler(cfg, providers, store)

	ctx := context.Background()
	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("scheduler Start with CronExpression failed: %v", err)
	}
	if !scheduler.IsRunning() {
		t.Error("scheduler should be running after Start")
	}

	// Wait for at least one cron tick
	time.Sleep(1200 * time.Millisecond)

	scheduler.Stop()

	stats := scheduler.GetStats()
	if stats.TotalRuns == 0 {
		t.Error("expected at least one sync run with cron scheduling")
	}
}

// TestSchedulerIntervalStats verifies that stats are properly tracked
// for interval-based scheduling.
func TestSchedulerIntervalStats(t *testing.T) {
	mockProvider := &MockProvider{
		name:          "mock",
		authenticated: true,
		taskLists:     []model.TaskList{{ID: "list1", Name: "List 1", Source: "mock"}},
		tasks: map[string][]model.Task{
			"list1": {{ID: "task1", Title: "Task 1", Status: "todo", Source: "mock"}},
		},
	}
	providers := map[string]provider.Provider{"mock": mockProvider}
	store := NewMockStorage()

	cfg := SchedulerConfig{
		Interval:      200 * time.Millisecond,
		Direction:     DirectionPull,
		MaxRetries:    1,
		RetryInterval: 10 * time.Millisecond,
	}
	scheduler := NewScheduler(cfg, providers, store)

	ctx := context.Background()
	_ = scheduler.Start(ctx)
	time.Sleep(700 * time.Millisecond)
	scheduler.Stop()

	stats := scheduler.GetStats()
	if stats.TotalRuns == 0 {
		t.Fatal("expected at least one run")
	}
	if !stats.LastRunTime.IsZero() == false && stats.LastRunStatus == "" {
		t.Error("expected LastRunStatus to be set after runs")
	}
	if stats.AverageRuntime == 0 {
		t.Error("expected non-zero AverageRuntime after runs")
	}
}

// TestSchedulerStopIdempotent verifies that Stop can be called multiple times
// safely with interval-based scheduling.
func TestSchedulerStopIdempotent(t *testing.T) {
	mockProvider := &MockProvider{name: "mock", authenticated: true}
	providers := map[string]provider.Provider{"mock": mockProvider}
	store := NewMockStorage()

	cfg := SchedulerConfig{
		Interval:   1 * time.Minute,
		Direction:  DirectionPull,
		MaxRetries: 1,
	}
	scheduler := NewScheduler(cfg, providers, store)

	ctx := context.Background()
	_ = scheduler.Start(ctx)

	// Stop multiple times should not panic or error
	for i := 0; i < 3; i++ {
		if err := scheduler.Stop(); err != nil {
			t.Errorf("Stop call %d failed: %v", i+1, err)
		}
	}
}

// TestSchedulerTriggerWithInterval verifies that manual Trigger works
// when scheduler is configured with Interval.
func TestSchedulerTriggerWithInterval(t *testing.T) {
	mockProvider := &MockProvider{
		name:          "mock",
		authenticated: true,
		taskLists:     []model.TaskList{{ID: "list1", Name: "List 1", Source: "mock"}},
		tasks:         map[string][]model.Task{},
	}
	providers := map[string]provider.Provider{"mock": mockProvider}
	store := NewMockStorage()

	cfg := SchedulerConfig{
		Interval:   1 * time.Hour, // long interval, won't auto-trigger
		Direction:  DirectionPull,
		MaxRetries: 1,
	}
	scheduler := NewScheduler(cfg, providers, store)

	ctx := context.Background()
	_ = scheduler.Start(ctx)
	defer scheduler.Stop()

	result, err := scheduler.Trigger(ctx)
	if err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}
	if result == nil {
		t.Fatal("Trigger returned nil result")
	}

	stats := scheduler.GetStats()
	// The manual trigger should update stats
	if stats.TotalRuns == 0 {
		t.Error("expected TotalRuns >= 1 after Trigger")
	}
}
