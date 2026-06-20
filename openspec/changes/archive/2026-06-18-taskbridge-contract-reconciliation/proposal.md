# Proposal: TaskBridge Contract Reconciliation

## Why

TaskBridge has already converged on an English-first local task execution control plane for humans and agents. The accepted specs and current docs still contain a few stale statements from earlier planning rounds: some command docs describe default Chinese summaries, `sync-write-safety` names only `sync push` even though `sync bidirectional`, `sync watch`, and `serve --sync` also reach remote write paths, and `remote_write` appears in dangerous action examples without saying it is a reserved unsupported type.

These mismatches are planning debt. They can mislead future implementation work even though the current runtime direction is already clear.

## What Changes

- Update the accepted sync safety contract so remote delete and overwrite confirmation applies to `sync push`, `sync bidirectional`, `sync watch`, and `serve --sync` with `--sync-confirm`.
- Clarify that `remote_write` and `conflict_discard` are reserved dangerous action types: they require confirmation if ever supported, and today must not be advertised as implemented remote-write behavior.
- Translate the accepted Go development toolchain spec prose to English without changing task names, Taskfile requirements, build behavior, or quality gates.
- Update current TaskBridge command docs so default human output is English and automation must use `--json`, `--agent`, `--events`, or existing command-specific machine modes instead of parsing human text.

## Compatibility

- This change is additive and clarifying only.
- It does not change command names, flags, JSON envelope fields, `--agent` keys, `--events` event types, exit code semantics, Go APIs, storage schema, or Provider behavior.
- No deprecation window is required because this change aligns specs and docs with the current accepted product direction rather than changing runtime behavior.

## Non-Goals

- Do not edit archived OpenSpec changes or historical planning records.
- Do not implement new remote Provider writes, bidirectional auto-resolve, MCP adapter behavior, Apple Reminders, or OmniFocus.
- Do not change runtime code or generated release artifacts in this reconciliation.
