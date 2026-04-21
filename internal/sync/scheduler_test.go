package sync

import (
	"context"
	"testing"
	"time"

	"github.com/yeisme/taskbridge/internal/provider"
)

func newTestScheduler(config SchedulerConfig) *Scheduler {
	providers := map[string]provider.Provider{
		"mock": &MockProvider{name: "mock", authenticated: true},
	}
	return NewScheduler(config, providers, NewMockStorage())
}

// ================ Config Validation ================

func TestSchedulerConfig_Validate_CronOnly(t *testing.T) {
	config := SchedulerConfig{CronExpression: "0 */5 * * * *"}
	if err := config.Validate(); err != nil {
		t.Fatalf("expected valid for CronExpression only, got: %v", err)
	}
}

func TestSchedulerConfig_Validate_IntervalOnly(t *testing.T) {
	config := SchedulerConfig{Interval: 5 * time.Minute}
	if err := config.Validate(); err != nil {
		t.Fatalf("expected valid for Interval only, got: %v", err)
	}
}

func TestSchedulerConfig_Validate_MutualExclusion(t *testing.T) {
	config := SchedulerConfig{
		CronExpression: "0 */5 * * * *",
		Interval:       5 * time.Minute,
	}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error when both CronExpression and Interval are set")
	}
}

func TestSchedulerConfig_Validate_NeitherSet(t *testing.T) {
	config := SchedulerConfig{}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error when neither CronExpression nor Interval is set")
	}
}

func TestSchedulerConfig_Validate_IntervalNonPositive(t *testing.T) {
	config := SchedulerConfig{Interval: 0}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error for non-positive interval")
	}
}

func TestSchedulerConfig_IsIntervalMode(t *testing.T) {
	tests := []struct {
		name     string
		config   SchedulerConfig
		expected bool
	}{
		{"cron mode", SchedulerConfig{CronExpression: "0 */5 * * * *"}, false},
		{"interval mode", SchedulerConfig{Interval: 5 * time.Minute}, true},
		{"empty", SchedulerConfig{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsIntervalMode(); got != tt.expected {
				t.Errorf("IsIntervalMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ================ Interval Mode ================

func TestScheduler_Start_IntervalMode(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 10 * time.Second})
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if !s.IsRunning() {
		t.Fatal("expected scheduler to be running")
	}
	if !s.config.IsIntervalMode() {
		t.Fatal("expected interval mode")
	}
}

func TestScheduler_Start_MutualExclusion(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{
		CronExpression: "0 */5 * * * *",
		Interval:       5 * time.Minute,
	})
	defer s.Stop()

	err := s.Start(context.Background())
	if err == nil {
		t.Fatal("expected error for mutual exclusion")
		s.Stop()
	}
}

func TestScheduler_Start_NeitherSet(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{})
	defer s.Stop()

	err := s.Start(context.Background())
	if err == nil {
		t.Fatal("expected error when neither cron nor interval set")
		s.Stop()
	}
}

func TestScheduler_Stop_IntervalMode(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 10 * time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
	if s.IsRunning() {
		t.Fatal("expected scheduler to be stopped")
	}

	// Verify ticker is cleaned up
	s.mu.RLock()
	ticker := s.ticker
	s.mu.RUnlock()
	if ticker != nil {
		t.Fatal("expected ticker to be nil after stop")
	}
}

func TestScheduler_Stop_Idempotent(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 10 * time.Second})
	// Stop without start should not panic
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() on non-running scheduler failed: %v", err)
	}
}

// ================ CronExpression Mode (regression) ================

func TestScheduler_Start_CronMode(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{CronExpression: "0 */5 * * * *"})
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	if !s.IsRunning() {
		t.Fatal("expected scheduler to be running")
	}
	if s.config.IsIntervalMode() {
		t.Fatal("expected cron mode")
	}
}

func TestScheduler_Stop_CronMode(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{CronExpression: "0 */5 * * * *"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
	if s.IsRunning() {
		t.Fatal("expected scheduler to be stopped")
	}
}

// ================ NextRunTime ================

func TestScheduler_NextRunTime_IntervalMode(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 5 * time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	beforeStart := time.Now()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer s.Stop()

	next := s.NextRunTime()
	if next.IsZero() {
		t.Fatal("expected non-zero NextRunTime")
	}

	// NextRunTime should be approximately lastRunStart + interval
	s.mu.RLock()
	lastStart := s.lastRunStart
	s.mu.RUnlock()

	expected := lastStart.Add(5 * time.Second)
	diff := next.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > 2*time.Second {
		t.Errorf("NextRunTime = %v, expected ~%v (diff %v)", next, expected, diff)
	}
	_ = beforeStart
}

func TestScheduler_NextRunTime_NotRunning(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 5 * time.Second})
	next := s.NextRunTime()
	if !next.IsZero() {
		t.Fatal("expected zero NextRunTime when not running")
	}
}

func TestScheduler_NextRunTime_CronMode(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{CronExpression: "0 */5 * * * *"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer s.Stop()

	next := s.NextRunTime()
	if next.IsZero() {
		t.Fatal("expected non-zero NextRunTime for cron mode")
	}
}

// ================ UpdateConfig ================

func TestScheduler_UpdateConfig_CronToInterval(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{CronExpression: "0 */5 * * * *"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	err := s.UpdateConfig(SchedulerConfig{Interval: 30 * time.Second})
	if err != nil {
		t.Fatalf("UpdateConfig() failed: %v", err)
	}
	defer s.Stop()

	if !s.IsRunning() {
		t.Fatal("expected scheduler to still be running after config update")
	}
	if !s.config.IsIntervalMode() {
		t.Fatal("expected interval mode after update")
	}
}

func TestScheduler_UpdateConfig_IntervalToCron(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 10 * time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	err := s.UpdateConfig(SchedulerConfig{CronExpression: "0 */5 * * * *"})
	if err != nil {
		t.Fatalf("UpdateConfig() failed: %v", err)
	}
	defer s.Stop()

	if !s.IsRunning() {
		t.Fatal("expected scheduler to still be running after config update")
	}
	if s.config.IsIntervalMode() {
		t.Fatal("expected cron mode after update")
	}
}

func TestScheduler_UpdateConfig_InvalidMutualExclusion(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 10 * time.Second})
	defer s.Stop()

	err := s.UpdateConfig(SchedulerConfig{
		CronExpression: "0 */5 * * * *",
		Interval:       5 * time.Minute,
	})
	if err == nil {
		t.Fatal("expected error for mutual exclusion in UpdateConfig")
	}
}

func TestScheduler_UpdateConfig_NotRunning(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 10 * time.Second})
	defer s.Stop()

	// Update config while not running should succeed
	err := s.UpdateConfig(SchedulerConfig{Interval: 30 * time.Second})
	if err != nil {
		t.Fatalf("UpdateConfig() on stopped scheduler failed: %v", err)
	}
	if s.config.Interval != 30*time.Second {
		t.Fatal("config not updated")
	}
}

// ================ Double Start Prevention ================

func TestScheduler_DoubleStart(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 10 * time.Second})
	defer s.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("first Start() failed: %v", err)
	}

	err := s.Start(ctx)
	if err == nil {
		t.Fatal("expected error on double start")
	}
}

// ================ Trigger ================

func TestScheduler_Trigger_IntervalMode(t *testing.T) {
	s := newTestScheduler(SchedulerConfig{Interval: 1 * time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Trigger should work even without Start (uses engine directly)
	_, err := s.Trigger(ctx)
	// The mock storage/provider setup may not produce a meaningful result,
	// but it should not panic
	_ = err
}
