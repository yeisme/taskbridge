# taskbridge-integration-test-evidence Specification

## Purpose
TBD - created by archiving change taskbridge-control-plane-hardening. Update Purpose after archive.
## Requirements
### Requirement: TaskBridge integration tests SHALL write local redacted evidence
TaskBridge SHALL provide a project-local integration/process e2e test entrypoint that writes evidence for every run under `temp/integration-test-runs/<run-id>/`.

#### Scenario: successful integration run writes required files
- **WHEN** a developer runs `task test:integration`
- **THEN** TaskBridge SHALL execute command-level, process e2e, or golden-output tests
- **AND** it SHALL create `temp/integration-test-runs/<run-id>/summary.json`, `command.txt`, `stdout.log`, `stderr.log`, `env.json`, and `artifacts/`
- **AND** `summary.json` SHALL include `schema_version`, `project`, `run_id`, `layer`, `command`, `status`, `exit_code`, `started_at`, `finished_at`, `duration_ms`, `evidence`, and `redaction`

#### Scenario: failed integration run preserves evidence and exit code
- **GIVEN** an integration command fails
- **WHEN** `task test:integration` exits
- **THEN** the evidence directory SHALL still include the required files
- **AND** `summary.json` SHALL record `status=failed` and the original exit code
- **AND** the wrapper SHALL exit with the original non-zero exit code

### Requirement: Evidence SHALL be redacted consistently
TaskBridge SHALL redact sensitive values from integration evidence, stdout/stderr artifacts, event logs, snapshots, and golden files.

#### Scenario: sensitive content is not persisted
- **GIVEN** the test environment contains tokens, Authorization headers, Provider payload examples, raw prompts, or private tool arguments
- **WHEN** evidence is written under `temp/integration-test-runs/<run-id>/`
- **THEN** persisted logs and env files SHALL redact those values
- **AND** the evidence SHALL NOT contain hidden system prompts, full chain-of-thought, unredacted Provider payloads, cookies, passwords, private keys, or bearer tokens

### Requirement: Contract tests SHALL cover control-plane and agent boundaries
TaskBridge integration tests SHALL cover the user-visible and machine-visible contracts that make the control plane trustworthy.

#### Scenario: command contracts are covered by process tests
- **WHEN** `task test:integration` runs
- **THEN** tests SHALL cover `taskbridge demo today --json`, `taskbridge today --json`, `taskbridge next --agent`, `taskbridge review --apply-file actions.json --dry-run`, `taskbridge agent today`, and `taskbridge agent execute --action-file actions.json --dry-run`
- **AND** tests SHALL assert stdout parseability, stderr separation, exit code semantics, audit receipt creation, and no mutation during dry-run

