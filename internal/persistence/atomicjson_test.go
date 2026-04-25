package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteEnvelopeAtomicAndReadEnvelopeOrLegacy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	payload := map[string]interface{}{"name": "demo", "count": 2}

	if err := WriteEnvelopeAtomic(path, "test.schema", 1, payload); err != nil {
		t.Fatalf("WriteEnvelopeAtomic: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.Schema != "test.schema" || env.Version != 1 {
		t.Fatalf("unexpected envelope metadata: %+v", env)
	}

	var decoded map[string]interface{}
	version, legacy, err := ReadEnvelopeOrLegacy(data, "test.schema", &decoded)
	if err != nil {
		t.Fatalf("ReadEnvelopeOrLegacy: %v", err)
	}
	if legacy || version != 1 {
		t.Fatalf("expected non-legacy version 1, got legacy=%v version=%d", legacy, version)
	}
}

func TestReadEnvelopeOrLegacyLegacyPayload(t *testing.T) {
	raw := []byte(`{"name":"legacy"}`)
	var decoded map[string]string
	version, legacy, err := ReadEnvelopeOrLegacy(raw, "", &decoded)
	if err != nil {
		t.Fatalf("ReadEnvelopeOrLegacy legacy: %v", err)
	}
	if !legacy || version != 0 {
		t.Fatalf("expected legacy version 0, got legacy=%v version=%d", legacy, version)
	}
	if decoded["name"] != "legacy" {
		t.Fatalf("unexpected decoded payload: %+v", decoded)
	}
}
