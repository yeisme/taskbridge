## MODIFIED Requirements

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
