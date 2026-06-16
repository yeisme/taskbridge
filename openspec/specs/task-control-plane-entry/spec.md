# task-control-plane-entry Specification

## Purpose
定义 TaskBridge 本地任务执行控制面的无认证体验入口、每日决策命令和只读 review 行为，确保新用户和 agent 能在不触发远端写入的情况下理解下一步。
## Requirements
### Requirement: TaskBridge SHALL provide a no-auth control-plane demo path
TaskBridge SHALL let a new user see the daily control-plane value without authenticating a Provider, writing local tasks, or calling remote APIs.

#### Scenario: demo today works without provider credentials
- **GIVEN** no Provider credentials exist and the storage path is empty
- **WHEN** the operator runs `taskbridge demo today --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the payload SHALL describe a valid daily workbench with demo tasks, suggested sections, and next action metadata
- **AND** the command SHALL NOT create local tasks, read Provider tokens, or call remote Provider APIs

#### Scenario: doctor points to the fastest useful path
- **GIVEN** no Provider is authenticated
- **WHEN** the operator runs `taskbridge doctor --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the result SHALL include a runnable next action such as `taskbridge demo today`
- **AND** diagnostics SHALL NOT corrupt machine stdout

### Requirement: today and next SHALL be the primary daily control-plane commands
TaskBridge SHALL make `today` and `next` the concise daily entry points for deciding what to do, while keeping detailed list/search behavior in existing list commands.

#### Scenario: today summarizes decisions instead of dumping every task
- **GIVEN** the local store contains active, overdue, inbox, and project-linked tasks
- **WHEN** the operator runs `taskbridge today`
- **THEN** the default stdout SHALL be a short English summary with sections and one recommended next command
- **AND** it SHALL NOT be a raw JSON dump, full task table, debug log, or provider payload

#### Scenario: next returns a bounded recommendation set
- **GIVEN** the local store contains more than five active tasks
- **WHEN** the operator runs `taskbridge next --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the result SHALL include no more than the default recommendation limit unless `--limit` is set
- **AND** large tasks over the configured threshold SHALL be recommended for split/review rather than presented as direct next work

### Requirement: review SHALL propose actions without implicit writes
TaskBridge SHALL keep `review` read-only unless the operator explicitly supplies an action file and either `--dry-run` or `--confirm`.

#### Scenario: plain review does not mutate tasks
- **GIVEN** the local store contains overdue and oversized tasks
- **WHEN** the operator runs `taskbridge review --json`
- **THEN** stdout SHALL parse as one JSON object containing suggested actions
- **AND** task status, due date, metadata, and local storage files SHALL remain unchanged

#### Scenario: apply-file requires explicit execution mode
- **GIVEN** an action file exists
- **WHEN** the operator runs `taskbridge review --apply-file actions.json`
- **THEN** the command SHALL fail with a non-zero exit code
- **AND** stderr or the approved human output channel SHALL explain that `--dry-run` or `--confirm` is required
- **AND** no task SHALL be modified

