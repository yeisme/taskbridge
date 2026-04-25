package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Envelope struct {
	Schema  string          `json:"schema"`
	Version int             `json:"version"`
	Data    json.RawMessage `json:"data"`
}

func WriteEnvelopeAtomic(path, schema string, version int, payload interface{}) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	env := Envelope{Schema: schema, Version: version, Data: raw}
	return WriteAtomic(path, env)
}

func WriteAtomic(path string, value interface{}) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	return WriteBytesAtomic(path, raw)
}

func WriteBytesAtomic(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func ReadEnvelopeOrLegacy(data []byte, schema string, out interface{}) (int, bool, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err == nil && env.Schema != "" && len(env.Data) > 0 {
		if schema != "" && env.Schema != schema {
			return 0, false, fmt.Errorf("unexpected schema %q", env.Schema)
		}
		if err := json.Unmarshal(env.Data, out); err != nil {
			return 0, false, fmt.Errorf("unmarshal envelope data: %w", err)
		}
		return env.Version, false, nil
	}

	if err := json.Unmarshal(data, out); err != nil {
		return 0, true, fmt.Errorf("unmarshal legacy payload: %w", err)
	}
	return 0, true, nil
}
