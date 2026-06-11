package clioutput

import "testing"

func TestDisplayWidthHandlesCJKAndEmoji(t *testing.T) {
	if DisplayWidth("飞书") != 4 {
		t.Fatalf("DisplayWidth(飞书) = %d, want 4", DisplayWidth("飞书"))
	}
	if DisplayWidth("🔴") < 1 {
		t.Fatalf("DisplayWidth emoji should be positive")
	}
}

func TestTruncateDisplayRespectsDisplayWidth(t *testing.T) {
	got := TruncateDisplay("飞书任务管理", 7)
	if DisplayWidth(got) > 7 {
		t.Fatalf("TruncateDisplay width = %d, want <= 7 for %q", DisplayWidth(got), got)
	}
	if got == "飞书任务管理" {
		t.Fatalf("TruncateDisplay did not truncate: %q", got)
	}
}
