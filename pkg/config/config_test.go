package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigIncludesIntelligenceDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Intelligence.Enabled {
		t.Fatalf("expected intelligence.enabled=true")
	}
	if cfg.Intelligence.Timezone != "Asia/Shanghai" {
		t.Fatalf("unexpected timezone: %s", cfg.Intelligence.Timezone)
	}
	if cfg.Intelligence.Overdue.WarningThreshold != 3 {
		t.Fatalf("unexpected warning threshold: %d", cfg.Intelligence.Overdue.WarningThreshold)
	}
	if cfg.Intelligence.Decompose.PreferredStrategy != "project_split" {
		t.Fatalf("unexpected preferred strategy: %s", cfg.Intelligence.Decompose.PreferredStrategy)
	}
	if len(cfg.Intelligence.Decompose.AbstractKeywords) == 0 {
		t.Fatalf("expected abstract keyword defaults")
	}
}

func TestValidateCoversIntelligenceRules(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Intelligence.Overdue.OverloadThreshold = 1
	cfg.Intelligence.Overdue.WarningThreshold = 2
	cfg.Intelligence.Decompose.ComplexityThreshold = 120

	issues := cfg.Validate()
	if !hasIssue(issues, ValidationLevelError, "intelligence.overdue.overload_threshold") {
		t.Fatalf("expected overload threshold error: %#v", issues)
	}
	if !hasIssue(issues, ValidationLevelError, "intelligence.decomposition.complexity_threshold") {
		t.Fatalf("expected complexity threshold error: %#v", issues)
	}
}

func TestLoadPreservesDefaultsForMissingIntelligenceFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	content := []byte("intelligence:\n  enabled: true\n")
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Intelligence.Timezone != "Asia/Shanghai" {
		t.Fatalf("expected default timezone, got %s", cfg.Intelligence.Timezone)
	}
	if cfg.Intelligence.Overdue.MaxCandidates != 30 {
		t.Fatalf("expected default max candidates, got %d", cfg.Intelligence.Overdue.MaxCandidates)
	}
}

func hasIssue(issues []ValidationIssue, level, field string) bool {
	for _, issue := range issues {
		if issue.Level == level && issue.Field == field {
			return true
		}
	}
	return false
}
