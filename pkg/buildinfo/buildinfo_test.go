package buildinfo

import "testing"

func TestResolveVersionUsesModuleVersionWhenLdflagsVersionIsDev(t *testing.T) {
	got := resolveVersion("dev", "v1.2.3")
	if got != "1.2.3" {
		t.Fatalf("resolveVersion() = %q, want 1.2.3", got)
	}
}

func TestResolveVersionPrefersInjectedLdflagsVersion(t *testing.T) {
	got := resolveVersion("1.2.3-4-gabc123", "v1.2.3")
	if got != "1.2.3-4-gabc123" {
		t.Fatalf("resolveVersion() = %q, want injected version", got)
	}
}
