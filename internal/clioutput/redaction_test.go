package clioutput

import (
	"strings"
	"testing"
)

func TestRedactValueRedactsSensitiveMapKeys(t *testing.T) {
	input := map[string]any{
		"token":  "abc123",
		"nested": map[string]any{"Authorization": "Bearer secret"},
		"name":   "visible",
	}

	redacted := RedactValue(input).(map[string]any)
	if redacted["token"] != RedactedValue {
		t.Fatalf("token = %v, want redacted", redacted["token"])
	}
	if redacted["nested"].(map[string]any)["Authorization"] != RedactedValue {
		t.Fatalf("nested authorization was not redacted: %#v", redacted["nested"])
	}
	if redacted["name"] != "visible" {
		t.Fatalf("name = %v, want visible", redacted["name"])
	}
}

func TestRenderAgentRedactsSensitiveFacts(t *testing.T) {
	p := New("auth.status")
	p.Facts["token"] = "abc123"
	p.Facts["state"] = "configured"

	out := RenderAgent(p)
	if strings.Contains(out, "abc123") {
		t.Fatalf("RenderAgent leaked sensitive fact:\n%s", out)
	}
	if !strings.Contains(out, "fact.token="+RedactedValue) {
		t.Fatalf("RenderAgent did not include redacted token:\n%s", out)
	}
}
