package cmd

import (
	"errors"
	"testing"

	"github.com/yeisme/taskbridge/internal/sync"
)

func TestAgentExecuteConfirmDisablesDryRun(t *testing.T) {
	if effectiveAgentExecuteDryRun(true, true) {
		t.Fatalf("expected --confirm to execute even when dry-run default is true")
	}
	if !effectiveAgentExecuteDryRun(true, false) {
		t.Fatalf("expected default agent execute to stay dry-run")
	}
	if effectiveAgentExecuteDryRun(false, false) {
		t.Fatalf("expected explicit dry-run=false to be false")
	}
}

func TestReviewApplyRequiresDryRunOrConfirm(t *testing.T) {
	if err := validateReviewApplyMode(false, false); err == nil {
		t.Fatalf("expected apply-file without dry-run or confirm to fail")
	}
	if err := validateReviewApplyMode(true, false); err != nil {
		t.Fatalf("expected dry-run apply to be allowed: %v", err)
	}
	if err := validateReviewApplyMode(false, true); err != nil {
		t.Fatalf("expected confirmed apply to be allowed: %v", err)
	}
}

func TestSyncDeleteRequiresDryRunOrConfirm(t *testing.T) {
	if err := validateSyncWriteMode(false, false, false, true); err == nil {
		t.Fatalf("expected sync delete without dry-run or confirm to fail")
	}
	if err := validateSyncWriteMode(true, false, false, true); err != nil {
		t.Fatalf("expected sync delete dry-run preview to be allowed: %v", err)
	}
	if err := validateSyncWriteMode(false, true, false, true); err != nil {
		t.Fatalf("expected confirmed sync delete to be allowed: %v", err)
	}
}

func TestSyncForceRequiresDryRunOrConfirm(t *testing.T) {
	if err := validateSyncWriteMode(false, false, true, false); err == nil {
		t.Fatalf("expected sync force without dry-run or confirm to fail")
	}
	if err := validateSyncWriteMode(true, false, true, false); err != nil {
		t.Fatalf("expected sync force dry-run preview to be allowed: %v", err)
	}
	if err := validateSyncWriteMode(false, true, true, false); err != nil {
		t.Fatalf("expected confirmed sync force to be allowed: %v", err)
	}
}

// TestSyncRunErrorMapsConfirmationToUsage verifies the engine confirmation
// error surfaces as a usage error (exit 2), matching the --delete/--force
// pre-checks, while generic sync errors stay command errors (exit 1).
func TestSyncRunErrorMapsConfirmationToUsage(t *testing.T) {
	var cliErr *CLIError

	err := syncRunError(&sync.ConfirmationRequiredError{Operation: "overwrite"})
	if !errors.As(err, &cliErr) || cliErr.ExitCode != 2 {
		t.Fatalf("expected usage error (exit 2) for confirmation error, got %+v", err)
	}

	err = syncRunError(&sync.ConfirmationRequiredError{Operation: "delete"})
	if !errors.As(err, &cliErr) || cliErr.ExitCode != 2 {
		t.Fatalf("expected usage error (exit 2) for delete confirmation, got %+v", err)
	}

	err = syncRunError(errors.New("network down"))
	if !errors.As(err, &cliErr) || cliErr.ExitCode != 1 {
		t.Fatalf("expected command error (exit 1) for generic error, got %+v", err)
	}
}

// TestServeSyncConfirmFlagRegistered verifies the scheduled-sync confirmation
// flag exists on serve and defaults to off (safe default).
func TestServeSyncConfirmFlagRegistered(t *testing.T) {
	flag := serveCmd.Flags().Lookup("sync-confirm")
	if flag == nil {
		t.Fatal("--sync-confirm flag not registered on serve command")
	}
	if flag.DefValue != "false" {
		t.Fatalf("--sync-confirm default = %q, want \"false\"", flag.DefValue)
	}
}

// TestSyncBidirectionalAndWatchExposeConfirmFlag verifies the --confirm flag is
// wired onto the bidirectional and watch subcommands (regression guard for the
// sync-write-safety entry-point fix).
func TestSyncBidirectionalAndWatchExposeConfirmFlag(t *testing.T) {
	for _, name := range []string{"bidirectional", "watch", "push"} {
		sub, _, err := rootCmd.Find([]string{"sync", name})
		if err != nil {
			t.Fatalf("could not find sync %s: %v", name, err)
		}
		if sub.Flags().Lookup("confirm") == nil {
			t.Fatalf("sync %s missing --confirm flag", name)
		}
	}
}
