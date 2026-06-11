package cmd

import (
	"encoding/json"
	"testing"

	"github.com/yeisme/taskbridge/internal/clioutput"
)

func TestBuildVersionProjection(t *testing.T) {
	p := buildVersionProjection()
	if p.Command != "version.show" || p.Status != clioutput.StatusSuccess {
		t.Fatalf("unexpected version projection: %#v", p)
	}
	if p.Summary == "" {
		t.Fatal("version projection should have human summary")
	}
	if p.Facts["version"] == nil || p.Facts["go_version"] == nil {
		t.Fatalf("version facts missing: %#v", p.Facts)
	}
}

func TestVersionProjectionJSONEnvelope(t *testing.T) {
	out, err := clioutput.RenderJSON(buildVersionProjection())
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("version JSON is not parseable: %v", err)
	}
	if envelope["mode"] != "json" || envelope["command"] != "version.show" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
}
