# Tasks: TaskBridge Contract Reconciliation

- [x] 1. Update OpenSpec deltas for sync safety, action execution audit, and Go toolchain prose.
  - Owner: `cli/taskbridge`
  - Lane: specs
  - Depends on: none
  - Acceptance: `openspec validate taskbridge-contract-reconciliation --strict` passes.

- [x] 2. Update current command docs to remove stale default-Chinese and parse-Chinese wording.
  - Owner: `cli/taskbridge`
  - Lane: docs
  - Depends on: 1
  - Acceptance: conflict scan finds no stale default-Chinese command-output wording in current docs/specs.

- [x] 3. Validate and archive the reconciliation change.
  - Owner: `cli/taskbridge`
  - Lane: closeout
  - Depends on: 1, 2
  - Acceptance: `openspec validate --all --strict` passes after archive, and no archived historical files were edited.
