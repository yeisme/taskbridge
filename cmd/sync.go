package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/sync"
	"github.com/yeisme/taskbridge/pkg/ui"
)

// syncCmd synchronization command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Task synchronization",
	Long: `Synchronize local tasks with remote Todo providers.

Supported synchronization modes:
  - pull: copy remote changes into local storage
  - push: copy local changes to the remote provider
  - bidirectional: run pull and push in one operation

Examples:
  taskbridge sync pull google
  taskbridge sync push google --dry-run
  taskbridge sync bidirectional google
  taskbridge sync watch google --interval 5m`,
}

// syncPullCmd pull command
var syncPullCmd = &cobra.Command{
	Use:   "pull <provider>",
	Short: "Pull tasks from remote to local",
	Long: `Pull all tasks from the selected provider into local storage.

Examples:
  taskbridge sync pull google
  taskbridge sync pull google --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncPull,
}

// syncPushCmd push command
var syncPushCmd = &cobra.Command{
	Use:   "push <provider>",
	Short: "Push tasks from local to remote",
	Long: `Push local task changes to the selected provider.

Examples:
  taskbridge sync push google
  taskbridge sync push google --force`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncPush,
}

// syncBidirectionalCmd bidirectional synchronization command
var syncBidirectionalCmd = &cobra.Command{
	Use:   "bidirectional <provider>",
	Short: "Two-way synchronization tasks",
	Long: `Run bidirectional synchronization by pulling remote changes before pushing local changes.

Example:
  taskbridge sync bidirectional google`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncBidirectional,
}

// syncWatchCmd continuous synchronization command
var syncWatchCmd = &cobra.Command{
	Use:   "watch <provider>",
	Short: "Continuously monitor and synchronize",
	Long: `Continuously monitor and synchronize tasks regularly.

Example:
  taskbridge sync watch google
  taskbridge sync watch google --interval 5m`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncWatch,
}

// syncStatusCmd synchronization status command
var syncStatusCmd = &cobra.Command{
	Use:   "status [provider]",
	Short: "View sync status",
	Long: `Displays the synchronization status of the specified Provider or all Providers.

Example:
  taskbridge sync status
  taskbridge sync status google`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSyncStatus,
}

var (
	syncDryRun       bool
	syncForce        bool
	syncInterval     time.Duration
	syncOutput       string
	syncDeleteRemote bool
)

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncBidirectionalCmd)
	syncCmd.AddCommand(syncWatchCmd)
	syncCmd.AddCommand(syncStatusCmd)

	//General options
	for _, cmd := range []*cobra.Command{syncPullCmd, syncPushCmd, syncBidirectionalCmd} {
		cmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Simulated execution, no actual synchronization")
		cmd.Flags().BoolVar(&syncForce, "force", false, "Force sync, ignore conflict detection")
		cmd.Flags().StringVarP(&syncOutput, "output", "o", "text", "Output format (text, json)")
	}
	syncStatusCmd.Flags().StringVarP(&syncOutput, "output", "o", "text", "Output format (text, json)")

	//Push command specific options
	syncPushCmd.Flags().BoolVar(&syncDeleteRemote, "delete", false, "Delete tasks that exist remotely but not locally")

	//watch command options
	syncWatchCmd.Flags().DurationVar(&syncInterval, "interval", 5*time.Minute, "synchronization interval")
}

// getSyncEngine gets the synchronization engine
func getSyncEngine() (*sync.Engine, error) {
	return getSyncEngineForProvider("")
} //getSyncEngineForProvider Gets the synchronization engine of the specified Provider
func getSyncEngineForProvider(providerName string) (*sync.Engine, error) { //Parse provider abbreviation
	providerName = provider.ResolveProviderName(providerName) //Create storage
	store, cleanup, err := getStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	defer cleanup()

	providers, err := loadAuthenticatedProviders(providerName)
	if err != nil {
		return nil, err
	}

	return sync.NewEngine(providers, store), nil
}

// runSyncPull executes pull
func runSyncPull(cmd *cobra.Command, args []string) error {
	providerName := provider.ResolveProviderName(args[0])
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}

	engine, err := getSyncEngineForProvider(providerName)
	if err != nil {
		return commandError("Failed to initialize sync engine", err)
	}

	opts := sync.Options{
		Direction: sync.DirectionPull,
		Provider:  providerName,
		DryRun:    syncDryRun,
		Force:     syncForce,
	}

	result, err := engine.Sync(context.Background(), opts)
	if err != nil {
		return commandError("Sync failed", err)
	}

	return printSyncResult(result)
}

// runSyncPush executes push
func runSyncPush(cmd *cobra.Command, args []string) error {
	providerName := provider.ResolveProviderName(args[0])
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}

	engine, err := getSyncEngineForProvider(providerName)
	if err != nil {
		return commandError("Failed to initialize sync engine", err)
	}

	opts := sync.Options{
		Direction:    sync.DirectionPush,
		Provider:     providerName,
		DryRun:       syncDryRun,
		Force:        syncForce,
		DeleteRemote: syncDeleteRemote,
	}

	result, err := engine.Sync(context.Background(), opts)
	if err != nil {
		return commandError("Sync failed", err)
	}

	return printSyncResult(result)
}

// runSyncBidirectional performs bidirectional synchronization
func runSyncBidirectional(cmd *cobra.Command, args []string) error {
	providerName := provider.ResolveProviderName(args[0])
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}

	engine, err := getSyncEngineForProvider(providerName)
	if err != nil {
		return commandError("Failed to initialize sync engine", err)
	}

	opts := sync.Options{
		Direction: sync.DirectionBidirectional,
		Provider:  providerName,
		DryRun:    syncDryRun,
		Force:     syncForce,
	}

	result, err := engine.Sync(context.Background(), opts)
	if err != nil {
		return commandError("Sync failed", err)
	}

	return printSyncResult(result)
}

// runSyncWatch performs continuous synchronization
func runSyncWatch(cmd *cobra.Command, args []string) error {
	providerName := provider.ResolveProviderName(args[0])
	if resolveOutputFormat("text") != "text" || IsQuietMode() {
		return usageError("sync watch does not support machine output; omit --json, --agent, --events, --explain, and --quiet")
	}

	engine, err := getSyncEngineForProvider(providerName)
	if err != nil {
		return commandError("Failed to initialize sync engine", err)
	}

	opts := sync.Options{
		Direction: sync.DirectionBidirectional,
		Provider:  providerName,
	}

	fmt.Printf("🔄 Start continuous synchronization %s (interval: %v)\n", providerName, syncInterval)
	fmt.Println("Press Ctrl+C to stop")

	err = engine.Watch(context.Background(), opts, syncInterval)
	if err != nil {
		return commandError("Continuous synchronization failed", err)
	}
	return nil
}

// runSyncStatus execution status query
func runSyncStatus(cmd *cobra.Command, args []string) error {
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}
	engine, err := getSyncEngine()
	if err != nil {
		return commandError("Failed to initialize sync engine", err)
	}

	statuses := make([]*sync.Status, 0)
	if len(args) > 0 {
		status, err := engine.GetStatus(context.Background(), args[0])
		if err != nil {
			return commandError("Failed to get sync status", err)
		}
		statuses = append(statuses, status)
	} else {
		for _, p := range provider.GetAllProviderNames() {
			status, err := engine.GetStatus(context.Background(), p)
			if err != nil {
				continue
			}
			statuses = append(statuses, status)
		}
	}

	projection := buildSyncStatusProjection(statuses)
	return printProjection(syncOutput, projection, func() {
		fmt.Print(renderSyncStatus(projection))
	})
}

func ensureSyncProjectionMode() error {
	if resolveOutputFormat(syncOutput) == "events" {
		return usageError("sync commands do not support --events; use --json or --agent for one-shot machine output")
	}
	return nil
}

type syncResultReceipt struct {
	Provider      string         `json:"provider"`
	Direction     sync.Direction `json:"direction"`
	DryRun        bool           `json:"dry_run"`
	Compared      int            `json:"compared"`
	PlannedWrites int            `json:"planned_writes"`
	Written       int            `json:"written"`
	Pulled        int            `json:"pulled"`
	Pushed        int            `json:"pushed"`
	Updated       int            `json:"updated"`
	Deleted       int            `json:"deleted"`
	Skipped       int            `json:"skipped"`
	Errors        []sync.Error   `json:"errors,omitempty"`
	Duration      string         `json:"duration"`
	LastSyncTime  time.Time      `json:"last_sync_time"`
}

func buildSyncResultProjection(result *sync.Result) clioutput.Projection {
	command := "sync.result"
	if result != nil && result.Direction != "" {
		command = "sync." + string(result.Direction)
	}
	p := clioutput.New(command)
	if result == nil {
		p.Status = clioutput.StatusFailed
		p.Summary = "Sync did not return a result."
		return p
	}

	plannedWrites := result.Pulled + result.Pushed + result.Updated + result.Deleted
	written := plannedWrites
	if syncDryRun {
		written = 0
	}
	compared := plannedWrites + result.Skipped + len(result.Errors)

	p.Summary = fmt.Sprintf("Sync %s completed for %s.", result.Direction, result.Provider)
	if syncDryRun {
		p.Summary = fmt.Sprintf("Sync %s preview completed for %s; no data was modified.", result.Direction, result.Provider)
	}
	if len(result.Errors) > 0 {
		p.Status = clioutput.StatusPartial
		p.Risks = append(p.Risks, fmt.Sprintf("%d sync operations reported errors.", len(result.Errors)))
	}
	p.Facts["provider"] = result.Provider
	p.Facts["direction"] = string(result.Direction)
	p.Facts["dry_run"] = syncDryRun
	p.Facts["compared"] = compared
	p.Facts["planned_writes"] = plannedWrites
	p.Facts["written"] = written
	p.Facts["pulled"] = result.Pulled
	p.Facts["pushed"] = result.Pushed
	p.Facts["updated"] = result.Updated
	p.Facts["deleted"] = result.Deleted
	p.Facts["skipped"] = result.Skipped
	p.Facts["errors"] = len(result.Errors)
	p.Facts["duration"] = result.Duration.String()
	p.Data = syncResultReceipt{
		Provider:      result.Provider,
		Direction:     result.Direction,
		DryRun:        syncDryRun,
		Compared:      compared,
		PlannedWrites: plannedWrites,
		Written:       written,
		Pulled:        result.Pulled,
		Pushed:        result.Pushed,
		Updated:       result.Updated,
		Deleted:       result.Deleted,
		Skipped:       result.Skipped,
		Errors:        result.Errors,
		Duration:      result.Duration.String(),
		LastSyncTime:  result.LastSyncTime,
	}
	p.Actions = append(p.Actions, clioutput.Action{Name: "status", Command: "taskbridge sync status " + result.Provider})
	return p
}

// printSyncResult prints synchronization results.
func printSyncResult(result *sync.Result) error {
	projection := buildSyncResultProjection(result)
	return printProjection(syncOutput, projection, func() {
		fmt.Print(renderSyncResult(projection))
	})
}

func renderSyncResult(projection clioutput.Projection) string {
	return clioutput.RenderSummary(projection) + "\n" + renderProjectionMetrics("Sync result", projection, []metricRow{
		{Label: "Provider", Key: "provider"},
		{Label: "Direction", Key: "direction"},
		{Label: "Dry run", Key: "dry_run"},
		{Label: "Compared", Key: "compared"},
		{Label: "Planned writes", Key: "planned_writes"},
		{Label: "Written", Key: "written"},
		{Label: "Pulled", Key: "pulled"},
		{Label: "Pushed", Key: "pushed"},
		{Label: "Updated", Key: "updated"},
		{Label: "Deleted", Key: "deleted"},
		{Label: "Skipped", Key: "skipped"},
		{Label: "Errors", Key: "errors"},
		{Label: "Duration", Key: "duration"},
	})
}

type syncStatusRow struct {
	Provider       string `json:"provider"`
	Name           string `json:"name"`
	Authenticated  bool   `json:"authenticated"`
	LastSyncTime   string `json:"last_sync_time"`
	PendingChanges int    `json:"pending_changes"`
}

func buildSyncStatusProjection(statuses []*sync.Status) clioutput.Projection {
	p := clioutput.New("sync.status")
	rows := make([]syncStatusRow, 0, len(statuses))
	authenticated := 0
	pending := 0
	for _, status := range statuses {
		if status == nil {
			continue
		}
		if status.Authenticated {
			authenticated++
		}
		pending += status.PendingChanges
		lastSync := "never"
		if !status.LastSyncTime.IsZero() {
			lastSync = status.LastSyncTime.Format(time.RFC3339)
		}
		rows = append(rows, syncStatusRow{
			Provider:       status.Provider,
			Name:           syncProviderDisplayName(status.Provider),
			Authenticated:  status.Authenticated,
			LastSyncTime:   lastSync,
			PendingChanges: status.PendingChanges,
		})
	}

	p.Summary = fmt.Sprintf("Sync status loaded for %d provider(s).", len(rows))
	p.Facts["providers"] = len(rows)
	p.Facts["authenticated"] = authenticated
	p.Facts["pending_changes"] = pending
	p.Facts["compared"] = len(rows)
	p.Facts["written"] = 0
	if len(rows) == 1 {
		p.Facts["provider"] = rows[0].Provider
		p.Facts["last_sync_time"] = rows[0].LastSyncTime
		p.Facts["provider_authenticated"] = rows[0].Authenticated
	}
	p.Data = map[string]any{"providers": rows}
	return p
}

func renderSyncStatus(projection clioutput.Projection) string {
	var out string
	out += clioutput.RenderSummary(projection)
	out += "\nSync providers\n\n"
	table := ui.NewSimpleTable(
		ui.Column{Header: "Provider", AlignLeft: true},
		ui.Column{Header: "Name", AlignLeft: true},
		ui.Column{Header: "Authenticated", AlignLeft: true},
		ui.Column{Header: "Last sync", AlignLeft: true},
		ui.Column{Header: "Pending", AlignRight: true},
	)
	if payload, ok := projection.Data.(map[string]any); ok {
		if rows, ok := payload["providers"].([]syncStatusRow); ok {
			for _, row := range rows {
				table.AddRow(row.Provider, row.Name, yesNo(row.Authenticated), row.LastSyncTime, fmt.Sprint(row.PendingChanges))
			}
		}
	}
	out += table.Render()
	return out
}

type metricRow struct {
	Label string
	Key   string
}

func renderProjectionMetrics(title string, projection clioutput.Projection, rows []metricRow) string {
	table := ui.NewSimpleTable(
		ui.Column{Header: "Metric", AlignLeft: true},
		ui.Column{Header: "Value", AlignLeft: true},
	)
	for _, row := range rows {
		if value, ok := projection.Facts[row.Key]; ok {
			table.AddRow(row.Label, fmt.Sprint(value))
		}
	}
	return title + "\n\n" + table.Render()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func syncProviderDisplayName(providerName string) string {
	switch providerName {
	case "google":
		return "Google Tasks"
	case "microsoft":
		return "Microsoft Todo"
	case "feishu":
		return "Feishu"
	case "ticktick":
		return "TickTick"
	case "dida":
		return "Dida365"
	case "todoist":
		return "Todoist"
	default:
		return providerName
	}
}
