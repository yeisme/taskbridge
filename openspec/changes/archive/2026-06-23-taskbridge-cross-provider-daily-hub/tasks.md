# Tasks: TaskBridge Cross-Provider Daily Hub

- [x] 1. Define task domain model and projection contract.
  - Owner: `cli/taskbridge`
  - Lane: model/output
  - Depends on: none
  - Acceptance: task projection exposes `domain` with `work|life|personal|unknown`; legacy tasks without domain render as `unknown`; JSON/agent output remains parseable.
  - Evidence: `internal/model/task.go`, `internal/taskoutput/projection.go`, `internal/taskoutput/projection_test.go`; `go test ./internal/taskoutput -run Projection -count=1` passed via focused tests and `task test`.

- [x] 2. Implement conservative domain inference and user-visible rules.
  - Owner: `cli/taskbridge`
  - Lane: model/controlplane
  - Depends on: 1
  - Acceptance: explicit domain beats list/project/tag inference; unresolved tasks become `unknown`; no Provider/source-only rule hides tasks.
  - Evidence: `internal/controlplane/domain.go`, `internal/controlplane/service_test.go`; `TestDomainClassificationExplicitBeatsInferenceAndLegacyIsUnknown` verifies explicit domain wins and legacy tasks stay visible as `unknown`.

- [x] 3. Add `sync pull --all` aggregation.
  - Owner: `cli/taskbridge`
  - Lane: sync/provider
  - Depends on: none
  - Acceptance: all enabled authenticated Providers are attempted; partial failure returns per-provider status; `--json` includes attempted/succeeded/failed counts and does not corrupt stdout.
  - Evidence: `cmd/sync.go`, `cmd/sync_output_test.go`; `TestSyncPullAllAggregatesSuccessFailureAndSkippedProviders` verifies success/failure/skipped aggregation and no remote write API calls.

- [x] 4. Redesign `today` as the cross-provider daily hub.
  - Owner: `cli/taskbridge`
  - Lane: controlplane/output
  - Depends on: 1, 2, 3
  - Acceptance: `today` groups Work, Life, Inbox, Overdue, Recommended next, and Sync warnings; default output is concise English; `today --json` has stable sections and source/domain metadata.
  - Evidence: `internal/controlplane/service.go`, `cmd/controlplane.go`, `cmd/controlplane_contract_test.go`; `TestTodayJSONContract` verifies stable sections and source/domain metadata.

- [x] 5. Update `next` ranking for cross-provider work/life recommendations.
  - Owner: `cli/taskbridge`
  - Lane: controlplane
  - Depends on: 4
  - Acceptance: default recommendations are bounded; ranking includes due date, priority, domain, source, project next, and sync risk; conflicted tasks recommend review/resolve rather than direct mutation.
  - Evidence: `internal/controlplane/service.go`, `cmd/controlplane_contract_test.go`; `TestNextReasonsIncludeDomainSourceProjectAndSyncRisk` and `TestNextAgentContract` verify ranking metadata and agent facts.

- [x] 6. Update `review` for weekly coverage and backlog analysis.
  - Owner: `cli/taskbridge`
  - Lane: governance/controlplane
  - Depends on: 1, 4
  - Acceptance: review reports work/life coverage, unknown-domain count, overdue backlog, Provider health, and suggested actions without implicit writes.
  - Evidence: `internal/controlplane/service.go`, `internal/controlplane/service_test.go`; `TestReviewSuggestsSplittingLargeTasks` verifies review summary includes unknown-domain and provider-health facts while keeping review read-only unless action file is explicitly applied.

- [x] 7. Update README and command docs around the new product promise.
  - Owner: `cli/taskbridge`
  - Lane: docs
  - Depends on: 3, 4, 5, 6
  - Acceptance: first-run path shows `sync pull --all -> today -> next`; docs describe TaskBridge as a CLI command center for all Todo apps; generated command docs match Cobra help where applicable.
  - Evidence: `README.md`, `docs/README.md`, `docs/commands/README.md`, `docs/commands/today.md`, `docs/commands/next.md`, `docs/commands/review.md`, `docs/commands/sync.md` updated with real user-runnable commands.

- [x] 8. Add integration smoke for the core daily hub path.
  - Owner: `cli/taskbridge`
  - Lane: test/integration
  - Depends on: 3, 4, 5
  - Acceptance: temp storage run covers `doctor`, `demo today`, `sync pull --all --dry-run` or approved no-auth fallback, `today --json`, and `next --json`; machine stdout parses; no remote write APIs are called.
  - Evidence: `cmd/daily_hub_integration_test.go`, `Taskfile.yml`; `task test:integration` passed and wrote evidence to `temp/integration-test-runs/20260623T143801Z-2015752/summary.json`.

- [x] 9. Run quality gates.
  - Owner: `cli/taskbridge`
  - Lane: validation
  - Depends on: 1-8
  - Acceptance: `task test`, `task test:integration`, `task build`, `task release:check`, and `task check` pass.
  - Evidence: `task test`, `task test:integration`, `task build`, `task release:check`, and `task check` passed on 2026-06-23.

## Final Verification

```bash
go test ./internal/controlplane ./internal/taskoutput ./cmd -run 'Domain|Today|Next|Review|SyncPullAll|DailyHub|Controlplane|Projection|DemoToday' -count=1
task test:integration
task test
task build
task release:check
task check
openspec validate --all --strict
```
