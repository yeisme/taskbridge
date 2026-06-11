package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/clioutput"
	"github.com/yeisme/taskbridge/internal/syncaudit"
)

var (
	syncTarget   string
	syncStrategy string
)

var syncDiffCmd = &cobra.Command{
	Use:   "diff <source>",
	Short: "Preview sync differences",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncDiff,
}

var syncConflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "List synchronization conflicts",
	RunE:  runSyncConflicts,
}

var syncResolveCmd = &cobra.Command{
	Use:   "resolve <conflict-id>",
	Short: "Resolve synchronization conflicts",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncResolve,
}

var syncBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Sync data backup",
}

var syncBackupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create local data snapshot",
	RunE:  runSyncBackupCreate,
}

var syncBackupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-id>",
	Short: "Restore local data snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncBackupRestore,
}

var syncAuditCmd = &cobra.Command{
	Use:   "audit <session-id>",
	Short: "View sync audit records",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncAudit,
}

func init() {
	syncCmd.AddCommand(syncDiffCmd)
	syncCmd.AddCommand(syncConflictsCmd)
	syncCmd.AddCommand(syncResolveCmd)
	syncCmd.AddCommand(syncBackupCmd)
	syncCmd.AddCommand(syncAuditCmd)
	syncBackupCmd.AddCommand(syncBackupCreateCmd)
	syncBackupCmd.AddCommand(syncBackupRestoreCmd)

	for _, cmd := range []*cobra.Command{syncDiffCmd, syncConflictsCmd, syncResolveCmd, syncBackupCreateCmd, syncBackupRestoreCmd, syncAuditCmd} {
		cmd.Flags().StringVarP(&syncOutput, "output", "o", "text", "Output format (text, json)")
	}
	syncDiffCmd.Flags().StringVar(&syncTarget, "target", "", "target provider")
	syncResolveCmd.Flags().StringVar(&syncStrategy, "strategy", "manual", "solution strategy")
}

func runSyncDiff(_ *cobra.Command, args []string) error {
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}
	taskStore, cleanup, err := getStore()
	if err != nil {
		return commandError("Failed to initialize storage", err)
	}
	defer cleanup()
	store := syncaudit.Store{BasePath: cfg.Storage.Path}
	session, err := store.Diff(context.Background(), taskStore, args[0], syncTarget)
	if err != nil {
		return commandError("Failed to generate sync diff", err)
	}
	if err := store.SaveSession(session); err != nil {
		return commandError("Failed to save sync audit", err)
	}
	projection := buildSyncDiffProjection(session)
	return printProjection(syncOutput, projection, func() {
		fmt.Print(renderSyncControlProjection(projection))
	})
}

func runSyncConflicts(_ *cobra.Command, _ []string) error {
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}
	conflicts, err := syncaudit.Store{BasePath: cfg.Storage.Path}.ListConflicts()
	if err != nil {
		return commandError("Read sync conflict failed", err)
	}
	projection := buildSyncConflictsProjection(conflicts)
	return printProjection(syncOutput, projection, func() {
		fmt.Print(renderSyncControlProjection(projection))
	})
}

func runSyncResolve(_ *cobra.Command, args []string) error {
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}
	conflict, err := syncaudit.Store{BasePath: cfg.Storage.Path}.ResolveConflict(args[0], syncStrategy)
	if err != nil {
		return commandError("Resolve synchronization conflicts failed", err)
	}
	projection := buildSyncResolveProjection(conflict)
	return printProjection(syncOutput, projection, func() {
		fmt.Print(renderSyncControlProjection(projection))
	})
}

func runSyncBackupCreate(_ *cobra.Command, _ []string) error {
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}
	result, err := syncaudit.Store{BasePath: cfg.Storage.Path}.CreateBackup()
	if err != nil {
		return commandError("Failed to create backup", err)
	}
	projection := buildSyncBackupCreateProjection(result)
	return printProjection(syncOutput, projection, func() {
		fmt.Print(renderSyncControlProjection(projection))
	})
}

func runSyncBackupRestore(_ *cobra.Command, args []string) error {
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}
	result, err := syncaudit.Store{BasePath: cfg.Storage.Path}.RestoreBackup(args[0])
	if err != nil {
		return commandError("Restoring backup failed", err)
	}
	projection := buildSyncBackupRestoreProjection(result)
	return printProjection(syncOutput, projection, func() {
		fmt.Print(renderSyncControlProjection(projection))
	})
}

func runSyncAudit(_ *cobra.Command, args []string) error {
	if err := ensureSyncProjectionMode(); err != nil {
		return err
	}
	session, err := syncaudit.Store{BasePath: cfg.Storage.Path}.LoadSession(args[0])
	if err != nil {
		return commandError("Read sync audit failed", err)
	}
	projection := buildSyncAuditProjection(session)
	return printProjection(syncOutput, projection, func() {
		fmt.Print(renderSyncControlProjection(projection))
	})
}

func buildSyncDiffProjection(session *syncaudit.Session) clioutput.Projection {
	p := clioutput.New("sync.diff")
	if session == nil {
		p.Status = clioutput.StatusFailed
		p.Summary = "Sync diff did not return a session."
		return p
	}
	plannedWrites := session.Stats.Created + session.Stats.Updated + session.Stats.Deleted
	p.Summary = fmt.Sprintf("Sync diff compared %s to %s.", session.Source, displaySyncTarget(session.Target))
	p.Facts["session_id"] = session.ID
	p.Facts["source"] = session.Source
	p.Facts["target"] = session.Target
	p.Facts["dry_run"] = session.DryRun
	p.Facts["compared"] = len(session.Operations)
	p.Facts["planned_writes"] = plannedWrites
	p.Facts["written"] = 0
	p.Facts["created"] = session.Stats.Created
	p.Facts["updated"] = session.Stats.Updated
	p.Facts["deleted"] = session.Stats.Deleted
	p.Facts["skipped"] = session.Stats.Skipped
	p.Facts["conflicts"] = session.Stats.Conflicts
	p.Facts["errors"] = session.Stats.Errors
	p.Data = session
	p.Evidence = append(p.Evidence, "audit session "+session.ID)
	return p
}

