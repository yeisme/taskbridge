## ADDED Requirements

### Requirement: Action file execution SHALL be explicit, auditable, and reversible at the local evidence level
TaskBridge SHALL execute action files only through explicit dry-run or confirm modes and SHALL produce an audit receipt for every action execution attempt.

#### Scenario: dry-run action execution records preview without mutation
- **GIVEN** a valid `taskbridge.actions.v1` file contains a dangerous task action
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --dry-run`
- **THEN** stdout SHALL report dry-run status and per-action preview results
- **AND** no task SHALL be modified in local storage
- **AND** an audit receipt SHALL record the dry-run attempt, action count, skipped count, errors, and receipt id

#### Scenario: confirmed action execution writes and records audit receipt
- **GIVEN** a valid action file contains `complete_task`, `defer_task`, `reschedule_task`, or `split_task`
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --confirm`
- **THEN** TaskBridge SHALL apply supported local task mutations through the task store
- **AND** stdout SHALL include the audit receipt id or evidence path
- **AND** the audit receipt SHALL contain command name, dry_run=false, confirm=true, action ids, task ids, status, stats, errors, and timestamps
- **AND** the receipt SHALL NOT contain Provider tokens, Authorization headers, raw prompts, hidden system prompts, or unredacted Provider payloads

#### Scenario: failed confirmed execution does not claim success
- **GIVEN** a task store write fails during confirmed action execution
- **WHEN** the operator runs `taskbridge review --apply-file actions.json --confirm --json`
- **THEN** the command SHALL return a failed or partial machine result
- **AND** the process exit code SHALL be non-zero for failed execution
- **AND** the audit receipt SHALL record the failed action and error
- **AND** success counters SHALL include only actions that were actually persisted

### Requirement: Dangerous actions SHALL require confirmation
TaskBridge SHALL treat destructive or state-changing actions as dangerous and SHALL refuse to execute them without explicit confirmation.

#### Scenario: dangerous action without confirm requests confirmation
- **GIVEN** an action file contains `complete_task`, `delete_task`, `defer_task`, `reschedule_task`, `remote_write`, or `conflict_discard`
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --dry-run=false`
- **THEN** TaskBridge SHALL NOT mutate tasks unless `--confirm` is present
- **AND** the result SHALL include `requires_confirmation=true`
- **AND** the command SHALL expose a runnable next action using `taskbridge agent execute --action-file actions.json --confirm`

#### Scenario: unsupported action type fails loudly
- **GIVEN** an action file contains an unsupported action type
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --confirm`
- **THEN** the command SHALL return a failed or partial result naming the unsupported action id and type
- **AND** no unsupported action SHALL be silently ignored

### Requirement: Audit receipts SHALL be CLI-authored structured assets
TaskBridge SHALL create and update action audit receipts through application services, not by requiring users or agents to hand-write metadata files.

#### Scenario: receipt is written by TaskBridge
- **WHEN** `taskbridge agent execute` or `taskbridge review --apply-file` runs
- **THEN** TaskBridge SHALL create the receipt under the configured TaskBridge storage/audit location
- **AND** the receipt SHALL include `schema_version`, `session_id`, `command`, `status`, `started_at`, `finished_at`, `duration_ms`, `stats`, `operations`, `errors`, and `redaction`
- **AND** users and agents SHALL inspect receipts through TaskBridge commands or documented read-only paths, not by editing receipt JSON directly
