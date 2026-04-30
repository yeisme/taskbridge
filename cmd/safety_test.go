package cmd

import "testing"

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
