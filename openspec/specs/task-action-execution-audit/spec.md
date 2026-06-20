# task-action-execution-audit Specification

## Purpose
Defines TaskBridge action file execution requirements for explicit dry-run/confirm gates, reproducible audit receipts, failure accounting, local write evidence, and reserved dangerous action handling so dangerous operations cannot silently change task state.
## Requirements
### Requirement: Action file execution SHALL be explicit, auditable, and reversible at the local evidence level
TaskBridge SHALL execute action files only through explicit dry-run or confirm modes and SHALL route action execution through a shared internal facade that produces per-action outcomes and an audit receipt for every action execution attempt.

#### Scenario: dry-run action execution records preview without mutation
- **GIVEN** a valid `taskbridge.actions.v1` file contains a dangerous task action
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --dry-run`
- **THEN** stdout SHALL report dry-run status and per-action preview results
- **AND** no task SHALL be modified in local storage
- **AND** an audit receipt SHALL record the dry-run attempt, action count, skipped count, errors, and receipt id

#### Scenario: confirmed action execution writes and records audit receipt
- **GIVEN** a valid action file contains `complete_task`, `defer_task`, `reschedule_task`, or `split_task`
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --confirm`
- **THEN** TaskBridge SHALL apply supported local task mutations through the shared action execution facade and task store
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

#### Scenario: shared facade owns action execution flow
- **GIVEN** `agent execute`, `review --apply-file`, or confirmed `project adjust` executes actions
- **WHEN** the command loads, previews, or confirms the action set
- **THEN** a shared internal action execution facade SHALL own Load, Execute, per-action outcomes, stats, and audit receipt creation
- **AND** Cobra command code SHALL NOT reconstruct failures, operations, or receipt metadata by positional guessing

#### Scenario: per-action outcomes are the source of truth
- **GIVEN** an action execution contains successful, skipped, failed, or unsupported actions
- **WHEN** TaskBridge renders command output or writes an audit receipt
- **THEN** each action outcome SHALL expose `action_id`, `type`, `task_id`, `project_id`, `status`, `error`, and `reason` where applicable
- **AND** audit receipt operations SHALL be derived from those outcomes
- **AND** command stats SHALL count only outcomes that actually reached the corresponding state

### Requirement: Dangerous actions SHALL require confirmation
TaskBridge SHALL treat destructive or state-changing actions as dangerous and SHALL refuse to execute them without explicit confirmation. Reserved dangerous action types SHALL fail loudly when unsupported and SHALL NOT be advertised as implemented capabilities.

#### Scenario: dangerous action without confirm requests confirmation
- **GIVEN** an action file contains `complete_task`, `delete_task`, `defer_task`, `reschedule_task`, `remote_write`, or `conflict_discard`
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --dry-run=false`
- **THEN** TaskBridge SHALL NOT mutate tasks unless `--confirm` is present
- **AND** the result SHALL include `requires_confirmation=true`
- **AND** the command SHALL expose a runnable next action using `taskbridge agent execute --action-file actions.json --confirm`

#### Scenario: reserved remote action types are not advertised as supported writes
- **GIVEN** `remote_write` or `conflict_discard` appears in schema, validation, audit, or dangerous-action documentation
- **WHEN** TaskBridge reports agent capabilities or executable action support
- **THEN** TaskBridge SHALL NOT advertise remote Provider write execution, MCP adapter behavior, or conflict-discard execution unless those behaviors are implemented by a separate accepted change
- **AND** unsupported reserved action types SHALL return a failed or partial result instead of silently executing or being ignored

#### Scenario: unsupported action type fails loudly
- **GIVEN** an action file contains an unsupported action type
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --confirm`
- **THEN** the command SHALL return a failed or partial result naming the unsupported action id and type
- **AND** no unsupported action SHALL be silently ignored

#### Scenario: confirmed execution failures return non-zero consistently
- **GIVEN** confirmed action execution fails for one or more actions
- **WHEN** the command renders human, `--json`, `--agent`, or legacy machine output
- **THEN** TaskBridge SHALL emit the selected output safely
- **AND** the process exit code SHALL be non-zero
- **AND** stderr SHALL contain only safe diagnostics when machine stdout is selected

### Requirement: Audit receipts SHALL be CLI-authored structured assets
TaskBridge SHALL create and update action audit receipts through application services, not by requiring users or agents to hand-write metadata files.

#### Scenario: receipt is written by TaskBridge
- **WHEN** `taskbridge agent execute`, `taskbridge review --apply-file`, or confirmed `taskbridge project adjust` runs
- **THEN** TaskBridge SHALL create the receipt under the configured TaskBridge storage/audit location
- **AND** the receipt SHALL include `schema_version`, `session_id`, `command`, `status`, `started_at`, `finished_at`, `duration_ms`, `stats`, `operations`, `errors`, and `redaction`
- **AND** users and agents SHALL inspect receipts through TaskBridge commands or documented read-only paths, not by editing receipt JSON directly

#### Scenario: dry-run receipt is allowed evidence without task mutation
- **GIVEN** an action execution runs in dry-run or preview mode
- **WHEN** TaskBridge writes the action audit receipt
- **THEN** the receipt SHALL be treated as execution evidence
- **AND** no task mutation SHALL be persisted because of the dry-run itself
