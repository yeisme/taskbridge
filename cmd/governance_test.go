package cmd

import (
	"strings"
	"testing"
)

func TestGovernanceDefaultFormatIsHumanTable(t *testing.T) {
	if governanceFormat != "table" {
		t.Fatalf("governanceFormat = %q, want table", governanceFormat)
	}
}

func TestRenderGovernanceOverdueHealthUsesTableNotJSON(t *testing.T) {
	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"overdue_count":        0,
			"severe_overdue_count": 0,
			"is_warning":           false,
			"is_overload":          false,
		},
		"config_applied": map[string]interface{}{
			"max_candidates":     30,
			"overload_threshold": 10,
			"severe_days":        7,
			"warning_threshold":  3,
		},
		"actions":    []string{"defer", "reschedule", "delete", "split_then_schedule"},
		"questions":  []string{"Which tasks should be deferred?", "Which tasks should be deleted?"},
		"candidates": []interface{}{},
	}
	out := renderGovernanceOverdueHealth(result, buildGovernanceOverdueHealthProjection(result))

	for _, want := range []string{"Overdue Health", "Summary", "Overdue", "Severe", "Recommended actions", "defer", "split_then_schedule", "No overdue tasks found"} {
		if !strings.Contains(out, want) {
			t.Fatalf("renderGovernanceOverdueHealth missing %q:\n%s", want, out)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") || strings.Contains(out, "\"config_applied\"") {
		t.Fatalf("default governance output should be human table, not JSON:\n%s", out)
	}
}
