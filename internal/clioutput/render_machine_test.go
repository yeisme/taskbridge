package clioutput

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSONSetsModeAndEnvelope(t *testing.T) {
	p := New("provider.list")
	p.Summary = "Providers listed"
	p.Data = map[string]any{"count": 2}

	out, err := RenderJSON(p)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("RenderJSON output is not JSON: %v\n%s", err, out)
	}
	if got["spec_version"] != "1.0" || got["mode"] != "json" || got["command"] != "provider.list" || got["status"] != "success" {
		t.Fatalf("unexpected envelope: %#v", got)
	}
}

func TestRenderAgentOrdersFactsAndQuotesValues(t *testing.T) {
	p := New("auth.status")
	p.Facts["zeta"] = "last value"
	p.Facts["alpha"] = 2
	p.Actions = []Action{{Name: "login", Command: "taskbridge auth login google"}}
	p.Evidence = []string{"provider catalog"}

	out := RenderAgent(p)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	want := []string{
		"spec_version=1.0",
		"mode=agent",
		"command=auth.status",
		"status=success",
		"fact.alpha=2",
		"fact.zeta=\"last value\"",
		"action.login=\"taskbridge auth login google\"",
		"evidence.0=\"provider catalog\"",
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), len(want), out)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d = %q, want %q\nfull output:\n%s", i, lines[i], want[i], out)
		}
	}
}
