## Why

Taskbridge has user-visible command output, help text, errors, diagnostics, logs, and automation modes that must follow one repository-wide language and output contract. The root workspace now requires English for project artifacts and CLI-facing operator text, while machine protocols keep stable English keys and existing command names.

This change plans the migration for Taskbridge without implementing it from the repository root. It gives the owning subproject a concrete OpenSpec task package for auditing localized output, moving human text to English, preserving machine formats, and adding contract tests for the shared AI-native CLI output modes.

## What Changes

- Audit every `taskbridge` command, help page, default summary, error path, long-running progress output, run receipt, fixture, golden file, and operator-facing documentation for non-English user-visible text.
- Require default human output, help text, stderr diagnostics, user-visible logs, and `--explain` reports to use concise English by default.
- Keep machine protocol fields stable: command names, flags, JSON envelope keys, `--agent` keys, enum values, event types, schema fields, provider ids, model ids, and third-party API fields remain English or existing names.
- Route all supported output modes through one command projection: summary, `--json`, `--agent`, `--events`, and `--explain` where the project exposes them.
- Add contract tests that prevent localized prose, ANSI, logs, raw prompts, provider payloads, secrets, or chain-of-thought from leaking into machine stdout, sidecars, traces, fixtures, or evidence.

## Impact

- Owner: `cli/taskbridge`.
- User-facing scope: CLI/default output, help text, stderr diagnostics, docs command examples, test goldens, task descriptions, and service runner text owned by this subproject.
- Machine-facing scope: output envelope compatibility, `--agent` key=value lines, `--json` objects, `--events` NDJSON, stdout/stderr separation, redaction, and schema versioning.
- Out of scope: changing command names, flags, provider ids, schema keys, persisted domain content language, or third-party payload field names.
- Coordination: Taskbridge already has CLI contract integration evidence; this change should reuse that runner and keep evidence redacted.
