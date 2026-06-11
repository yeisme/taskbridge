package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeisme/taskbridge/pkg/config"
	"github.com/yeisme/taskbridge/pkg/paths"
)

func TestAuthLoginMissingCredentialsReturnError(t *testing.T) {
	t.Setenv("TASKBRIDGE_HOME", t.TempDir())

	for name, fn := range map[string]func() error{
		"microsoft": loginMicrosoft,
		"feishu":    loginFeishu,
	} {
		t.Run(name, func(t *testing.T) {
			stdout := captureStdout(t, func() {
				if err := fn(); err == nil {
					t.Fatalf("expected missing credentials error")
				}
			})
			if stdout == "" {
				t.Fatalf("expected setup guidance on stdout")
			}
		})
	}
}

func TestAuthRefreshBrokenStaticTokenStoreReturnsError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("TASKBRIDGE_HOME", home)
	if err := paths.EnsureCredentialsDir(); err != nil {
		t.Fatalf("EnsureCredentialsDir: %v", err)
	}
	tokenPath := filepath.Join(paths.GetCredentialsDir(), paths.TokenFileName)
	if err := os.WriteFile(tokenPath, []byte(`{"version":1,"providers":{`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := refreshTickStyleProvider("ticktick"); err == nil {
			t.Fatalf("expected broken token store error")
		}
	})
	if stdout == "" {
		t.Fatalf("expected token read error on stdout")
	}
}

func TestRenderAuthStatusUsesEnglishFeishuAndStatusIcons(t *testing.T) {
	t.Setenv("TASKBRIDGE_HOME", t.TempDir())
	oldCfg := cfg
	cfg = config.DefaultConfig()
	defer func() { cfg = oldCfg }()

	out := renderAuthStatus(buildAuthStatusProjection())
	if !strings.Contains(out, "Feishu Tasks") {
		t.Fatalf("auth status should use English Feishu display name:\n%s", out)
	}
	if strings.Contains(out, "飞书任务") || strings.Contains(out, "Feishu mission") {
		t.Fatalf("auth status should not use old Feishu labels:\n%s", out)
	}
	if !strings.Contains(out, "⚪ Not configured") {
		t.Fatalf("auth status should decorate status with an icon:\n%s", out)
	}
}

func TestAuthLoginRejectsMachineModes(t *testing.T) {
	oldJSON := outputJSON
	outputJSON = true
	t.Cleanup(func() { outputJSON = oldJSON })
	if err := runAuthLogin(nil, []string{"google"}); err == nil || !strings.Contains(err.Error(), "--json") {
		t.Fatalf("auth login should reject machine modes before interactive output, got %v", err)
	}
}

func TestAuthRefreshRejectsMachineModes(t *testing.T) {
	oldAgent := outputAgent
	outputAgent = true
	t.Cleanup(func() { outputAgent = oldAgent })
	if err := runAuthRefresh(nil, []string{"google"}); err == nil || !strings.Contains(err.Error(), "--agent") {
		t.Fatalf("auth refresh should reject machine modes before progress output, got %v", err)
	}
}
