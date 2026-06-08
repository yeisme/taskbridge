package cmd

import (
	"os"
	"path/filepath"
	"testing"

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
