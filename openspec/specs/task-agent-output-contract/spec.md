# task-agent-output-contract Specification

## Purpose
定义 TaskBridge 控制面和 agent 命令的机器可读输出、错误 exit code、能力声明和 schema 索引要求，确保 agent 只依赖稳定协议而不是人类摘要。
## Requirements
### Requirement: Control-plane commands SHALL support explicit machine output modes
TaskBridge SHALL render control-plane command results from one projection into human summary, `--json`, `--agent`, and legacy `--format json` modes, including cross-provider source/domain metadata needed by agents and scripts.

#### Scenario: json mode emits one envelope object
- **WHEN** the operator runs `taskbridge today --json`
- **THEN** stdout SHALL contain exactly one JSON object
- **AND** the object SHALL include `spec_version`, `mode`, `command`, `status`, and command-specific data
- **AND** cross-provider task entries SHALL expose stable source and domain metadata when present
- **AND** stderr SHALL contain diagnostics only, never JSON payload fragments required for parsing

#### Scenario: agent mode emits low-token key value facts
- **WHEN** the operator runs `taskbridge next --agent`
- **THEN** stdout SHALL contain stable `key=value` lines
- **AND** it SHALL include `spec_version`, `mode=agent`, `command`, `status`, and bounded recommendation facts
- **AND** it SHALL include enough source/domain context for an agent to explain why a recommendation spans work or life
- **AND** it SHALL NOT include ANSI, tables, localized prose paragraphs, raw prompts, Provider payloads, or debug dumps

### Requirement: Agent commands SHALL keep machine stdout and correct exit status
TaskBridge agent commands SHALL always return machine-readable stdout and SHALL use exit codes that reflect success, confirmation-required, partial, or failed execution consistently.

#### Scenario: agent error returns JSON and non-zero exit
- **GIVEN** the storage path is invalid
- **WHEN** the operator runs `taskbridge agent today`
- **THEN** stdout SHALL contain a valid `taskbridge.agent-result.v1` error object or compatible JSON envelope
- **AND** the process exit code SHALL be non-zero
- **AND** stderr SHALL contain only diagnostics safe for humans and logs

#### Scenario: confirmation required is machine visible
- **GIVEN** an action file contains a dangerous action and no `--confirm` is supplied
- **WHEN** the operator runs `taskbridge agent execute --action-file actions.json --dry-run=false`
- **THEN** stdout SHALL include `requires_confirmation=true`
- **AND** the result SHALL include the dry-run/effective execution mode
- **AND** no task mutation SHALL occur

### Requirement: Agent capabilities and schemas SHALL reflect implemented behavior
TaskBridge SHALL expose accurate capabilities and schema information for agents and scripts.

#### Scenario: capabilities list real commands and safety rules
- **WHEN** the operator runs `taskbridge agent capabilities`
- **THEN** stdout SHALL contain valid JSON
- **AND** the result SHALL list only implemented agent commands, supported schema versions, output modes, dangerous action types, and audit support flags
- **AND** it SHALL NOT advertise unavailable remote writes, MCP adapter behavior, or schema content that cannot be validated

#### Scenario: schemas output is validator friendly
- **WHEN** the operator runs `taskbridge agent schemas --json`
- **THEN** stdout SHALL contain a machine-readable schema index
- **AND** each schema entry SHALL include id, version, location or inline schema content, and compatibility status
- **AND** the result SHALL be sufficient for tests to validate `taskbridge.agent-result.v1`, `taskbridge.actions.v1`, and control-plane JSON envelopes

