## ADDED Requirements

### Requirement: Sync push dry-run SHALL never call remote write APIs
TaskBridge SHALL plan sync push operations separately from remote Provider mutation so dry-run can report intended work without creating, updating, or deleting remote tasks.

#### Scenario: dry-run create branch does not create remotely
- **GIVEN** local tasks need to be created in a remote Provider
- **WHEN** the operator runs `taskbridge sync push --dry-run`
- **THEN** TaskBridge SHALL report the planned create operations
- **AND** it SHALL NOT call Provider `CreateTask`
- **AND** actual remote write counters SHALL remain zero

#### Scenario: dry-run update branch does not update remotely
- **GIVEN** local tasks would update existing remote Provider tasks
- **WHEN** the operator runs `taskbridge sync push --dry-run`
- **THEN** TaskBridge SHALL report the planned update operations
- **AND** it SHALL NOT call Provider `UpdateTask`
- **AND** actual remote write counters SHALL remain zero

#### Scenario: dry-run delete branch does not delete remotely
- **GIVEN** local state would delete remote Provider tasks when delete mode is enabled
- **WHEN** the operator runs `taskbridge sync push --delete --dry-run`
- **THEN** TaskBridge SHALL report the planned delete operations
- **AND** it SHALL NOT call Provider `DeleteTask`
- **AND** actual remote write counters SHALL remain zero

### Requirement: Sync write accounting SHALL distinguish planned and written operations
TaskBridge SHALL expose planned write counts separately from actual persisted remote write counts so dry-run output cannot be mistaken for completed remote mutation.

#### Scenario: dry-run has planned writes but zero written writes
- **GIVEN** a sync push plan contains create, update, or delete operations
- **WHEN** the operator runs `taskbridge sync push --dry-run --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the result SHALL distinguish planned operations from written operations
- **AND** written create, update, and delete counts SHALL be zero

#### Scenario: confirmed writes report actual writes only after Provider success
- **GIVEN** a confirmed sync push performs remote Provider mutations
- **WHEN** a Provider write succeeds or fails
- **THEN** actual written counters SHALL include only successful Provider writes
- **AND** failed writes SHALL be represented as errors or failed outcomes, not as successful writes

### Requirement: Remote destructive sync operations SHALL require explicit confirmation
TaskBridge SHALL block remote deletes and remote overwrites unless the operator explicitly confirms the sync push operation.

#### Scenario: remote delete without confirmation is blocked before Provider call
- **GIVEN** `taskbridge sync push --delete` would delete a remote task
- **WHEN** the operator omits `--confirm` or the approved equivalent confirmation flag
- **THEN** TaskBridge SHALL fail before calling Provider `DeleteTask`
- **AND** the result SHALL explain that explicit confirmation is required
- **AND** no remote task SHALL be deleted

#### Scenario: remote overwrite without confirmation is blocked before Provider call
- **GIVEN** `taskbridge sync push` would overwrite an existing remote task
- **WHEN** the operator omits `--confirm` or the approved equivalent confirmation flag
- **THEN** TaskBridge SHALL fail before calling Provider `UpdateTask`
- **AND** the result SHALL explain that explicit confirmation is required
- **AND** no remote task SHALL be overwritten

#### Scenario: confirmed remote write uses English projection output
- **GIVEN** a sync push is explicitly confirmed
- **WHEN** TaskBridge renders human or machine output
- **THEN** human-facing text SHALL be English
- **AND** machine stdout SHALL remain parseable through the existing projection/output helpers
