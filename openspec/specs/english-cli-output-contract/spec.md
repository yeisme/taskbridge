# english-cli-output-contract Specification

## Purpose
定义 TaskBridge 默认英文 CLI 输出、稳定机器协议、stdout/stderr 分离、脱敏和输出合同测试要求，确保人类界面可读且自动化无需解析本地化文本。
## Requirements
### Requirement: Taskbridge defaults to English user-visible output

Taskbridge SHALL render default CLI output, help text, usage text, validation errors, stderr diagnostics, operator logs, task descriptions, documentation command examples, and explain summaries in English by default.

#### Scenario: Default command output uses English operator labels
- **WHEN** a user runs a representative `taskbridge` command without a machine-output flag
- **THEN** stdout SHALL use English human-facing labels and prose
- **AND** the output SHALL include a concise current status and at most one primary recommended next command when a next step is useful
- **AND** the output SHALL NOT require agents or scripts to parse localized human text.

#### Scenario: Help and errors are English
- **WHEN** a user runs `taskbridge --help`, a command-specific help page, an invalid command, or a validation-failure path
- **THEN** help, usage, suggestions, and error explanations SHALL be English
- **AND** any suggested command SHALL be a real command a human can run.

#### Scenario: Domain content is not blindly translated
- **GIVEN** user-authored content, quoted source material, provider-returned content, fixture story text, or third-party payload fields are intentionally non-English
- **WHEN** the command renders that content as data
- **THEN** Taskbridge SHALL preserve the source content language
- **AND** English-only enforcement SHALL apply to CLI chrome, not to user or provider data.

### Requirement: Machine output remains stable and language-neutral

Taskbridge SHALL keep machine-readable output parseable, stable, redacted, and independent from human-language prose.

#### Scenario: JSON output uses the shared envelope
- **WHEN** a user runs a supported command with `--json`
- **THEN** stdout SHALL contain exactly one valid JSON object
- **AND** the object SHALL include `spec_version`, `mode=json`, `command`, and `status`
- **AND** logs, progress text, ANSI, banners, and human suggestions SHALL NOT be written to stdout.

#### Scenario: Agent output uses stable key=value lines
- **WHEN** a user runs a supported command with `--agent`
- **THEN** stdout SHALL contain stable ASCII key=value lines
- **AND** it SHALL include `spec_version`, `mode=agent`, `command`, and `status`
- **AND** it SHALL NOT include localized prose, ANSI, tables, raw prompts, hidden system prompts, provider payloads, private tool arguments, or chain-of-thought.

#### Scenario: Event streams are NDJSON only
- **WHEN** a user runs a long-running supported command with `--events`
- **THEN** stdout SHALL be newline-delimited JSON events starting with `start` and ending with `end` or `error`
- **AND** progress logs and diagnostics SHALL be written to stderr or structured events, not mixed prose on stdout.

#### Scenario: Explain mode is a redacted English review summary
- **WHEN** a user requests an explain or decision summary mode
- **THEN** Taskbridge SHALL output English sections for conclusion, evidence, confidence, risks, tradeoffs, and recommended next step where applicable
- **AND** it SHALL NOT output full chain-of-thought, hidden prompts, raw provider payloads, secrets, or private tool arguments.

### Requirement: Output contract tests cover language, stdout, stderr, and redaction

Taskbridge SHALL include automated tests that guard English user-visible output and stable machine-output behavior.

#### Scenario: Contract tests parse machine modes
- **WHEN** the focused output test suite runs
- **THEN** tests SHALL parse `--json` as JSON
- **AND** parse `--agent` as key=value lines
- **AND** parse `--events` as NDJSON for commands that support event streams.

#### Scenario: Contract tests reject forbidden leaks
- **WHEN** output, traces, sidecars, snapshots, fixtures, or integration evidence are generated
- **THEN** tests SHALL reject secrets, tokens, Authorization headers, cookies, raw prompts, hidden system prompts, unredacted provider payloads, private tool arguments, full chain-of-thought, ANSI in machine stdout, and localized CLI chrome in machine modes.

#### Scenario: Integration evidence is project-owned
- **WHEN** integration, component, system, or e2e tests are changed for this output contract
- **THEN** evidence SHALL be written by the project runner under `temp/integration-test-runs/<run-id>/`
- **AND** the evidence SHALL include `summary.json`, `command.txt`, `stdout.log`, `stderr.log`, `env.json`, and `artifacts/`
- **AND** agents SHALL NOT hand-write the official evidence metadata.

