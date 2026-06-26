package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yeisme/taskbridge/internal/loader"
	"github.com/yeisme/taskbridge/internal/provider"
)

func TestDailyHubIntegrationSmoke(t *testing.T) {
	tmp := t.TempDir()
	storagePath := filepath.Join(tmp, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		t.Fatal(err)
	}
	setupTestConfig(t, storagePath)

	oldLoader := loadProvidersWithStatusFunc
	loadProvidersWithStatusFunc = func(target string) *loader.ProviderLoadResult {
		statuses := map[string]*loader.ProviderLoadStatus{}
		for _, name := range provider.GetAllProviderNames() {
			statuses[name] = &loader.ProviderLoadStatus{Name: name, Error: "test no-auth fallback"}
		}
		return &loader.ProviderLoadResult{Providers: map[string]provider.Provider{}, Statuses: statuses}
	}
	defer func() { loadProvidersWithStatusFunc = oldLoader }()

	assertJSONCommand(t, "doctor", func() error {
		doctorFormat = ""
		outputJSON = true
		defer func() { outputJSON = false }()
		return runDoctor(nil, nil)
	})
	assertJSONCommand(t, "demo today", func() error {
		controlFormat = ""
		outputJSON = true
		defer func() { outputJSON = false }()
		return runDemoToday(nil, nil)
	})
	assertJSONCommand(t, "sync pull --all --dry-run", func() error {
		syncOutput = ""
		syncAll = true
		syncDryRun = true
		outputJSON = true
		defer func() { syncAll = false; syncDryRun = false; outputJSON = false }()
		return runSyncPull(nil, nil)
	})
	assertJSONCommand(t, "today", func() error {
		controlFormat = ""
		outputJSON = true
		defer func() { outputJSON = false }()
		return runToday(nil, nil)
	})
	assertJSONCommand(t, "next", func() error {
		controlFormat = ""
		outputJSON = true
		defer func() { outputJSON = false }()
		return runNext(nil, nil)
	})
}

func assertJSONCommand(t *testing.T, name string, run func() error) {
	t.Helper()
	output := captureStdout(t, func() {
		if err := run(); err != nil {
			t.Fatalf("%s failed: %v", name, err)
		}
	})
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("%s stdout should be parseable JSON:\n%s\nerror: %v", name, output, err)
	}
	if envelope["status"] == "" {
		t.Fatalf("%s JSON missing status: %#v", name, envelope)
	}
}
