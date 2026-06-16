## Why

TaskBridge sync currently includes paths that can translate local state into remote Provider writes. Dry-run, delete, conflict, and overwrite behavior must be specified before implementation expands remote write coverage, because silent remote writes or ambiguous counters would break the project safety rules.

## What Changes

- Add the `sync-write-safety` capability for sync push write gates and accounting.
- Require `sync push --dry-run` to avoid remote write APIs while still reporting planned operations.
- Require explicit confirmation before remote deletes and remote overwrites.
- Require output to distinguish planned writes from actual writes through the existing projection helpers.

## Non-Goals

- Do not add new Provider implementations.
- Do not implement bidirectional automatic conflict resolution.
- Do not change release distribution, package manager, or installer behavior.
- Do not bypass the Provider interface.

## Impact

- Main code areas: `cmd/sync.go`, `cmd/sync_control.go`, `internal/sync/`, Provider fake/test implementations.
- Main tests: focused sync command/engine tests for dry-run write suppression, delete confirmation, overwrite confirmation, and counters.
- Related specs: `taskbridge-cli-output-experience` for projection-based English output; `task-action-execution-audit` only if future remote writes receive audit receipts in a separate change.
