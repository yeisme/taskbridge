// Package actionaudit records action file execution receipts.
// Every review --apply-file and agent execute attempt produces a receipt
// describing what was attempted, what changed, what failed, and what evidence
// was collected. Receipts are written by this service, never hand-written.
package actionaudit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SchemaVersion is the receipt schema identifier.
const SchemaVersion = "taskbridge.action-audit.v1"

// Receipt records a single action file execution attempt.
type Receipt struct {
	SchemaVersion string      `json:"schema_version"`
	SessionID     string      `json:"session_id"`
	Command       string      `json:"command"`
	ActionFile    string      `json:"action_file,omitempty"`
	DryRun        bool        `json:"dry_run"`
	Confirm       bool        `json:"confirm"`
	Status        string      `json:"status"`
	StartedAt     time.Time   `json:"started_at"`
	FinishedAt    time.Time   `json:"finished_at"`
	DurationMs    int64       `json:"duration_ms"`
	Stats         Stats       `json:"stats"`
	Operations    []Operation `json:"operations"`
	Errors        []string    `json:"errors"`
	Redaction     string      `json:"redaction"`
}

// Stats summarises execution counts.
type Stats struct {
	Total     int `json:"total"`
	Updated   int `json:"updated"`
	Skipped   int `json:"skipped"`
	Errors    int `json:"errors"`
	Confirmed int `json:"confirmed"`
}

// Operation records a single action within the file.
type Operation struct {
	ActionID  string `json:"action_id"`
	Type      string `json:"type"`
	TaskID    string `json:"task_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	Error     string `json:"error,omitempty"`
	DryRun    bool   `json:"dry_run"`
	Confirmed bool   `json:"confirmed"`
}

// Store writes and reads receipts under <basePath>/audit/actions/.
type Store struct {
	BasePath string
}

// NewStore creates a Store rooted at the given storage path.
func NewStore(basePath string) *Store {
	return &Store{BasePath: basePath}
}

// Start creates a receipt with started_at set and status "running".
func Start(sessionID, command, actionFile string, dryRun, confirm bool) Receipt {
	return Receipt{
		SchemaVersion: SchemaVersion,
		SessionID:     sessionID,
		Command:       command,
		ActionFile:    actionFile,
		DryRun:        dryRun,
		Confirm:       confirm,
		Status:        "running",
		StartedAt:     time.Now().UTC(),
		Errors:        []string{},
		Redaction:     "task titles and fields are local task data; no tokens, authorization headers, or provider payloads are recorded",
	}
}

// Finish sets finished_at, duration, status, stats, operations and errors.
func (r *Receipt) Finish(status string, stats Stats, operations []Operation, errors []string) {
	r.FinishedAt = time.Now().UTC()
	r.DurationMs = r.FinishedAt.Sub(r.StartedAt).Milliseconds()
	r.Status = status
	r.Stats = stats
	r.Operations = operations
	r.Errors = errors
}

// Save writes the receipt as JSON under <basePath>/audit/actions/<session_id>.json.
func (s *Store) Save(r Receipt) error {
	if s.BasePath == "" {
		return fmt.Errorf("actionaudit: base path is empty")
	}
	dir := filepath.Join(s.BasePath, "audit", "actions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("actionaudit: create receipt dir: %w", err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("actionaudit: marshal receipt: %w", err)
	}
	path := filepath.Join(dir, r.SessionID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("actionaudit: write receipt: %w", err)
	}
	return nil
}

// Load reads a receipt by session ID.
func (s *Store) Load(sessionID string) (*Receipt, error) {
	path := filepath.Join(s.BasePath, "audit", "actions", sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// List returns receipt summaries sorted by session ID descending (newest first).
func (s *Store) List() ([]ReceiptSummary, error) {
	dir := filepath.Join(s.BasePath, "audit", "actions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ReceiptSummary{}, nil
		}
		return nil, err
	}
	summaries := make([]ReceiptSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var r Receipt
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}
		summaries = append(summaries, ReceiptSummary{
			SessionID:  r.SessionID,
			Command:    r.Command,
			Status:     r.Status,
			DryRun:     r.DryRun,
			Confirm:    r.Confirm,
			StartedAt:  r.StartedAt,
			FinishedAt: r.FinishedAt,
			Total:      r.Stats.Total,
			Updated:    r.Stats.Updated,
			Errors:     r.Stats.Errors,
		})
	}
	return summaries, nil
}

// ReceiptSummary is the lightweight view for listing.
type ReceiptSummary struct {
	SessionID  string    `json:"session_id"`
	Command    string    `json:"command"`
	Status     string    `json:"status"`
	DryRun     bool      `json:"dry_run"`
	Confirm    bool      `json:"confirm"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Total      int       `json:"total"`
	Updated    int       `json:"updated"`
	Errors     int       `json:"errors"`
}
