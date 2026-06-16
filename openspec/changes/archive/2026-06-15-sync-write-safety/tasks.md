## 1. RED tests

- [x] 1.1 Add a sync push dry-run test with a fake Provider that fails if `CreateTask`, `UpdateTask`, or `DeleteTask` is called. Acceptance: create, update, and delete branches report planned operations but perform zero remote writes.
- [x] 1.2 Add counter tests that distinguish `planned_*` from `written_*` or equivalent fields. Acceptance: dry-run may have non-zero planned counts while written counts remain zero.
- [x] 1.3 Add remote delete confirmation tests for `sync push --delete`. Acceptance: without `--confirm` or the approved equivalent, execution fails before `DeleteTask` and reports the confirmation requirement.
- [x] 1.4 Add remote overwrite/update confirmation tests for existing remote tasks. Acceptance: overwrite-capable update paths require explicit confirmation before calling `UpdateTask` when remote state would be replaced.

## 2. Implementation

- [x] 2.1 Move sync push write planning ahead of Provider mutation calls. Acceptance: dry-run builds the same plan shape as confirm mode but never calls remote write APIs.
- [x] 2.2 Gate remote delete and remote overwrite/update operations on explicit confirmation. Acceptance: unsafe operations are blocked before Provider mutation and return a non-zero command result when confirmation is missing.
- [x] 2.3 Update result accounting to keep planned and actual writes separate. Acceptance: output and machine payloads cannot imply that dry-run operations were written.
- [x] 2.4 Route user-facing output through existing projection helpers. Acceptance: human text is English; machine stdout stays parseable; command handlers remain thin.

## 3. Verification

- [x] 3.1 Run focused sync tests only. Evidence: `go test ./internal/sync -run 'TestPushDryRunDoesNotUpdateExistingRemoteTask|TestPushDryRunDoesNotCreateOrDeleteRemoteTasks|TestPushDeleteRemoteRequiresConfirmation|TestPushDeleteWithoutConfirmIsBlockedBeforeRemoteDelete|TestPushUpdateWithoutConfirmIsBlockedBeforeRemoteOverwrite|TestPushConfirmedUpdateAndDeleteCallRemoteWriteAPIs|TestOptions' -count=1` passed; `go test ./cmd -run 'TestSyncDeleteRequiresDryRunOrConfirm|TestSyncForceRequiresDryRunOrConfirm|TestRenderSyncResultUsesSummaryAndMetricsTable|TestRenderSyncResultDryRunSeparatesPlannedAndActualWrites|TestRenderSyncStatusUsesProviderTable|TestRenderSyncControlProjectionUsesAuditTable' -count=1` passed.
- [x] 3.2 Validate this OpenSpec change. Evidence: `openspec validate sync-write-safety --strict` passed.

## Acceptance

- `sync push --dry-run` never calls remote write APIs in create, update, or delete branches.
- Planned operations and actual written operations are distinct in command results.
- Remote delete and overwrite/update paths require explicit confirmation before remote mutation.
- Output remains English for humans and parseable for machines.
