package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/auth"
	"github.com/yeisme/taskbridge/internal/loader"
	syncengine "github.com/yeisme/taskbridge/internal/sync"
)

// serveCmd 后台服务命令
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动后台服务",
	Long: `启动 TaskBridge 后台服务。

功能:
  - Token 自动刷新（防止过期）
  - 定时同步任务
  - 健康检查（可选）

示例:
  taskbridge serve                    # 启动服务（默认配置）
  taskbridge serve --check-interval 2m # 设置检查间隔为 2 分钟`,
	RunE: runServe,
}

var (
	// 服务配置
	serveCheckInterval     string
	serveEnableSync        bool
	serveSyncInterval      string
	serveSyncOnStart       bool
	serveEnableHealth      bool
	serveHealthPort        int
	serveRefreshBuffer     string
	serveEnableAutoRefresh bool
)

func init() {
	rootCmd.AddCommand(serveCmd)

	// Token 刷新配置
	serveCmd.Flags().BoolVar(&serveEnableAutoRefresh, "enable-auto-refresh", true, "启用 Token 自动刷新")
	serveCmd.Flags().StringVar(&serveCheckInterval, "check-interval", "1m", "Token 检查间隔")
	serveCmd.Flags().StringVar(&serveRefreshBuffer, "refresh-buffer", "5m", "刷新提前量（Token 过期前多久刷新）")

	// 同步配置
	serveCmd.Flags().BoolVar(&serveEnableSync, "enable-sync", false, "启用定时同步")
	serveCmd.Flags().StringVar(&serveSyncInterval, "sync-interval", "5m", "同步间隔")
	serveCmd.Flags().BoolVar(&serveSyncOnStart, "sync-on-start", true, "启动服务后立即执行一次同步")

	// 健康检查配置
	serveCmd.Flags().BoolVar(&serveEnableHealth, "enable-health", false, "启用健康检查端点")
	serveCmd.Flags().IntVar(&serveHealthPort, "health-port", 8081, "健康检查端口")
}

// runServe 执行后台服务
func runServe(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 TaskBridge 后台服务启动中...")

	// 解析配置
	checkInterval, err := time.ParseDuration(serveCheckInterval)
	if err != nil {
		return commandError("无效的检查间隔", err)
	}

	refreshBuffer, err := time.ParseDuration(serveRefreshBuffer)
	if err != nil {
		return commandError("无效的刷新提前量", err)
	}

	// 创建 Token 管理器
	tokenManager := auth.NewTokenManager(auth.TokenManagerConfig{
		CheckInterval: checkInterval,
		RefreshBuffer: refreshBuffer,
		MaxRetries:    3,
		RetryInterval: 30 * time.Second,
	})

	loadResult := loadProvidersWithStatus("")
	registerProviders(tokenManager, loadResult)

	// 设置刷新回调
	tokenManager.SetOnRefreshCallback(func(provider string, err error) {
		if err != nil {
			fmt.Printf("❌ %s Token 刷新失败: %v\n", provider, err)
		} else {
			fmt.Printf("✅ %s Token 刷新成功\n", provider)
		}
	})

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var scheduler *syncengine.Scheduler
	if serveEnableSync && len(loadResult.Providers) > 0 {
		syncInterval, err := time.ParseDuration(serveSyncInterval)
		if err != nil {
			return commandError("无效的同步间隔", err)
		}

		store, cleanup, err := getStore()
		defer cleanup()
		if err != nil {
			return commandError("初始化同步存储失败", err)
		}

		scheduler = syncengine.NewScheduler(syncengine.SchedulerConfig{
			Interval:        syncInterval,
			Direction:       syncengine.DirectionBidirectional,
			Incremental:     true,
			MaxRetries:      3,
			RetryInterval:   30 * time.Second,
			ConflictResolve: "newer",
		}, loadResult.Providers, store)
	}

	// 启动 Token 自动刷新
	if serveEnableAutoRefresh {
		if err := tokenManager.Start(ctx); err != nil {
			return commandError("启动 Token 管理器失败", err)
		}
		fmt.Printf("✅ Token 自动刷新已启用 (检查间隔: %s, 刷新提前量: %s)\n", checkInterval, refreshBuffer)
	}

	// 显示当前 Token 状态
	printTokenStatus(tokenManager)

	if scheduler != nil {
		if err := scheduler.Start(ctx); err != nil {
			return commandError("启动定时同步失败", err)
		}
		fmt.Printf("🔄 定时同步已启用 (间隔: %s)\n", serveSyncInterval)
		if serveSyncOnStart {
			if result, err := triggerInitialSync(ctx, scheduler); err != nil {
				fmt.Printf("⚠️ 启动时首次同步失败: %v\n", err)
			} else {
				fmt.Printf("✅ 启动时首次同步完成 (pulled=%d, pushed=%d, errors=%d)\n", result.Pulled, result.Pushed, len(result.Errors))
			}
		}
	} else if serveEnableSync {
		fmt.Println("⚠️ 未找到可用已认证 Provider，定时同步未启动")
	}

	// 启动健康检查（如果启用）
	if serveEnableHealth {
		go startHealthCheck(ctx, serveHealthPort, func() *HealthResponse {
			return buildHealthResponse(tokenManager, loadResult, scheduler, serveSyncInterval)
		})
	}

	fmt.Println("\n📋 服务已启动，按 Ctrl+C 停止")
	fmt.Println("─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\n\n🛑 正在停止服务...")

	// 停止 Token 管理器
	tokenManager.Stop()
	if scheduler != nil {
		_ = scheduler.Stop()
	}

	fmt.Println("👋 服务已停止")
	return nil
}

