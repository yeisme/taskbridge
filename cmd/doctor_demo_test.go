package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withIsolatedHome sets TASKBRIDGE_HOME to a temp dir and restores it after the test.
func withIsolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	old := os.Getenv("TASKBRIDGE_HOME")
	if err := os.Setenv("TASKBRIDGE_HOME", home); err != nil {
		t.Fatalf("Setenv TASKBRIDGE_HOME: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Setenv("TASKBRIDGE_HOME", old); err != nil {
			t.Fatalf("restore TASKBRIDGE_HOME: %v", err)
		}
	})
	return home
}

// TestDoctorRecommendsDemoToday verifies that when no provider is authenticated,
// doctor recommends "taskbridge demo today" (not the old "today --mock").
func TestDoctorRecommendsDemoToday(t *testing.T) {
	home := withIsolatedHome(t)
	storagePath := filepath.Join(home, "data")
	setupTestConfig(t, storagePath)

	result := buildDoctorResult()

	if result.NextAction != "taskbridge demo today" {
		t.Fatalf("doctor next_action = %q, want \"taskbridge demo today\"", result.NextAction)
	}
	for _, check := range result.Checks {
		if check.ID == "provider_auth" && !strings.Contains(check.NextAction, "demo today") {
			t.Fatalf("provider_auth next_action = %q, want it to mention demo today", check.NextAction)
		}
	}
}

// TestQuickstartRecommendsDemoToday verifies quickstart output mentions demo today
// when no provider is authenticated.
func TestQuickstartRecommendsDemoToday(t *testing.T) {
	home := withIsolatedHome(t)
	storagePath := filepath.Join(home, "data")
	setupTestConfig(t, storagePath)

	quickstartFormat = "text"
	defer func() { quickstartFormat = "" }()

	output := captureStdout(t, func() {
		if err := runQuickstart(nil, nil); err != nil {
			t.Fatalf("runQuickstart: %v", err)
		}
	})

	if !strings.Contains(output, "demo today") {
		t.Fatalf("quickstart should recommend demo today when no provider is authenticated:\n%s", output)
	}
	if strings.Contains(output, "today --mock") {
		t.Fatalf("quickstart should not recommend today --mock to users:\n%s", output)
	}
}
