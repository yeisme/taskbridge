package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/auth"
	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/loader"
	syncengine "github.com/yeisme/taskbridge/internal/sync"
	"github.com/yeisme/taskbridge/pkg/ui"
)

// serveCmd background service command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start background service",
	Long: `Start the TaskBridge background service.

Functions:
  - Automatically refresh tokens before expiration
  - Run scheduled synchronization tasks when enabled
  - Serve an optional health endpoint

Examples:
  taskbridge serve
  taskbridge serve --check-interval 2m`,
	RunE: runServe,
}

var (
	// Service configuration.
	serveCheckInterval     string
	serveEnableSync        bool
	serveSyncInterval      string
	serveSyncOnStart       bool
	serveSyncConfirm       bool
	serveEnableHealth      bool
	serveHealthPort        int
	serveRefreshBuffer     string
	serveEnableAutoRefresh bool
)

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().BoolVar(&serveEnableAutoRefresh, "enable-auto-refresh", true, "Enable automatic token refresh")
	serveCmd.Flags().StringVar(&serveCheckInterval, "check-interval", "1m", "Token check interval")
	serveCmd.Flags().StringVar(&serveRefreshBuffer, "refresh-buffer", "5m", "Refresh buffer before token expiration")

	// Sync configuration.
	serveCmd.Flags().BoolVar(&serveEnableSync, "enable-sync", false, "Enable scheduled synchronization")
	serveCmd.Flags().StringVar(&serveSyncInterval, "sync-interval", "5m", "Synchronization interval")
	serveCmd.Flags().BoolVar(&serveSyncOnStart, "sync-on-start", true, "Run one synchronization immediately after service startup")
	serveCmd.Flags().BoolVar(&serveSyncConfirm, "sync-confirm", false, "Confirm remote overwrites/deletes during scheduled sync (required for remote writes)")

	// Health check configuration.
	serveCmd.Flags().BoolVar(&serveEnableHealth, "enable-health", false, "Enable health endpoint")
	serveCmd.Flags().IntVar(&serveHealthPort, "health-port", 8081, "Health endpoint port")
}

// runServe executes background services
func runServe(cmd *cobra.Command, args []string) error {
	if resolveOutputFormat("text") != "text" || IsQuietMode() {
		return usageError("serve does not support machine output; omit --json, --agent, --events, --explain, and --quiet")
	}

	printServeSummary("serve.start", "TaskBridge background service is starting.", nil)

	//Parse configuration
	checkInterval, err := time.ParseDuration(serveCheckInterval)
	if err != nil {
		return commandError("Invalid check interval", err)
	}

	refreshBuffer, err := time.ParseDuration(serveRefreshBuffer)
	if err != nil {
		return commandError("Invalid refresh buffer", err)
	}

	// Create a token manager.
	tokenManager := auth.NewTokenManager(auth.TokenManagerConfig{
		CheckInterval: checkInterval,
		RefreshBuffer: refreshBuffer,
		MaxRetries:    3,
		RetryInterval: 30 * time.Second,
	})

	loadResult := loadProvidersWithStatus("")
	registerProviders(tokenManager, loadResult) //Set refresh callback
	tokenManager.SetOnRefreshCallback(func(provider string, err error) {
		if err != nil {
			fmt.Printf("Token refresh failed for %s: %v\n", provider, err)
		} else {
			fmt.Printf("Token refreshed for %s\n", provider)
		}
	})

	// Create context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var scheduler *syncengine.Scheduler
	if serveEnableSync && len(loadResult.Providers) > 0 {
		syncInterval, err := time.ParseDuration(serveSyncInterval)
		if err != nil {
			return commandError("Invalid synchronization interval", err)
		}

		store, cleanup, err := getStore()
		if err != nil {
			return commandError("Failed to initialize sync storage", err)
		}
		defer cleanup()

		scheduler = syncengine.NewScheduler(syncengine.SchedulerConfig{
			Interval:        syncInterval,
			Direction:       syncengine.DirectionBidirectional,
			Incremental:     true,
			MaxRetries:      3,
			RetryInterval:   30 * time.Second,
			ConflictResolve: "newer",
			Confirm:         serveSyncConfirm,
		}, loadResult.Providers, store)
	}

	// Start automatic token refresh.
	if serveEnableAutoRefresh {
		if err := tokenManager.Start(ctx); err != nil {
			return commandError("Failed to start token manager", err)
		}
		fmt.Printf("Token auto-refresh enabled (check interval: %s, refresh buffer: %s)\n", checkInterval, refreshBuffer)
	}

	// Show current token status.
	printTokenStatus(tokenManager)

	if scheduler != nil {
		if err := scheduler.Start(ctx); err != nil {
			return commandError("Failed to start scheduled synchronization", err)
		}
		fmt.Printf("Scheduled synchronization enabled (interval: %s)\n", serveSyncInterval)
		if serveSyncOnStart {
			if result, err := triggerInitialSync(ctx, scheduler); err != nil {
				fmt.Printf("Initial sync failed at startup: %v\n", err)
			} else {
				fmt.Printf("Initial sync completed at startup (pulled=%d, pushed=%d, errors=%d)\n", result.Pulled, result.Pushed, len(result.Errors))
			}
		}
	} else if serveEnableSync {
		fmt.Println("No authenticated provider is available; scheduled synchronization was not started.")
	}

	// Start health checks when enabled.
	if serveEnableHealth {
		go startHealthCheck(ctx, serveHealthPort, func() *HealthResponse {
			return buildHealthResponse(tokenManager, loadResult, scheduler, serveSyncInterval)
		})
	}

	fmt.Println("\nService started. Press Ctrl+C to stop it.")
	fmt.Println("────────────────────────────────────────")

	// Wait for an interrupt signal.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\nStopping service...")

	// Stop the token manager.
	tokenManager.Stop()
	if scheduler != nil {
		_ = scheduler.Stop()
	}

	printServeSummary("serve.stop", "TaskBridge background service stopped.", nil)
	return nil
}

