package clioutput

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func RenderJSON(p Projection) ([]byte, error) {
	p.Mode = ModeJSON
	if p.SpecVersion == "" {
		p.SpecVersion = SpecVersion
	}
	if p.Status == "" {
		p.Status = StatusSuccess
	}
	if strings.TrimSpace(p.Command) == "" {
		p.Command = "unknown"
	}
	p = RedactProjection(p)
	return json.MarshalIndent(p, "", "  ")
}

func RenderAgent(p Projection) string {
	p.Mode = ModeAgent
	if p.SpecVersion == "" {
		p.SpecVersion = SpecVersion
	}
	if p.Status == "" {
		p.Status = StatusSuccess
	}
	if strings.TrimSpace(p.Command) == "" {
		p.Command = "unknown"
	}
	p = RedactProjection(p)

	var b strings.Builder
	writeAgentLine(&b, "spec_version", p.SpecVersion)
	writeAgentLine(&b, "mode", string(p.Mode))
	writeAgentLine(&b, "command", p.Command)
	writeAgentLine(&b, "status", string(p.Status))

	factKeys := make([]string, 0, len(p.Facts))
	for key := range p.Facts {
		factKeys = append(factKeys, key)
	}
	sort.Strings(factKeys)
	for _, key := range factKeys {
		writeAgentLine(&b, "fact."+sanitizeAgentKey(key), p.Facts[key])
	}

	actions := append([]Action(nil), p.Actions...)
	sort.SliceStable(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	for _, action := range actions {
		if strings.TrimSpace(action.Name) == "" || strings.TrimSpace(action.Command) == "" {
			continue
		}
		writeAgentLine(&b, "action."+sanitizeAgentKey(action.Name), action.Command)
	}

	for i, evidence := range p.Evidence {
		if strings.TrimSpace(evidence) == "" {
			continue
		}
		writeAgentLine(&b, fmt.Sprintf("evidence.%d", i), evidence)
	}
	return b.String()
}

func writeAgentLine(b *strings.Builder, key string, value any) {
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(formatAgentValue(value))
	b.WriteByte('\n')
}

func formatAgentValue(value any) string {
	s := fmt.Sprint(value)
	if s == "" {
		return strconv.Quote(s)
	}
	if strings.ContainsAny(s, " \t\n\r\"'`$\\") {
		return strconv.Quote(s)
	}
	return s
}

func sanitizeAgentKey(key string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(key) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}
