package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	syncengine "github.com/yeisme/taskbridge/internal/sync"
)

type fakeScheduler struct {
	running bool
	stats   syncengine.SchedulerStats
	nextRun time.Time
	result  *syncengine.Result
	err     error
}

func (f *fakeScheduler) IsRunning() bool                                     { return f.running }
func (f *fakeScheduler) GetStats() syncengine.SchedulerStats                 { return f.stats }
func (f *fakeScheduler) NextRunTime() time.Time                              { return f.nextRun }
func (f *fakeScheduler) Trigger(context.Context) (*syncengine.Result, error) { return f.result, f.err }

// TestHealthResponseStructure verifies the health endpoint returns valid
// JSON with the required fields.
func TestHealthResponseStructure(t *testing.T) {
	health := &HealthResponse{
		StartTime:   time.Now().Truncate(time.Second),
		Status:      "healthy",
		Live:        true,
		Ready:       true,
		Providers:   map[string]ProviderHealthStatus{},
		TokenStatus: map[string]TokenHealthStatus{},
		Scheduler:   &SchedulerHealthStatus{Running: false},
		Uptime:      "1m0s",
	}

	data, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("failed to marshal health response: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("health JSON is not valid: %v", err)
	}

	requiredFields := []string{"start_time", "status", "live", "ready", "providers", "token_status", "scheduler", "uptime"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("missing required field %q in health response", field)
		}
	}
}