// registerProviders registers successfully loaded providers and prints provider status.
func registerProviders(tm *auth.TokenManager, result *loader.ProviderLoadResult) {
	for name, status := range result.Statuses {
		if providerImpl, ok := result.Providers[name]; ok {
			tm.RegisterProvider(providerImpl)
			fmt.Printf("Registered provider: %s\n", providerImpl.DisplayName())
			continue
		}
		if status != nil && status.Error != "" {
			fmt.Printf("Provider %s is not ready: %s\n", name, status.Error)
		} else {
			fmt.Printf("Provider %s is not ready\n", name)
		}
	}
}

// printTokenStatus prints token status.
func printTokenStatus(tm *auth.TokenManager) {
	fmt.Println("\nToken status")
	table := ui.NewSimpleTable(
		ui.Column{Header: "Provider", AlignLeft: true},
		ui.Column{Header: "Status", AlignLeft: true},
		ui.Column{Header: "Expires", AlignLeft: true},
		ui.Column{Header: "Remaining", AlignLeft: true},
	)

	status := tm.GetStatus()
	names := make([]string, 0, len(status))
	for name := range status {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		info := status[name]
		statusText := "not authenticated"
		if info.HasToken {
			if info.IsValid {
				if info.NeedsRefresh {
					statusText = "needs refresh"
				} else {
					statusText = "valid"
				}
			} else {
				statusText = "expired"
			}
		}

		expiresAt := "-"
		timeLeft := "-"
		if info.HasToken && !info.ExpiresAt.IsZero() {
			expiresAt = info.ExpiresAt.Format("2006-01-02 15:04:05")
			timeLeft = info.TimeUntilExpiry
		}

		table.AddRow(name, statusText, expiresAt, timeLeft)
	}
	fmt.Print(table.Render())
}

type ProviderHealthStatus struct {
	Loaded        bool   `json:"loaded"`
	Authenticated bool   `json:"authenticated"`
	Error         string `json:"error,omitempty"`
}

