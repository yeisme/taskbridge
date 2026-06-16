## 1. RED tests

- [x] 1.1 Add facade tests for Load → Execute → outcomes → receipt. Acceptance: a single result exposes per-action outcomes and receipt data without command-specific reconstruction.
- [x] 1.2 Add command tests for `agent execute` and `review --apply-file` sharing the facade. Acceptance: both commands produce consistent per-action status, stats, audit operations, and failure behavior for the same action file.
- [x] 1.3 Add or update confirmed `project adjust` tests where the command executes action files or generated actions. Acceptance: confirmed execution uses the same facade rather than command-local mutation loops.
- [x] 1.4 Add failure tests for partial and failed confirmed execution. Acceptance: machine-readable output is emitted where requested, the process exits non-zero, and success counters count only persisted actions.
- [x] 1.5 Add dry-run no-mutation tests. Acceptance: task storage is unchanged; audit receipt write is allowed and marked as dry-run evidence.

## 2. Implementation

- [x] 2.1 Introduce or consolidate an internal action execution facade. Acceptance: command entry points pass options and render results; business logic stays outside `cmd/`.
- [x] 2.2 Define per-action outcome fields: `action_id`, `type`, `task_id`, `project_id`, `status`, `error`, and `reason` where applicable. Acceptance: command output and audit receipt operations use these outcomes directly.
- [x] 2.3 Derive audit receipt operations from facade outcomes. Acceptance: receipts are not reconstructed by positional guesses or duplicate command-specific heuristics.
- [x] 2.4 Normalize exit status for confirmed failures. Acceptance: failed or partial confirmed execution returns non-zero after selected machine output is written safely.
- [x] 2.5 Preserve dry-run behavior. Acceptance: preview/dry-run does not persist task mutations, while audit receipt writing remains allowed as execution evidence.

## 3. Verification

- [x] 3.1 Run focused facade and command tests only. Evidence: `go test ./cmd ./internal/actionfile ./internal/actionaudit ./internal/actionexecution -run 'Action|Execute|Audit|ReviewApply|ProjectAdjust|Facade' -count=1` passed.
- [x] 3.2 Validate this OpenSpec change. Evidence: `openspec validate action-execution-facade --strict` passed.

## Acceptance

- `agent execute`, `review --apply-file`, and confirmed `project adjust` use one internal action execution facade.
- Per-action outcomes are the source of truth for command output, stats, audit receipt operations, and exit behavior.
- Confirmed failures return non-zero consistently after machine-readable output where applicable.
- Dry-run/preview never persists task mutations; audit receipt writes remain allowed evidence.