// registerProviders 注册所有已成功加载的 Provider，并输出所有 Provider 状态。
func registerProviders(tm *auth.TokenManager, result *loader.ProviderLoadResult) {
	for name, status := range result.Statuses {
		if providerImpl, ok := result.Providers[name]; ok {
			tm.RegisterProvider(providerImpl)
			fmt.Printf("✅ 已注册 Provider: %s\n", providerImpl.DisplayName())
			continue
		}
		if status != nil && status.Error != "" {
			fmt.Printf("⚠️ %s Provider 未就绪: %s\n", name, status.Error)
		} else {
			fmt.Printf("⚠️ %s Provider 未就绪\n", name)
		}
	}
}

// printTokenStatus 打印 Token 状态
func printTokenStatus(tm *auth.TokenManager) {
	fmt.Println("\n📊 Token 状态:")
	fmt.Println("┌────────────┬─────────┬─────────────────────┬──────────────┐")
	fmt.Println("│ Provider   │ 状态    │ 过期时间            │ 剩余时间     │")
	fmt.Println("├────────────┼─────────┼─────────────────────┼──────────────┤")

	status := tm.GetStatus()
	for name, info := range status {
		statusIcon := "❌ 未认证"
		if info.HasToken {
			if info.IsValid {
				if info.NeedsRefresh {
					statusIcon = "⚠️ 需刷新"
				} else {
					statusIcon = "✅ 有效"
				}
			} else {
				statusIcon = "❌ 已过期"
			}
		}

		expiresAt := "-"
		timeLeft := "-"
		if info.HasToken && !info.ExpiresAt.IsZero() {
			expiresAt = info.ExpiresAt.Format("2006-01-02 15:04:05")
			timeLeft = info.TimeUntilExpiry
		}

		fmt.Printf("│ %-10s │ %-7s │ %-19s │ %-12s │\n", name, statusIcon, expiresAt, timeLeft)
	}
	fmt.Println("└────────────┴─────────┴─────────────────────┴──────────────┘")
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

// startHealthCheck 启动健康检查
func startHealthCheck(ctx context.Context, port int, snapshot func() *HealthResponse) {
	fmt.Printf("🏥 健康检查端点: http://localhost:%d/health\n", port)

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
		fmt.Printf("❌ 健康检查服务失败: %v\n", err)
	}
}