type TokenHealthStatus struct {
	HasToken        bool      `json:"has_token"`
	IsValid         bool      `json:"is_valid"`
	NeedsRefresh    bool      `json:"needs_refresh"`
	Refreshable     bool      `json:"refreshable"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	TimeUntilExpiry string    `json:"time_until_expiry,omitempty"`
}

type SchedulerHealthStatus struct {
	Running       bool      `json:"running"`
	Interval      string    `json:"interval,omitempty"`
	LastRunTime   time.Time `json:"last_run_time,omitempty"`
	NextRunTime   time.Time `json:"next_run_time,omitempty"`
	LastRunStatus string    `json:"last_run_status,omitempty"`
	TotalRuns     int       `json:"total_runs"`
	SuccessRuns   int       `json:"success_runs"`
	FailedRuns    int       `json:"failed_runs"`
}

type HealthResponse struct {
	StartTime           time.Time                       `json:"start_time"`
	Status              string                          `json:"status"`
	Live                bool                            `json:"live"`
	Ready               bool                            `json:"ready"`
	TokenManagerRunning bool                            `json:"token_manager_running"`
	Providers           map[string]ProviderHealthStatus `json:"providers"`
	TokenStatus         map[string]TokenHealthStatus    `json:"token_status"`
	Scheduler           *SchedulerHealthStatus          `json:"scheduler,omitempty"`
	Uptime              string                          `json:"uptime"`
}

func DetermineHealthStatus(providers map[string]ProviderHealthStatus) string {
	if len(providers) == 0 {
		return "degraded"
	}
	for _, status := range providers {
		if !status.Loaded || !status.Authenticated || status.Error != "" {
			return "degraded"
		}
	}
	return "healthy"
}

type schedulerStatusProvider interface {
	IsRunning() bool
	GetStats() syncengine.SchedulerStats
	NextRunTime() time.Time
}

type schedulerTrigger interface {
	Trigger(context.Context) (*syncengine.Result, error)
}

func NewHealthHandler(health *HealthResponse) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, health)
	})
}

func newDynamicHealthHandler(snapshot func() *HealthResponse) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, snapshot())
	})
}

func newServeStatusMux(snapshot func() *HealthResponse) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/health", newDynamicHealthHandler(snapshot))
	mux.Handle("/healthz", newDynamicHealthHandler(snapshot))
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "alive",
			"live":   true,
		})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		health := snapshot()
		statusCode := http.StatusOK
		state := "ready"
		if !health.Ready {
			statusCode = http.StatusServiceUnavailable
			state = "not_ready"
		}
		writeJSON(w, statusCode, map[string]interface{}{
			"status": state,
			"ready":  health.Ready,
		})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, snapshot())
	})

	return mux
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func buildHealthResponse(
	tm *auth.TokenManager,
	loadResult *loader.ProviderLoadResult,
	scheduler schedulerStatusProvider,
	schedulerInterval string,
) *HealthResponse {
	providers := make(map[string]ProviderHealthStatus, len(loadResult.Statuses))
	for name, status := range loadResult.Statuses {
		if status == nil {
			continue
		}
		providers[name] = ProviderHealthStatus{
			Loaded:        status.Loaded,
			Authenticated: status.Authenticated,
			Error:         status.Error,
		}
	}

	tokenStatus := make(map[string]TokenHealthStatus)
	for name, info := range tm.GetStatus() {
		if info == nil {
			continue
		}
		tokenStatus[name] = TokenHealthStatus{
			HasToken:        info.HasToken,
			IsValid:         info.IsValid,
			NeedsRefresh:    info.NeedsRefresh,
			Refreshable:     info.Refreshable,
			ExpiresAt:       info.ExpiresAt,
			TimeUntilExpiry: info.TimeUntilExpiry,
		}
	}

	health := &HealthResponse{
		StartTime:           startTime,
		Live:                true,
		TokenManagerRunning: tm.IsRunning(),
		Providers:           providers,
		TokenStatus:         tokenStatus,
		Uptime:              time.Since(startTime).Truncate(time.Second).String(),
	}

	health.Status = DetermineHealthStatus(providers)
	health.Ready = determineReadyStatus(health.Status, scheduler)
	if scheduler != nil {
		stats := scheduler.GetStats()
		health.Scheduler = &SchedulerHealthStatus{
			Running:       scheduler.IsRunning(),
			Interval:      schedulerInterval,
			LastRunTime:   stats.LastRunTime,
			NextRunTime:   scheduler.NextRunTime(),
			LastRunStatus: stats.LastRunStatus,
			TotalRuns:     stats.TotalRuns,
			SuccessRuns:   stats.SuccessRuns,
			FailedRuns:    stats.FailedRuns,
		}
	}

	return health
}

var startTime = time.Now()

func determineReadyStatus(healthStatus string, scheduler schedulerStatusProvider) bool {
	if healthStatus != "healthy" {
		return false
	}
	if scheduler == nil {
		return true
	}
	return scheduler.IsRunning()
}

func triggerInitialSync(ctx context.Context, scheduler schedulerTrigger) (*syncengine.Result, error) {
	return scheduler.Trigger(ctx)
}

// startHealthCheck starts the health endpoint.
func startHealthCheck(ctx context.Context, port int, snapshot func() *HealthResponse) {
	fmt.Printf("Health endpoint: http://localhost:%d/health\n", port)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: newServeStatusMux(snapshot),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Health endpoint failed: %v\n", err)
	}
}

func printServeSummary(command, summary string, facts map[string]any) {
	p := clioutput.New(command)
	p.Summary = summary
	for key, value := range facts {
		p.Facts[key] = value
	}
	fmt.Print(clioutput.RenderSummary(p))
}
