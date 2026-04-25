package output

import (
	"bytes"
	"os"
	"testing"
)

func TestNewOutputContext_TTYDefaults(t *testing.T) {
	// In a test environment, os.Stdout is typically a pipe, so we test
	// the explicit format override path for TTY-like behavior.
	ctx := NewOutputContext("table", nil, 0, 0, false)
	if ctx.Format != "table" {
		t.Errorf("expected Format table, got %s", ctx.Format)
	}
	if ctx.Writer != os.Stdout {
		t.Error("expected Writer to be os.Stdout")
	}
}

func TestNewOutputContext_PipeDefaults(t *testing.T) {
	// In test env, stdout is a pipe, so isPipe=true
	ctx := NewOutputContext("", nil, 0, 0, false)
	if ctx.Format != "compact" {
		t.Errorf("expected Format compact for pipe, got %s", ctx.Format)
	}
	if ctx.Limit != 50 {
		t.Errorf("expected Limit 50 for pipe, got %d", ctx.Limit)
	}
	if !ctx.IsPipe {
		t.Error("expected IsPipe true in test environment")
	}
	if !ctx.IsQuiet {
		t.Error("expected IsQuiet true in pipe mode")
	}
}

func TestNewOutputContext_ExplicitFormatOverrides(t *testing.T) {
	ctx := NewOutputContext("json", nil, 10, 5, true)
	if ctx.Format != "json" {
		t.Errorf("expected Format json, got %s", ctx.Format)
	}
	if ctx.Limit != 10 {
		t.Errorf("expected Limit 10, got %d", ctx.Limit)
	}
	if ctx.Offset != 5 {
		t.Errorf("expected Offset 5, got %d", ctx.Offset)
	}
	if !ctx.IsQuiet {
		t.Error("expected IsQuiet true when quiet flag set")
	}
}

func TestNewOutputContext_AIMode(t *testing.T) {
	os.Setenv("TASKBRIDGE_AI_MODE", "1")
	defer os.Unsetenv("TASKBRIDGE_AI_MODE")

	ctx := NewOutputContext("", nil, 0, 0, false)
	if !ctx.IsAI {
		t.Error("expected IsAI true when TASKBRIDGE_AI_MODE set")
	}
	if ctx.Format != "compact" {
		t.Errorf("expected Format compact in AI mode, got %s", ctx.Format)
	}
	if ctx.Limit != 50 {
		t.Errorf("expected Limit 50 in AI mode, got %d", ctx.Limit)
	}
}

func TestNewOutputContext_ExplicitLimitOverridesPipeDefault(t *testing.T) {
	// In test env (pipe), but explicit limit should not be overridden
	ctx := NewOutputContext("", nil, 20, 0, false)
	if ctx.Limit != 20 {
		t.Errorf("expected explicit Limit 20, got %d", ctx.Limit)
	}
}

func TestParseFields_Valid(t *testing.T) {
	result, err := ParseFields("id,title,status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(result))
	}
	expected := []string{"id", "title", "status"}
	for i, f := range expected {
		if result[i] != f {
			t.Errorf("field %d: expected %s, got %s", i, f, result[i])
		}
	}
}

func TestParseFields_WithSpaces(t *testing.T) {
	result, err := ParseFields(" id , title , status ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(result))
	}
}

func TestParseFields_InvalidField(t *testing.T) {
	_, err := ParseFields("id,bogus,title")
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

func TestParseFields_Empty(t *testing.T) {
	result, err := ParseFields("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestOutputContext_WriterIsTestable(t *testing.T) {
	var buf bytes.Buffer
	ctx := &OutputContext{
		Writer: &buf,
		Format: "table",
	}
	_, _ = ctx.Writer.Write([]byte("hello"))
	if buf.String() != "hello" {
		t.Errorf("expected Writer to be bytes.Buffer, got %q", buf.String())
	}
}
