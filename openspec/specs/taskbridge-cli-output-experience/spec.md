# taskbridge-cli-output-experience Specification

## Purpose
定义 TaskBridge 全 CLI tree 的统一 projection/renderer 输出体验，覆盖 human、JSON、agent、events、explain、颜色宽度、错误分流和脱敏边界。
## Requirements
### Requirement: The full CLI tree SHALL render human and machine output from one projection
TaskBridge SHALL build a command projection once for every migrated CLI subtree and render default human output, `--json`, `--agent`, `--events`, and `--explain` from that projection instead of parsing localized human text or hand-writing unrelated payloads per mode. The maintained tree includes root/help/error, analyze, list/lists, provider/auth, task, sync, doctor/quickstart, today/next/inbox/review, project, governance, config/version, and serve/runtime output.

#### Scenario: json mode emits one envelope object
- **WHEN** the operator runs `taskbridge provider list --json`
- **THEN** stdout SHALL contain exactly one JSON object
- **AND** the object SHALL include `spec_version`, `mode=json`, `command`, `status`, and command-specific `data`
- **AND** stdout SHALL NOT contain ANSI, table borders, progress text, logs, or human hints outside the JSON object

#### Scenario: agent mode emits stable key value facts
- **WHEN** the operator runs `taskbridge auth status --agent`
- **THEN** stdout SHALL contain newline-delimited `key=value` facts
- **AND** it SHALL include `spec_version`, `mode=agent`, `command`, and `status`
- **AND** it SHALL NOT contain ANSI, tables, localized prose paragraphs, raw prompts, Provider payloads, or debug dumps

#### Scenario: default output is human first
- **WHEN** the operator runs `taskbridge version`
- **THEN** stdout SHALL be a short human-readable summary or detail view
- **AND** it SHALL NOT be a raw JSON dump
- **AND** any recommended next command SHALL be a real user-runnable command

### Requirement: Human output SHALL use command-appropriate visual forms
TaskBridge SHALL choose default human output forms by command type so routine browsing remains table-first, analysis output remains stats-panel based, and decision or governance output remains section-based.

#### Scenario: analyze priority renders a stats panel
- **WHEN** the operator runs `taskbridge analyze priority` against an empty task store
- **THEN** stdout SHALL show `Prioritization analysis`
- **AND** stdout SHALL show one boxed stats panel containing `Urgent (P0)`, `High (P1)`, `Medium (P2)`, `Low (P3)`, and `No priority`
- **AND** each row SHALL include count and percentage values
- **AND** stdout SHALL include a footer with `Total`, `Active`, and `Completed`
- **AND** the same command with `--format json` SHALL preserve the legacy parseable JSON payload

#### Scenario: result list uses a focused table
- **WHEN** the operator runs `taskbridge provider list`
- **THEN** stdout SHALL show a compact provider table with aligned CJK and ASCII text
- **AND** it SHALL not show schema names, evidence refs, renderer metadata, or multiple next-step paragraphs

#### Scenario: governance output uses fixed English sections
- **WHEN** the operator runs `taskbridge auth status`
- **THEN** stdout SHALL use fixed sections such as `Status`, `Highlights`, `Facts`, `Preview`, `Risks`, and `Recommended next step` when those sections carry real information
- **AND** it SHALL show at most one primary recommended next command

#### Scenario: empty result gives one useful next step
- **WHEN** the operator runs `taskbridge list` and no tasks match
- **THEN** stdout SHALL explain the empty result in English
- **AND** it SHALL include at most one primary next command such as `taskbridge list --sync-now`
- **AND** machine modes SHALL expose the same empty-result fact without parsing the human text

### Requirement: Visual rendering SHALL preserve readability across width and color modes
TaskBridge SHALL render terminal output that remains readable with CJK text, long values, no color, dumb terminals, and machine output modes.

#### Scenario: CJK table columns stay aligned
- **GIVEN** provider names include `Google Tasks` and `飞书任务`
- **WHEN** the operator runs `taskbridge provider list`
- **THEN** the table SHALL align columns by display width rather than byte length
- **AND** long cell values SHALL be truncated or wrapped according to renderer rules without shifting later columns

#### Scenario: no color mode remains readable
- **WHEN** the operator runs `NO_COLOR=1 taskbridge auth status`
- **THEN** stdout SHALL remain readable and SHALL NOT rely on color alone to communicate success, failure, warning, authentication state, or risk

#### Scenario: machine modes never emit ANSI
- **WHEN** the operator runs `taskbridge provider list --color always --json`
- **THEN** stdout SHALL contain no ANSI escape sequences
- **AND** the JSON payload SHALL remain parseable

### Requirement: Compatibility modes SHALL be explicit and tested
TaskBridge SHALL preserve known legacy output paths while guiding new automation to explicit `--json` and `--agent` modes, and each CLI subtree migration SHALL document whether `--format json` is legacy payload or an envelope alias.

#### Scenario: legacy format json remains parseable
- **WHEN** the operator runs `taskbridge config show --format json`
- **THEN** stdout SHALL remain parseable as one JSON object
- **AND** TaskBridge SHALL NOT append human configuration-source text, hints, logs, or progress after the JSON payload

#### Scenario: explicit json is preferred for new scripts
- **WHEN** documentation shows a machine-readable example
- **THEN** it SHALL use `--json` for envelope output or `--agent` for key=value output
- **AND** it SHALL NOT rely on pipe detection or `--quiet` auto-machine behavior as the recommended path

#### Scenario: legacy format ai maps to agent where supported
- **GIVEN** a command supports a historical `--format ai` mode
- **WHEN** the operator runs that mode
- **THEN** it SHALL render from the same projection as `--agent`
- **AND** contract tests SHALL prove the fact keys remain stable

### Requirement: Error and sensitive output SHALL stay separated and redacted
TaskBridge SHALL keep machine stdout parseable on errors and SHALL redact sensitive values in every output mode and persisted evidence.

#### Scenario: invalid provider has stable error output
- **WHEN** the operator runs `taskbridge provider info unknown --json`
- **THEN** stdout SHALL contain a failed JSON envelope with stable `error.code` and an English `error.message`
- **AND** stderr SHALL contain only safe diagnostics
- **AND** the process exit code SHALL be non-zero

#### Scenario: sensitive values are redacted
- **GIVEN** a command result includes fields named `token`, `secret`, `password`, `authorization`, `cookie`, or Provider payload snippets
- **WHEN** TaskBridge renders human, json, agent, events, explain, sidecar, snapshot, golden, or integration evidence output
- **THEN** those values SHALL be redacted
- **AND** raw prompts, hidden system prompts, private tool arguments, unredacted Provider payloads, and full chain-of-thought SHALL NOT be printed or persisted

### Requirement: Output contract tests SHALL cover every migrated command family
TaskBridge SHALL add tests close to migrated commands so output regressions are caught before release.

#### Scenario: migrated command family has contract tests
- **WHEN** analyze, provider/auth, list/lists, task, sync, control-plane, project/governance, config/version, or serve output is migrated to the shared renderer
- **THEN** tests SHALL cover default human output, `--json`, `--agent`, error output, `NO_COLOR=1`, and color-forced machine mode when applicable
- **AND** tests SHALL parse machine stdout instead of asserting only substrings

#### Scenario: integration evidence records output contract runs
- **WHEN** `task test:integration` runs output contract cases
- **THEN** TaskBridge SHALL write redacted evidence under `temp/integration-test-runs/<run-id>/`
- **AND** the evidence SHALL include command, stdout, stderr, env, summary, artifacts, status, and original exit code

