package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yeisme/taskbridge/internal/syncaudit"
)

var (
	syncTarget   string
	syncStrategy string
)

var syncDiffCmd = &cobra.Command{
	Use:   "diff <source>",
	Short: "预览同步差异",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncDiff,
}

var syncConflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "列出同步冲突",
	RunE:  runSyncConflicts,
}

var syncResolveCmd = &cobra.Command{
	Use:   "resolve <conflict-id>",
	Short: "解决同步冲突",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncResolve,
}

var syncBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "同步数据备份",
}

var syncBackupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "创建本地数据快照",
	RunE:  runSyncBackupCreate,
}

var syncBackupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-id>",
	Short: "恢复本地数据快照",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncBackupRestore,
}

var syncAuditCmd = &cobra.Command{
	Use:   "audit <session-id>",
	Short: "查看同步审计记录",
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
		cmd.Flags().StringVarP(&syncOutput, "output", "o", "text", "输出格式 (text, json)")
	}
	syncDiffCmd.Flags().StringVar(&syncTarget, "target", "", "目标 provider")
	syncResolveCmd.Flags().StringVar(&syncStrategy, "strategy", "manual", "解决策略")
}

func runSyncDiff(_ *cobra.Command, args []string) error {
	taskStore, cleanup, err := getStore()
	defer cleanup()
	if err != nil {
		return commandError("初始化存储失败", err)
	}
	store := syncaudit.Store{BasePath: cfg.Storage.Path}
	session, err := store.Diff(context.Background(), taskStore, args[0], syncTarget)
	if err != nil {
		return commandError("生成同步 diff 失败", err)
	}
	if err := store.SaveSession(session); err != nil {
		return commandError("保存同步审计失败", err)
	}
	return printStructured(syncOutput, session, func() {
		fmt.Printf("sync diff %s -> %s\n", session.Source, session.Target)
		fmt.Printf("operations: %d\n", len(session.Operations))
		for _, op := range session.Operations {
			fmt.Printf("- %s %s %s\n", op.Type, op.LocalID, op.Title)
		}
		fmt.Printf("audit: %s\n", session.ID)
	})
}

func runSyncConflicts(_ *cobra.Command, _ []string) error {
	conflicts, err := syncaudit.Store{BasePath: cfg.Storage.Path}.ListConflicts()
	if err != nil {
		return commandError("读取同步冲突失败", err)
	}
	payload := map[string]interface{}{"schema": "taskbridge.conflicts.v1", "status": "ok", "count": len(conflicts), "conflicts": conflicts}
	return printStructured(syncOutput, payload, func() {
		fmt.Printf("conflicts: %d\n", len(conflicts))
		for _, conflict := range conflicts {
			fmt.Printf("- %s %s %v\n", conflict.ID, conflict.LocalID, conflict.FieldConflicts)
		}
	})
}

func runSyncResolve(_ *cobra.Command, args []string) error {
	conflict, err := syncaudit.Store{BasePath: cfg.Storage.Path}.ResolveConflict(args[0], syncStrategy)
	if err != nil {
		return commandError("解决同步冲突失败", err)
	}
	return printStructured(syncOutput, conflict, func() {
		fmt.Printf("resolved %s with %s\n", conflict.ID, syncStrategy)
	})
}

func runSyncBackupCreate(_ *cobra.Command, _ []string) error {
	result, err := syncaudit.Store{BasePath: cfg.Storage.Path}.CreateBackup()
	if err != nil {
		return commandError("创建备份失败", err)
	}
	return printStructured(syncOutput, result, func() {
		fmt.Printf("backup created: %s\n", result["id"])
	})
}

func runSyncBackupRestore(_ *cobra.Command, args []string) error {
	result, err := syncaudit.Store{BasePath: cfg.Storage.Path}.RestoreBackup(args[0])
	if err != nil {
		return commandError("恢复备份失败", err)
	}
	return printStructured(syncOutput, result, func() {
		fmt.Printf("backup restored: %s\n", result["id"])
	})
}

func runSyncAudit(_ *cobra.Command, args []string) error {
	session, err := syncaudit.Store{BasePath: cfg.Storage.Path}.LoadSession(args[0])
	if err != nil {
		return commandError("读取同步审计失败", err)
	}
	return printStructured(syncOutput, session, func() {
		fmt.Printf("sync audit %s\n", session.ID)
		fmt.Printf("status: %s operations: %d\n", session.Status, len(session.Operations))
	})
}
