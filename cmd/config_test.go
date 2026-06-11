package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	pkgconfig "github.com/yeisme/taskbridge/pkg/config"
)

func TestConfigShowFormatJSONDoesNotAppendHumanText(t *testing.T) {
	oldCfg, oldFormat := cfg, configFormat
	cfg = pkgconfig.DefaultConfig()
	configFormat = "json"
	defer func() { cfg, configFormat = oldCfg, oldFormat }()

	output := captureStdout(t, func() {
		if err := runConfigShow(nil, nil); err != nil {
			t.Fatalf("runConfigShow: %v", err)
		}
	})
	if strings.Contains(output, "Configuration source") {
		t.Fatalf("json output should not append human source text:\n%s", output)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("config json is not parseable: %v\n%s", err, output)
	}
}

func TestConfigShowEventsRoutesThroughProjectionDispatcher(t *testing.T) {
	oldCfg, oldEvents := cfg, outputEvents
	cfg = pkgconfig.DefaultConfig()
	outputEvents = true
	t.Cleanup(func() { cfg, outputEvents = oldCfg, oldEvents })

	err := runConfigShow(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "--events") {
		t.Fatalf("config show --events should be rejected by projection dispatcher, got %v", err)
	}
}

func TestConfigShowExplainUsesProjectionRenderer(t *testing.T) {
	oldCfg, oldExplain := cfg, outputExplain
	cfg = pkgconfig.DefaultConfig()
	outputExplain = true
	t.Cleanup(func() { cfg, outputExplain = oldCfg, oldExplain })

	stdout := captureStdout(t, func() {
		if err := runConfigShow(nil, nil); err != nil {
			t.Fatalf("runConfigShow: %v", err)
		}
	})
	if !strings.Contains(stdout, "Conclusion") || strings.Contains(stdout, "storage:") {
		t.Fatalf("config show --explain should render explain sections, got:\n%s", stdout)
	}
}

func TestBuildConfigProjectionRedactsSensitiveFields(t *testing.T) {
	p := buildConfigProjection(pkgconfig.DefaultConfig())
	if p.Command != "config.show" || p.Facts["source"] == nil {
		t.Fatalf("unexpected config projection: %#v", p)
	}
}

func TestWriteValidationReportUsesSectionTableOutput(t *testing.T) {
	issues := []pkgconfig.ValidationIssue{
		{Level: pkgconfig.ValidationLevelWarning, Field: "providers.google", Message: "credentials file is missing"},
		{Level: pkgconfig.ValidationLevelError, Field: "storage.path", Message: "path is required"},
	}
	var out bytes.Buffer

	exitCode := writeValidationReport(&out, issues)
	output := out.String()

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}
	for _, want := range []string{"Configuration validation", "Summary", "Errors", "Warnings", "Field", "Message", "storage.path", "providers.google"} {
		if !strings.Contains(output, want) {
			t.Fatalf("validation output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "配置") || strings.Contains(output, "验证") || strings.Contains(output, "  - [") {
		t.Fatalf("validation output should be English section/table text, not localized bullets:\n%s", output)
	}
}

func TestConfigGetDefaultUsesHumanKeyValueOutput(t *testing.T) {
	oldCfg := cfg
	oldJSON, oldAgent := outputJSON, outputAgent
	cfg = pkgconfig.DefaultConfig()
	cfg.Storage.Path = "/tmp/taskbridge-data"
	outputJSON, outputAgent = false, false
	defer func() { cfg = oldCfg; outputJSON, outputAgent = oldJSON, oldAgent }()

	out := captureStdout(t, func() {
		if err := runConfigGet(nil, []string{"storage.path"}); err != nil {
			t.Fatalf("runConfigGet: %v", err)
		}
	})

	if !strings.Contains(out, "storage.path") || !strings.Contains(out, "/tmp/taskbridge-data") {
		t.Fatalf("config get should show key/value output, got:\n%s", out)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") || strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Fatalf("config get default should not be raw JSON:\n%s", out)
	}
	if strings.TrimSpace(out) == "/tmp/taskbridge-data" {
		t.Fatalf("config get default should include the key, not only the raw value")
	}
}

func TestConfigGetJSONUsesProjectionEnvelope(t *testing.T) {
	oldCfg := cfg
	oldJSON, oldAgent := outputJSON, outputAgent
	cfg = pkgconfig.DefaultConfig()
	cfg.App.LogLevel = "debug"
	outputJSON, outputAgent = true, false
	defer func() { cfg = oldCfg; outputJSON, outputAgent = oldJSON, oldAgent }()

	out := captureStdout(t, func() {
		if err := runConfigGet(nil, []string{"app.log_level"}); err != nil {
			t.Fatalf("runConfigGet: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("config get --json should be parseable JSON, got %v in %q", err, out)
	}
	if payload["command"] != "config.get" || payload["mode"] != "json" {
		t.Fatalf("unexpected config get envelope: %+v", payload)
	}
	facts, ok := payload["facts"].(map[string]any)
	if !ok || facts["key"] != "app.log_level" || facts["value"] != "debug" {
		t.Fatalf("unexpected config get facts: %+v", payload["facts"])
	}
}

func TestConfigGetAgentUsesParseableFacts(t *testing.T) {
	oldCfg := cfg
	oldJSON, oldAgent := outputJSON, outputAgent
	cfg = pkgconfig.DefaultConfig()
	cfg.Storage.Type = "file"
	outputJSON, outputAgent = false, true
	defer func() { cfg = oldCfg; outputJSON, outputAgent = oldJSON, oldAgent }()

	out := captureStdout(t, func() {
		if err := runConfigGet(nil, []string{"storage.type"}); err != nil {
			t.Fatalf("runConfigGet: %v", err)
		}
	})

	for _, want := range []string{"mode=agent", "command=config.get", "fact.key=storage.type", "fact.value=file"} {
		if !strings.Contains(out, want) {
			t.Fatalf("config get --agent missing %q in:\n%s", want, out)
		}
	}
}

func TestConfigValidateGlobalJSONUsesEnvelope(t *testing.T) {
	oldCfg, oldJSON := cfg, outputJSON
	cfg = pkgconfig.DefaultConfig()
	outputJSON = true
	t.Cleanup(func() { cfg, outputJSON = oldCfg, oldJSON })
	stdout := captureStdout(t, func() {
		if err := runConfigValidate(nil, nil); err != nil {
			t.Fatalf("runConfigValidate: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("config validate --json should output parseable JSON: %v\n%s", err, stdout)
	}
	if parsed["command"] != "config.validate" || parsed["mode"] != "json" {
		t.Fatalf("unexpected config validate envelope: %#v", parsed)
	}
}
