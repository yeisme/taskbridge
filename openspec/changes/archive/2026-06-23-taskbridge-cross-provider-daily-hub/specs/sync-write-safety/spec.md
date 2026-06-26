# sync-write-safety Specification Delta

## ADDED Requirements

### Requirement: Sync pull all SHALL aggregate Provider reads safely
TaskBridge SHALL provide an all-provider pull path that attempts enabled authenticated Providers and reports per-provider outcomes without calling remote write APIs.

#### Scenario: pull all attempts every available Provider
- **GIVEN** multiple Providers are enabled and authenticated
- **WHEN** the operator runs `taskbridge sync pull --all --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the result SHALL list Providers attempted, succeeded, failed, and skipped
- **AND** each Provider outcome SHALL include task counts and errors safe for human diagnostics
- **AND** TaskBridge SHALL NOT call remote Provider mutation APIs

#### Scenario: pull all reports partial failure
- **GIVEN** one Provider succeeds and another Provider fails during pull
- **WHEN** the operator runs `taskbridge sync pull --all --json`
- **THEN** the result status SHALL distinguish partial success from total success
- **AND** successful Provider results SHALL remain visible
- **AND** failed Provider errors SHALL include a safe next action such as `taskbridge auth status` or `taskbridge provider test <provider>`
