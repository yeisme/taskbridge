## Why

The archived control-plane hardening work established action audit receipts, but follow-up implementation needs a stable boundary that prevents command handlers from reconstructing action results heuristically. `agent execute`, `review --apply-file`, and confirmed `project adjust` must share one execution facade so receipts, machine output, exit status, and per-action failures stay consistent.

## What Changes

- Modify `task-action-execution-audit` to require a shared action execution facade under `internal/`.
- Require the facade to own Load → Execute → per-action outcomes → audit receipt.
- Require command output and audit receipt operations to derive from the same per-action outcomes.
- Require confirmed failures to return non-zero after emitting machine-readable output where applicable.
- Preserve dry-run no-mutation semantics while allowing audit receipts as execution evidence.

## Non-Goals

- Do not add new action types unless required by existing command contracts.
- Do not add remote Provider writes or MCP adapter behavior.
- Do not change release distribution files.
- Do not move business logic into Cobra command handlers.

## Impact

- Main code areas: `cmd/agent.go`, `cmd/controlplane.go`, project adjust command entry points, `internal/actionfile/`, `internal/actionaudit/`, and a shared internal facade/service.
- Main tests: facade unit tests, command tests for agent/review/project adjust, audit receipt operation derivation, and exit-code behavior.
