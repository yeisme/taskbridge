package clioutput

import "testing"

func TestNewProjectionDefaults(t *testing.T) {
	p := New("analyze.priority")

	if p.SpecVersion != "1.0" {
		t.Fatalf("SpecVersion = %q, want 1.0", p.SpecVersion)
	}
	if p.Command != "analyze.priority" {
		t.Fatalf("Command = %q, want analyze.priority", p.Command)
	}
	if p.Status != StatusSuccess {
		t.Fatalf("Status = %q, want %q", p.Status, StatusSuccess)
	}
	if p.Facts == nil {
		t.Fatal("Facts map should be initialized")
	}
}

func TestNewProjectionRejectsEmptyCommand(t *testing.T) {
	p := New("  ")

	if p.Command != "unknown" {
		t.Fatalf("Command = %q, want unknown", p.Command)
	}
}