func buildSyncConflictsProjection(conflicts []syncaudit.Conflict) clioutput.Projection {
	p := clioutput.New("sync.conflicts")
	p.Summary = fmt.Sprintf("Sync conflicts loaded: %d.", len(conflicts))
	p.Facts["count"] = len(conflicts)
	p.Facts["compared"] = len(conflicts)
	p.Facts["written"] = 0
	p.Data = map[string]any{
		"schema":    "taskbridge.conflicts.v1",
		"count":     len(conflicts),
		"conflicts": conflicts,
	}
	return p
}

func buildSyncResolveProjection(conflict *syncaudit.Conflict) clioutput.Projection {
	p := clioutput.New("sync.resolve")
	if conflict == nil {
		p.Status = clioutput.StatusFailed
		p.Summary = "Sync conflict resolution did not return a conflict."
		return p
	}
	p.Summary = fmt.Sprintf("Sync conflict %s resolved.", conflict.ID)
	p.Facts["conflict_id"] = conflict.ID
	p.Facts["local_id"] = conflict.LocalID
	p.Facts["strategy"] = syncStrategy
	p.Facts["status"] = conflict.Status
	p.Facts["compared"] = 1
	p.Facts["written"] = 1
	p.Data = conflict
	return p
}

func buildSyncBackupCreateProjection(result map[string]interface{}) clioutput.Projection {
	p := clioutput.New("sync.backup.create")
	id, _ := result["id"].(string)
	files, fileCount := syncBackupFileCount(result["files"])
	p.Summary = fmt.Sprintf("Sync backup created: %s.", id)
	p.Facts["backup_id"] = id
	p.Facts["files"] = fileCount
	p.Facts["compared"] = fileCount
	p.Facts["written"] = fileCount
	p.Data = result
	if files != "" {
		p.Evidence = append(p.Evidence, files)
	}
	return p
}

func buildSyncBackupRestoreProjection(result map[string]interface{}) clioutput.Projection {
	p := clioutput.New("sync.backup.restore")
	id, _ := result["id"].(string)
	files, fileCount := syncBackupFileCount(result["restored"])
	p.Summary = fmt.Sprintf("Sync backup restored: %s.", id)
	p.Facts["backup_id"] = id
	p.Facts["restored_files"] = fileCount
	p.Facts["compared"] = fileCount
	p.Facts["written"] = fileCount
	p.Data = result
	if files != "" {
		p.Evidence = append(p.Evidence, files)
	}
	return p
}

func buildSyncAuditProjection(session *syncaudit.Session) clioutput.Projection {
	p := clioutput.New("sync.audit")
	if session == nil {
		p.Status = clioutput.StatusFailed
		p.Summary = "Sync audit session was not found."
		return p
	}
	p.Summary = fmt.Sprintf("Sync audit %s loaded.", session.ID)
	p.Facts["session_id"] = session.ID
	p.Facts["status"] = session.Status
	p.Facts["source"] = session.Source
	p.Facts["target"] = session.Target
	p.Facts["dry_run"] = session.DryRun
	p.Facts["compared"] = len(session.Operations)
	p.Facts["planned_writes"] = session.Stats.Created + session.Stats.Updated + session.Stats.Deleted
	p.Facts["written"] = 0
	p.Facts["created"] = session.Stats.Created
	p.Facts["updated"] = session.Stats.Updated
	p.Facts["deleted"] = session.Stats.Deleted
	p.Facts["skipped"] = session.Stats.Skipped
	p.Facts["conflicts"] = session.Stats.Conflicts
	p.Facts["errors"] = session.Stats.Errors
	p.Data = session
	return p
}

func renderSyncControlProjection(projection clioutput.Projection) string {
	return clioutput.RenderSummary(projection) + "\n" + renderProjectionMetrics("Sync audit", projection, []metricRow{
		{Label: "Session", Key: "session_id"},
		{Label: "Source", Key: "source"},
		{Label: "Target", Key: "target"},
		{Label: "Dry run", Key: "dry_run"},
		{Label: "Compared", Key: "compared"},
		{Label: "Planned writes", Key: "planned_writes"},
		{Label: "Written", Key: "written"},
		{Label: "Created", Key: "created"},
		{Label: "Updated", Key: "updated"},
		{Label: "Deleted", Key: "deleted"},
		{Label: "Skipped", Key: "skipped"},
		{Label: "Conflicts", Key: "conflicts"},
		{Label: "Errors", Key: "errors"},
		{Label: "Backup", Key: "backup_id"},
		{Label: "Files", Key: "files"},
		{Label: "Restored files", Key: "restored_files"},
		{Label: "Conflict", Key: "conflict_id"},
		{Label: "Strategy", Key: "strategy"},
		{Label: "Status", Key: "status"},
	})
}

func displaySyncTarget(target string) string {
	if target == "" {
		return "empty target"
	}
	return target
}

func syncBackupFileCount(value interface{}) (string, int) {
	files, ok := value.([]string)
	if !ok {
		return "", 0
	}
	return fmt.Sprintf("files: %v", files), len(files)
}
