package clioutput

import (
	"encoding/json"
	"strings"
	"testing"
)

type redactionConfigFixture struct {
	Provider redactionProviderFixture `json:"provider"`
}

type redactionProviderFixture struct {
	ClientSecret string `json:"client_secret"`
	APIToken     string `json:"api_token"`
	APIKey       string `json:"api_key"`
	Name         string `json:"name"`
}

func TestRenderJSONRedactsSensitiveStructFields(t *testing.T) {
	p := New("config.show")
	p.Data = redactionConfigFixture{Provider: redactionProviderFixture{ClientSecret: "s3cr3t", APIToken: "t0k3n", APIKey: "k3y", Name: "visible"}}
	out, err := RenderJSON(p)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	if strings.Contains(string(out), "s3cr3t") || strings.Contains(string(out), "t0k3n") || strings.Contains(string(out), "k3y") {
		t.Fatalf("RenderJSON leaked sensitive struct fields:\n%s", out)
	}
	var envelope map[string]any
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("RenderJSON output should remain parseable: %v", err)
	}
	if !strings.Contains(string(out), "visible") {
		t.Fatalf("RenderJSON should preserve non-sensitive fields:\n%s", out)
	}
}
