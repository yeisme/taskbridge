# task-agent-output-contract Specification Delta

## MODIFIED Requirements

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
