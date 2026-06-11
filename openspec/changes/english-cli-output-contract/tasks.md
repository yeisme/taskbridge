# Tasks: English CLI Output Contract

Owner: `cli/taskbridge`  
Primary surface: `taskbridge`  
Project type: Go task orchestration CLI

Parallel lanes:

- Lane A: inventory, tests, and output contract audit.
- Lane B: projection and renderer implementation.
- Lane C: help, docs, task runner text, and golden updates.
- Lane D: final verification and evidence.

## 0. Baseline

- [ ] 0.1 Run `openspec validate --all` from `cli/taskbridge` and record whether existing unrelated changes block validation.
- [ ] 0.2 Run `go test ./cmd -run 'Output|English|Agent|JSON|Help|Provider|Auth|List|Config|Version' -count=1` to capture the current focused output behavior before edits.
- [ ] 0.3 Run `rg -n "[\p{Han}]" cmd internal src apps packages docs tests Taskfile.yml README.md AGENTS.md 2>/dev/null || true` from `cli/taskbridge` and classify matches as CLI chrome, docs/help, tests/goldens, user-authored fixture data, quoted source material, or third-party payload fields.

Acceptance: the task notes identify every output-facing Chinese/localized string that must change and every data/content string that must remain unchanged.

Failure re-check: if the scan returns too much fixture or product content, narrow the next audit to command/help/error files and separately list intentional non-English fixtures.

## 1. Lane A: Contract tests for English human output

- [ ] 1.1 Add focused tests for representative high-level commands proving default output uses English labels: `Status`, `Highlights`, optional `Risks`, optional `Evidence`, and one `Recommended next step`.
- [ ] 1.2 Add help/error tests proving root help, command help, unknown command, validation failure, and no-color mode do not emit localized CLI chrome.
- [ ] 1.3 Add tests that default output is not JSON, not key=value agent output, and not a raw debug dump.
- [ ] 1.4 Add targeted allowlist tests for intentional non-English domain content, such as user-authored notes, quoted source material, fixtures, provider text, or product data.

Validation: `go test ./cmd -run 'Output|English|Agent|JSON|Help|Provider|Auth|List|Config|Version' -count=1`.

Expected result: tests fail before implementation when localized CLI chrome is still present, while intentional domain data remains allowed.

Failure re-check: do not solve failures by deleting fixtures or weakening assertions; split output chrome from domain payloads in the projection.

## 2. Lane B: Shared projection and renderers

- [ ] 2.1 Locate the existing projection/renderer boundary and document it in code comments only where the flow is not self-explanatory.
- [ ] 2.2 Move command-local human strings into renderer-level English text when a command currently hand-builds summaries, tables, or errors.
- [ ] 2.3 Ensure `--json` emits exactly one valid envelope on stdout for supported commands, including failed envelopes on error paths.
- [ ] 2.4 Ensure `--agent` emits stable ASCII key=value lines with `spec_version`, `mode=agent`, `command`, and `status`; it must not include localized prose, ANSI, tables, raw prompts, or provider payloads.
- [ ] 2.5 Ensure long-running commands that support `--events` emit valid NDJSON on stdout and keep progress/log diagnostics on stderr.
- [ ] 2.6 Ensure `--explain`, where supported, is an English review summary with conclusion, evidence, confidence, risks, tradeoffs, and next step; it must not expose full chain-of-thought.

Validation: `go test ./cmd -run 'Output|English|Agent|JSON|Help|Provider|Auth|List|Config|Version' -count=1`.

Expected result: human, JSON, agent, events, and explain modes render from the same projection or an explicitly documented compatibility adapter.

Failure re-check: if a command has legacy machine output, preserve compatibility with a migration note instead of silently changing parseable fields.

## 3. Lane C: Help, docs, task runner text, and examples

- [ ] 3.1 Translate command help, usage, examples, validation messages, and common errors to English.
- [ ] 3.2 Update `Taskfile.yml`, README, docs, and command docs owned by `cli/taskbridge` so task descriptions and command examples are English.
- [ ] 3.3 Replace local wrapper examples with real commands a human can run, such as `taskbridge --help`, `taskbridge status --json`, project task commands, or documented service commands.
- [ ] 3.4 Update golden files and snapshots only after renderer tests prove machine output remains parseable and redacted.
- [ ] 3.5 Remove tests that require agents to parse localized default output; agents must use `--agent`, `--json`, or `--events`.

Validation: `rg -n "[\p{Han}]" README.md docs Taskfile.yml cmd internal src apps packages tests 2>/dev/null || true` plus `go test ./cmd -run 'Output|English|Agent|JSON|Help|Provider|Auth|List|Config|Version' -count=1`.

Expected result: remaining non-English matches are intentional domain data, quoted content, archived history, compatibility fixtures, or third-party payload samples with comments explaining why they remain.

Failure re-check: if docs mention commands that no longer exist, fix docs or help metadata rather than adding fake command aliases.

## 4. Lane D: Redaction, evidence, and integration boundary

- [ ] 4.1 Add or update redaction tests proving stdout, stderr, events, traces, snapshots, sidecars, and integration evidence do not include secrets, tokens, Authorization headers, cookies, raw prompts, hidden system prompts, provider payloads, private tool arguments, or chain-of-thought.
- [ ] 4.2 If integration/component/e2e tests are changed, ensure the project-owned runner writes evidence under `temp/integration-test-runs/<run-id>/` with `summary.json`, `command.txt`, `stdout.log`, `stderr.log`, `env.json`, and `artifacts/`.
- [ ] 4.3 Run integration evidence only if this change touches integration, component, system, e2e, service lifecycle, or provider behavior.

Validation: `task test:integration`.

Expected result: evidence is generated by project scripts or services, redacted, and not committed.

Failure re-check: if evidence generation fails, fix the project runner instead of hand-writing evidence metadata.

## 5. Final verification

- [ ] 5.1 Run focused output tests: `go test ./cmd -run 'Output|English|Agent|JSON|Help|Provider|Auth|List|Config|Version' -count=1`.
- [ ] 5.2 Run build/typecheck: `go test ./... && go build -trimpath -o dist/taskbridge .`.
- [ ] 5.3 Run `openspec validate english-cli-output-contract --strict` from `cli/taskbridge`.
- [ ] 5.4 Run the full local project gate if focused tests and build pass.
- [ ] 5.5 Record command evidence in this task file before archive: command, exit status, and one-line result.
- [ ] 5.6 Inspect `git status --short` and confirm no build artifacts, coverage output, provider cache, local credentials, temp evidence, node_modules, or unrelated generated files are included.

Expected result: Taskbridge has English default CLI output and preserved machine output contracts, with tests guarding both.

Failure re-check: if validation is blocked by pre-existing unrelated worktree changes, record the blocker and still validate this change directly where possible.