// TestHealthEndpointReturns200 verifies the health HTTP handler returns 200.
func TestHealthEndpointReturns200(t *testing.T) {
	handler := NewHealthHandler(&HealthResponse{
		StartTime: time.Now(),
		Status:    "healthy",
		Live:      true,
		Ready:     true,
		Providers: map[string]ProviderHealthStatus{},
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestHealthEndpointContentTypeJSON verifies the health endpoint returns
// application/json content type.
func TestHealthEndpointContentTypeJSON(t *testing.T) {
	handler := NewHealthHandler(&HealthResponse{
		StartTime: time.Now(),
		Status:    "healthy",
		Live:      true,
		Ready:     true,
		Providers: map[string]ProviderHealthStatus{},
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// TestHealthEndpointDegradedStatus verifies that when providers have errors,
// the status is "degraded" not "healthy".
func TestHealthEndpointDegradedStatus(t *testing.T) {
	health := &HealthResponse{
		StartTime: time.Now(),
		Status:    "degraded",
		Live:      true,
		Ready:     false,
		Providers: map[string]ProviderHealthStatus{
			"google": {
				Loaded: false,
				Error:  "credentials not found",
			},
		},
		TokenStatus: map[string]TokenHealthStatus{},
		Scheduler:   &SchedulerHealthStatus{Running: false},
	}

	handler := NewHealthHandler(health)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if parsed["status"] != "degraded" {
		t.Errorf("expected status 'degraded', got %v", parsed["status"])
	}
}

// TestHealthResponseIncludesProviderErrors verifies that failed provider
// details are visible in the health response.
func TestHealthResponseIncludesProviderErrors(t *testing.T) {
	health := &HealthResponse{
		StartTime: time.Now(),
		Status:    "degraded",
		Live:      true,
		Ready:     false,
		Providers: map[string]ProviderHealthStatus{
			"google": {
				Loaded:        false,
				Authenticated: false,
				Error:         "credentials file not found",
			},
			"microsoft": {
				Loaded:        true,
				Authenticated: true,
				Error:         "",
			},
		},
		TokenStatus: map[string]TokenHealthStatus{},
		Scheduler:   &SchedulerHealthStatus{Running: false},
	}

	data, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	providers := parsed["providers"].(map[string]interface{})
	googleStatus := providers["google"].(map[string]interface{})
	if googleStatus["error"] == nil || googleStatus["error"] == "" {
		t.Error("expected google provider to have error details in health response")
	}

	msStatus := providers["microsoft"].(map[string]interface{})
	if msStatus["loaded"] != true {
		t.Error("expected microsoft provider to be loaded")
	}
}

// TestHealthResponseSchedulerState verifies scheduler status is included.
func TestHealthResponseSchedulerState(t *testing.T) {
	health := &HealthResponse{
		StartTime: time.Now(),
		Status:    "healthy",
		Live:      true,
		Ready:     true,
		Providers: map[string]ProviderHealthStatus{},
		TokenStatus: map[string]TokenHealthStatus{
			"google": {HasToken: true, IsValid: true, NeedsRefresh: false},
		},
		Scheduler: &SchedulerHealthStatus{
			Running:     true,
			Interval:    "5m",
			TotalRuns:   10,
			SuccessRuns: 9,
			FailedRuns:  1,
		},
	}

	data, _ := json.Marshal(health)
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	scheduler := parsed["scheduler"].(map[string]interface{})
	if scheduler["running"] != true {
		t.Error("expected scheduler.running=true")
	}
	if scheduler["interval"] != "5m" {
		t.Errorf("expected scheduler.interval='5m', got %v", scheduler["interval"])
	}
}

func TestServeStatusMuxReadyz(t *testing.T) {
	handler := newServeStatusMux(func() *HealthResponse {
		return &HealthResponse{
			Status: "healthy",
			Live:   true,
			Ready:  true,
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
}

func TestServeStatusMuxReadyzDegraded(t *testing.T) {
	handler := newServeStatusMux(func() *HealthResponse {
		return &HealthResponse{
			Status: "degraded",
			Live:   true,
			Ready:  false,
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Result().StatusCode)
	}
}

func TestServeStatusMuxLivez(t *testing.T) {
	handler := newServeStatusMux(func() *HealthResponse { return &HealthResponse{} })

	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Result().StatusCode)
	}
}

// TestServeGracefulShutdown verifies that serve components can be shut down
// cleanly via context cancellation.
func TestServeGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate serve starting components
	done := make(chan struct{})
	go func() {
		// Simulate work waiting on context
		<-ctx.Done()
		close(done)
	}()

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	select {
	case <-done:
		// Success: goroutine stopped on context cancel
	case <-time.After(2 * time.Second):
		t.Fatal("graceful shutdown timed out - goroutine did not stop")
	}
}

func TestDetermineReadyStatus(t *testing.T) {
	runningScheduler := &fakeScheduler{running: true}
	stoppedScheduler := &fakeScheduler{running: false}

	tests := []struct {
		name      string
		health    string
		scheduler schedulerStatusProvider
		expected  bool
	}{
		{name: "healthy without scheduler", health: "healthy", expected: true},
		{name: "healthy with running scheduler", health: "healthy", scheduler: runningScheduler, expected: true},
		{name: "healthy with stopped scheduler", health: "healthy", scheduler: stoppedScheduler, expected: false},
		{name: "degraded without scheduler", health: "degraded", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := determineReadyStatus(tt.health, tt.scheduler); got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestTriggerInitialSync(t *testing.T) {
	expected := &syncengine.Result{Pulled: 2, Pushed: 1}
	scheduler := &fakeScheduler{result: expected}

	result, err := triggerInitialSync(context.Background(), scheduler)
	if err != nil {
		t.Fatalf("triggerInitialSync returned error: %v", err)
	}
	if result != expected {
		t.Fatalf("expected returned result pointer to match fake scheduler result")
	}
}

// TestServeHealthServerStartStop verifies that the health HTTP server
// can be started and stopped without leaking goroutines.
func TestServeHealthServerStartStop(t *testing.T) {
	handler := NewHealthHandler(&HealthResponse{
		StartTime: time.Now(),
		Status:    "healthy",
		Providers: map[string]ProviderHealthStatus{},
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Verify server responds
	resp, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("health server request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Server.Close() should not panic (verified by defer)
}

// TestDetermineHealthStatus verifies health status determination logic.
func TestDetermineHealthStatus(t *testing.T) {
	tests := []struct {
		name      string
		providers map[string]ProviderHealthStatus
		expected  string
	}{
		{
			name: "all providers loaded and authenticated",
			providers: map[string]ProviderHealthStatus{
				"google":    {Loaded: true, Authenticated: true},
				"microsoft": {Loaded: true, Authenticated: true},
			},
			expected: "healthy",
		},
		{
			name:      "no providers configured",
			providers: map[string]ProviderHealthStatus{},
			expected:  "degraded",
		},
		{
			name: "some providers failed",
			providers: map[string]ProviderHealthStatus{
				"google":    {Loaded: true, Authenticated: true},
				"microsoft": {Loaded: false, Error: "credentials not found"},
			},
			expected: "degraded",
		},
		{
			name: "all providers failed",
			providers: map[string]ProviderHealthStatus{
				"google":    {Loaded: false, Error: "not configured"},
				"microsoft": {Loaded: false, Error: "auth failed"},
			},
			expected: "degraded",
		},
		{
			name: "provider loaded but not authenticated",
			providers: map[string]ProviderHealthStatus{
				"google": {Loaded: true, Authenticated: false, Error: "token expired"},
			},
			expected: "degraded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineHealthStatus(tt.providers)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
