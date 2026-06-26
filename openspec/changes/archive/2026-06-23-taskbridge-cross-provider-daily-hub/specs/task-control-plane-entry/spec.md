# task-control-plane-entry Specification Delta

## MODIFIED Requirements

### Requirement: today and next SHALL be the primary daily control-plane commands
TaskBridge SHALL make `today` and `next` the concise daily entry points for deciding what to do across all connected Todo Providers, while keeping detailed list/search behavior in existing list commands.

#### Scenario: today summarizes decisions instead of dumping every task
- **GIVEN** the local store contains active, overdue, inbox, project-linked, work-domain, and life-domain tasks
- **WHEN** the operator runs `taskbridge today`
- **THEN** the default stdout SHALL be a short English summary with decision sections and one recommended next command
- **AND** it SHALL represent both work and life tasks when present
- **AND** it SHALL NOT be a raw JSON dump, full task table, debug log, or provider payload

#### Scenario: next returns a bounded recommendation set
- **GIVEN** the local store contains more than five active tasks across multiple Providers or domains
- **WHEN** the operator runs `taskbridge next --json`
- **THEN** stdout SHALL parse as one JSON object
- **AND** the result SHALL include no more than the default recommendation limit unless `--limit` is set
- **AND** large tasks over the configured threshold SHALL be recommended for split/review rather than presented as direct next work
- **AND** recommendations SHALL preserve source and domain metadata for traceability
