package clioutput

import (
	"encoding/json"
	"reflect"
	"strings"
)

const RedactedValue = "[REDACTED]"

func RedactProjection(p Projection) Projection {
	p.Facts = redactMap(p.Facts)
	p.Data = RedactValue(p.Data)
	if p.Error != nil {
		errCopy := *p.Error
		errCopy.Details = redactMap(p.Error.Details)
		p.Error = &errCopy
	}
	return p
}

func RedactValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return redactMap(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = RedactValue(item)
		}
		return out
	case []string:
		out := make([]string, len(v))
		copy(out, v)
		return out
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			return value
		}
		switch rv.Kind() {
		case reflect.Struct, reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
			data, err := json.Marshal(value)
			if err != nil {
				return value
			}
			var decoded any
			if err := json.Unmarshal(data, &decoded); err != nil {
				return value
			}
			return RedactValue(decoded)
		default:
			return value
		}
	}
}

func redactMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if isSensitiveKey(key) {
			out[key] = RedactedValue
			continue
		}
		out[key] = RedactValue(value)
	}
	return out
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, needle := range []string{"token", "secret", "password", "authorization", "cookie", "prompt", "provider_payload", "private_tool", "api_key", "apikey"} {
		if strings.Contains(k, needle) {
			return true
		}
	}
	return false
}
