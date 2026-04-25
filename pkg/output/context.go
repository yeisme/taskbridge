package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ValidFields returns the list of accepted field names for --fields.
var ValidFields = []string{
	"id", "title", "status", "priority", "quadrant",
	"due_date", "source", "list_name", "tags", "progress", "description",
}

var validFieldSet map[string]bool

func init() {
	validFieldSet = make(map[string]bool, len(ValidFields))
	for _, f := range ValidFields {
		validFieldSet[f] = true
	}
}

// OutputContext captures how CLI output should be rendered.
type OutputContext struct {
	Writer  io.Writer
	IsPipe  bool
	IsAI    bool
	IsQuiet bool
	Format  string
	Fields  []string
	Limit   int
	Offset  int
}

// NewOutputContext detects the output environment and returns an OutputContext.
// Parameters are the user's explicit CLI flag values (empty/zero means "not set").
func NewOutputContext(format string, fields []string, limit, offset int, quiet bool) *OutputContext {
	isPipe := !isTerminal(os.Stdout)
	isAI := os.Getenv("TASKBRIDGE_AI_MODE") != ""

	// Determine format
	effectiveFormat := format
	if effectiveFormat == "" {
		if isPipe || isAI {
			effectiveFormat = "compact"
		} else {
			effectiveFormat = "table"
		}
	}

	// Determine limit
	effectiveLimit := limit
	if limit == 0 && (isPipe || isAI) {
		effectiveLimit = 50
	}

	return &OutputContext{
		Writer:  os.Stdout,
		IsPipe:  isPipe,
		IsAI:    isAI,
		IsQuiet: quiet || isPipe || isAI,
		Format:  effectiveFormat,
		Fields:  fields,
		Limit:   effectiveLimit,
		Offset:  offset,
	}
}

// ParseFields parses and validates a comma-separated field list.
func ParseFields(input string) ([]string, error) {
	if input == "" {
		return nil, nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !validFieldSet[p] {
			return nil, fmt.Errorf("unknown field %q; valid fields: %s", p, strings.Join(ValidFields, ", "))
		}
		result = append(result, p)
	}
	return result, nil
}

// isTerminal returns true if the file is a character device (TTY).
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
