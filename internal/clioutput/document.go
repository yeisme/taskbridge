package clioutput

import "strings"

const SpecVersion = "1.0"

type Status string

const (
	StatusSuccess Status = "success"
	StatusPartial Status = "partial"
	StatusFailed  Status = "failed"
)

type Mode string

const (
	ModeSummary Mode = "summary"
	ModeJSON    Mode = "json"
	ModeAgent   Mode = "agent"
	ModeEvents  Mode = "events"
	ModeExplain Mode = "explain"
)

type Projection struct {
	SpecVersion string         `json:"spec_version"`
	Mode        Mode           `json:"mode"`
	Command     string         `json:"command"`
	Status      Status         `json:"status"`
	Summary     string         `json:"summary,omitempty"`
	Facts       map[string]any `json:"facts,omitempty"`
	Actions     []Action       `json:"actions,omitempty"`
	Evidence    []string       `json:"evidence,omitempty"`
	Confidence  *float64       `json:"confidence,omitempty"`
	Data        any            `json:"data,omitempty"`
	Error       *OutputError   `json:"error,omitempty"`

	Preview []PreviewItem `json:"-"`
	Tables  []Table       `json:"-"`
	Stats   []StatPanel   `json:"-"`
	Risks   []string      `json:"-"`
	Hint    string        `json:"-"`
}

type Action struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type OutputError struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Suggestion string         `json:"suggestion,omitempty"`
	Retryable  bool           `json:"retryable,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}

type PreviewItem struct {
	Label string
	Value string
}

type Table struct {
	Columns []Column
	Rows    [][]string
}

type Column struct {
	Header string
	Width  int
	Right  bool
}

type StatPanel struct {
	Title  string
	Rows   []StatRow
	Footer string
}

type StatRow struct {
	Icon    string
	Label   string
	Value   string
	Percent string
	Hint    string
}

func New(command string) Projection {
	command = strings.TrimSpace(command)
	if command == "" {
		command = "unknown"
	}
	return Projection{
		SpecVersion: SpecVersion,
		Mode:        ModeSummary,
		Command:     command,
		Status:      StatusSuccess,
		Facts:       make(map[string]any),
	}
}
