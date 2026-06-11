package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	pkgconfig "github.com/yeisme/taskbridge/pkg/config"
)

func TestBuildProviderListProjection(t *testing.T) {
	oldCfg := cfg
	cfg = pkgconfig.DefaultConfig()
	defer func() { cfg = oldCfg }()

	p := buildProviderListProjection()
	if p.Command != "provider.list" || p.Facts["count"] == nil || p.Data == nil {
		t.Fatalf("unexpected provider list projection: %#v", p)
	}
}

func TestRenderProviderListUsesEnglishHint(t *testing.T) {
	oldCfg := cfg
	cfg = pkgconfig.DefaultConfig()
	defer func() { cfg = oldCfg }()

	out := renderProviderList(buildProviderListProjection())
	if !strings.Contains(out, "Provider") || !strings.Contains(out, "taskbridge provider info") {
		t.Fatalf("provider list output missing table/hint:\n%s", out)
	}
	if strings.Contains(out, "提示") {
		t.Fatalf("provider list output should use English CLI chrome:\n%s", out)
	}
}

func TestRenderProviderListUsesEnglishFeishuAndStatusIcons(t *testing.T) {
	t.Setenv("TASKBRIDGE_HOME", t.TempDir())
	oldCfg := cfg
	cfg = pkgconfig.DefaultConfig()
	defer func() { cfg = oldCfg }()

	out := renderProviderList(buildProviderListProjection())
	if !strings.Contains(out, "Feishu Tasks") {
		t.Fatalf("provider list should use English Feishu display name:\n%s", out)
	}
	if strings.Contains(out, "飞书任务") || strings.Contains(out, "Feishu mission") {
		t.Fatalf("provider list should not use old Feishu labels:\n%s", out)
	}
	if !strings.Contains(out, "⚪ Disabled") {
		t.Fatalf("provider list should decorate status with an icon:\n%s", out)
	}
}

func TestStatusWithIcon(t *testing.T) {
	cases := map[string]string{
		"Connected":         "✅ Connected",
		"Expired":           "⚠️ Expired",
		"Not authenticated": "❌ Not authenticated",
		"Enabled":           "🟢 Enabled",
		"Disabled":          "⚪ Disabled",
		"Not configured":    "⚪ Not configured",
	}
	for input, want := range cases {
		if got := statusWithIcon(input); got != want {
			t.Fatalf("statusWithIcon(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildProviderInfoProjection(t *testing.T) {
	oldCfg := cfg
	cfg = pkgconfig.DefaultConfig()
	defer func() { cfg = oldCfg }()

	p := buildProviderInfoProjection("google")
	if p.Command != "provider.info" || p.Facts["provider"] != "google" || p.Data == nil {
		t.Fatalf("unexpected provider info projection: %#v", p)
	}
}

func TestBuildProviderTestProjection(t *testing.T) {
	oldCfg := cfg
	cfg = pkgconfig.DefaultConfig()
	defer func() { cfg = oldCfg }()

	p := buildProviderTestProjection("google")
	if p.Command != "provider.test" || p.Facts["provider"] != "google" {
		t.Fatalf("unexpected provider test projection: %#v", p)
	}
}

func TestProviderEnableUsesProjectionReceipt(t *testing.T) {
	oldCfg := cfg
	oldJSON, oldAgent := outputJSON, outputAgent
	cfg = pkgconfig.DefaultConfig()
	outputJSON, outputAgent = false, false
	defer func() { cfg = oldCfg; outputJSON, outputAgent = oldJSON, oldAgent }()

	out := captureStdout(t, func() {
		if err := runProviderEnable(nil, []string{"google"}); err != nil {
			t.Fatalf("runProviderEnable: %v", err)
		}
	})

	if !strings.Contains(out, "Status") || !strings.Contains(out, "✅") {
		t.Fatalf("provider enable should render a projection receipt with a status icon:\n%s", out)
	}
	if !strings.Contains(out, "taskbridge auth login google") || !strings.Contains(out, "taskbridge provider test google") {
		t.Fatalf("provider enable receipt should include next steps:\n%s", out)
	}
	if strings.Contains(out, "Provider googleEnabled") {
		t.Fatalf("provider enable should not use the old manual printf receipt:\n%s", out)
	}
}

func TestProviderEnableJSONUsesProjectionEnvelope(t *testing.T) {
	oldCfg := cfg
	oldJSON, oldAgent := outputJSON, outputAgent
	cfg = pkgconfig.DefaultConfig()
	outputJSON, outputAgent = true, false
	defer func() { cfg = oldCfg; outputJSON, outputAgent = oldJSON, oldAgent }()

	out := captureStdout(t, func() {
		if err := runProviderEnable(nil, []string{"todo"}); err != nil {
			t.Fatalf("runProviderEnable: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("provider enable --json should be parseable JSON, got %v in %q", err, out)
	}
	if payload["command"] != "provider.enable" || payload["mode"] != "json" || payload["status"] != "success" {
		t.Fatalf("unexpected provider enable envelope: %+v", payload)
	}
}

func TestProviderDisableUsesProjectionReceipt(t *testing.T) {
	oldCfg := cfg
	oldJSON, oldAgent := outputJSON, outputAgent
	cfg = pkgconfig.DefaultConfig()
	cfg.Providers.Google.Enabled = true
	outputJSON, outputAgent = false, false
	defer func() { cfg = oldCfg; outputJSON, outputAgent = oldJSON, oldAgent }()

	out := captureStdout(t, func() {
		if err := runProviderDisable(nil, []string{"google"}); err != nil {
			t.Fatalf("runProviderDisable: %v", err)
		}
	})

	if !strings.Contains(out, "Status") || !strings.Contains(out, "⚪") {
		t.Fatalf("provider disable should render a projection receipt with a status icon:\n%s", out)
	}
	if !strings.Contains(out, "taskbridge provider enable google") {
		t.Fatalf("provider disable receipt should include a re-enable next step:\n%s", out)
	}
	if strings.Contains(out, "Provider googleDisabled") {
		t.Fatalf("provider disable should not use the old manual printf receipt:\n%s", out)
	}
}
