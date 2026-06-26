# cross-provider-daily-hub Specification

## Purpose
Define TaskBridge's cross-provider daily hub contract: classify tasks by domain without replacing Provider source, summarize work and life tasks in `today`, bound `next` recommendations across Providers and domains, and keep `review` read-only while reporting coverage and backlog health.
## Requirements
### Requirement: TaskBridge SHALL classify tasks by domain without replacing Provider source
TaskBridge SHALL represent a task's life context with a stable `domain` value while preserving the existing Provider `source` and remote identity fields.

#### Scenario: legacy tasks without domain remain visible
- **GIVEN** a local task has a valid source but no domain metadata
- **WHEN** TaskBridge renders `today --json` or `next --json`
- **THEN** the task SHALL remain visible
- **AND** its domain SHALL be reported as `unknown`
- **AND** its original `source` and remote identity metadata SHALL be preserved

#### Scenario: explicit domain wins over inferred signals
- **GIVEN** a task has an explicit domain set to `life`
- **AND** its Provider source or list name looks work-related
- **WHEN** TaskBridge classifies the task for daily hub output
- **THEN** the explicit `life` domain SHALL be used
- **AND** TaskBridge SHALL NOT silently overwrite the explicit domain

#### Scenario: provider source is not a domain
- **GIVEN** Microsoft, Todoist, Feishu, TickTick, Dida365, Google Tasks, or local tasks contain both work and life items
- **WHEN** TaskBridge groups tasks
- **THEN** it SHALL group by domain and decision section, not only by Provider source
- **AND** Provider source SHALL remain available as metadata for traceability

### Requirement: TaskBridge SHALL provide a cross-provider daily hub
TaskBridge SHALL make `today` the primary cross-provider summary for work, life, inbox, overdue, recommended next actions, and sync warnings.

#### Scenario: today summarizes work and life together
- **GIVEN** the local store contains active tasks from multiple Providers and domains
- **WHEN** the operator runs `taskbridge today`
- **THEN** the default output SHALL show a concise English daily hub summary
- **AND** it SHALL include Work and Life sections when those domains have relevant tasks
- **AND** it SHALL include one runnable next command
- **AND** it SHALL NOT require opening each Provider app separately to understand the day

#### Scenario: today JSON exposes stable sections
- **GIVEN** the local store contains work, life, unknown-domain, overdue, and sync-warning tasks
- **WHEN** the operator runs `taskbridge today --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the data SHALL include stable section identifiers for `work`, `life`, `inbox`, `overdue`, `recommended_next`, and `sync_warnings`
- **AND** task entries SHALL include `source` and `domain` metadata
- **AND** stderr SHALL contain diagnostics only

### Requirement: TaskBridge SHALL recommend next work across Providers and domains
TaskBridge SHALL rank next recommendations across all local tasks instead of returning one Provider's task list.

#### Scenario: next returns a bounded cross-provider recommendation set
- **GIVEN** more than five active tasks exist across multiple Providers and domains
- **WHEN** the operator runs `taskbridge next --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the default result SHALL include a bounded recommendation set
- **AND** each recommendation SHALL include reason metadata covering due date, priority, domain, source, project, or sync risk as applicable

#### Scenario: sync-risk tasks are not recommended as direct mutations
- **GIVEN** a high-priority task has a sync conflict or uncertain Provider state
- **WHEN** TaskBridge ranks recommendations
- **THEN** it MAY recommend reviewing or resolving the task
- **AND** it SHALL NOT recommend direct complete, defer, reschedule, or remote-write execution for that task without conflict resolution and confirmation

### Requirement: TaskBridge SHALL review work/life coverage and backlog health
TaskBridge SHALL use `review` to summarize weekly cross-domain coverage, unknown-domain backlog, overdue risk, and Provider sync health without implicit writes.

#### Scenario: review reports coverage without writing tasks
- **GIVEN** the local store contains work, life, personal, unknown-domain, and overdue tasks
- **WHEN** the operator runs `taskbridge review --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the result SHALL include work/life coverage, unknown-domain count, overdue backlog, Provider health, and suggested actions
- **AND** no task status, due date, domain, source, or Provider state SHALL be modified
