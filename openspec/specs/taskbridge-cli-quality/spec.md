# taskbridge-cli-quality Specification

## Purpose
定义 TaskBridge CLI 质量基线：机器 stdout 可解析、命令层保持薄、provider/store 行为单一真源、测试证据脱敏、错误路径非零且无 panic。
## Requirements
### Requirement: List JSON output SHALL remain parseable
`taskbridge list --format json` SHALL write exactly one JSON object to stdout for every successful list invocation, without human text, emoji, table rows, pagination hints, logs, prompts, or diagnostics appended before or after the JSON payload.

#### Scenario: empty result remains JSON
- GIVEN the task store contains no tasks matching the list filters
- WHEN an operator runs `taskbridge list --format json`
- THEN stdout SHALL parse as one JSON object
- AND stdout SHALL NOT contain localized empty-result prose, emoji, markdown, table borders, or pagination hint text
- AND empty result state SHALL be represented by JSON fields, not by human prose on stdout

#### Scenario: paginated result remains JSON
- GIVEN the task store contains more tasks than the requested limit
- WHEN an operator runs `taskbridge list --format json --limit 5`
- THEN stdout SHALL parse as one JSON object
- AND pagination metadata such as `limit`, `offset`, `total`, or `has_more` SHALL be represented in JSON
- AND stdout SHALL NOT append a localized `--offset` hint after the JSON payload

#### Scenario: human hints stay outside machine stdout
- GIVEN an operator requests a machine-readable list format
- WHEN TaskBridge needs to show diagnostics, warnings, or next-page hints
- THEN those messages SHALL be represented as structured JSON fields or written to stderr
- AND they SHALL NOT corrupt stdout parsing

### Requirement: Task output rendering SHALL use an internal projection boundary
TaskBridge SHALL render task list output through an internal projection boundary so CLI renderers do not expose `internal/model.Task` through a public package API.

#### Scenario: renderer does not expose internal model publicly
- GIVEN a renderer package is importable outside `internal/`
- WHEN its exported API is inspected
- THEN it SHALL NOT accept or return `internal/model.Task`
- AND CLI-specific task rendering SHOULD live under an `internal/` package unless a stable external DTO is explicitly designed

#### Scenario: one projection supports multiple renderers
- GIVEN a task list result is ready for output
- WHEN TaskBridge renders JSON, table, compact, TSV, or markdown
- THEN each renderer SHALL consume the same task projection DTO
- AND filtering, sorting, pagination, and field derivation SHALL NOT be duplicated inside format-specific renderers

### Requirement: List command SHALL remain a thin command layer
`cmd/list.go` SHALL be limited to CLI flag parsing, dependency wiring, output stream selection, and command exit behavior.

#### Scenario: query logic stays outside cmd
- GIVEN list filtering, sorting, pagination, or search behavior changes
- WHEN the implementation is reviewed
- THEN that behavior SHALL live in `internal/taskquery`, storage query helpers, or a focused internal service
- AND `cmd/list.go` SHALL NOT grow new business-rule branches for those behaviors

#### Scenario: render logic stays outside cmd
- GIVEN a new list output format or field is added
- WHEN the implementation is reviewed
- THEN projection and renderer behavior SHALL live outside `cmd/list.go`
- AND command code SHALL only choose the renderer based on flags and output mode

### Requirement: Provider and filestore query behavior SHALL have one source of truth
TaskBridge SHALL keep provider enumeration and filestore query behavior centralized to prevent hard-coded list drift and duplicated filtering logic.

#### Scenario: provider list is centralized
- GIVEN code needs to iterate default providers
- WHEN loader, auth, sync, provider status, or storage setup code is reviewed
- THEN it SHALL use the provider catalog API
- AND it SHALL NOT introduce a second hard-coded provider name list

#### Scenario: filestore query behavior is shared
- GIVEN `FileStorage` and `ProviderStorage` both expose task query behavior
- WHEN filtering, sorting, or pagination is tested against equivalent fixture data
- THEN both stores SHALL produce consistent results
- AND query behavior SHALL be implemented by shared helpers rather than duplicated branches

### Requirement: Command-level output contracts SHALL be tested with evidence
TaskBridge SHALL include command-level tests for machine-readable stdout and preserve redacted evidence for integration, process e2e, or golden-output runs.

#### Scenario: JSON stdout is parsed by tests
- GIVEN a command-level test covers JSON list output
- WHEN the test validates stdout
- THEN it SHALL parse stdout as JSON
- AND it SHALL fail if non-JSON human text is mixed into stdout

#### Scenario: integration evidence is retained
- GIVEN a command-level, process e2e, or golden-output test run is executed
- WHEN the run completes or fails
- THEN evidence SHALL be written under `temp/integration-test-runs/<run-id>/`
- AND evidence SHALL include `summary.json`, `command.txt`, `stdout.log`, `stderr.log`, `env.json`, and `artifacts/`
- AND evidence SHALL redact secrets, tokens, Authorization headers, raw prompts, provider payloads, hidden system prompts, tool private parameters, and full chain-of-thought

### Requirement: Store initialization failures SHALL not panic
TaskBridge CLI commands SHALL handle storage initialization failures as normal command errors and SHALL NOT panic because cleanup functions are nil.

#### Scenario: storage path points to an existing file
- GIVEN `<existing-file>` is a regular file path
- WHEN an operator runs `taskbridge --storage-path <existing-file> list --format json`
- THEN the command SHALL exit non-zero
- AND stderr SHALL contain an English storage initialization error summary
- AND stdout SHALL NOT contain a panic stack trace
- AND the process SHALL NOT dereference a nil cleanup function

#### Scenario: command code defers cleanup safely
- GIVEN command code calls the shared store factory
- WHEN the store factory returns an error
- THEN cleanup SHALL either be a non-nil noop function or SHALL NOT be deferred before the error is handled
- AND the original storage error SHALL remain visible to the CLI error formatter

### Requirement: Auth error paths SHALL return non-zero exit codes
TaskBridge auth commands SHALL return command errors for provider credential and token failures instead of printing an error and returning success.

#### Scenario: OAuth credentials are missing
- GIVEN Microsoft or Feishu credential files do not exist
- WHEN an operator runs `taskbridge auth login microsoft` or `taskbridge auth login feishu`
- THEN the command SHALL return a non-zero exit code
- AND stderr or the approved human output channel SHALL include an English setup guide
- AND scripts SHALL NOT observe the command as successful

#### Scenario: static token refresh cannot read token state
- GIVEN TickTick or Dida token state cannot be read
- WHEN an operator runs `taskbridge auth refresh ticktick` or `taskbridge auth refresh dida`
- THEN the command SHALL return a non-zero exit code
- AND the error SHALL identify the provider and token read failure

#### Scenario: unsupported provider messaging is centralized
- GIVEN auth command code needs to describe supported providers
- WHEN a provider name is invalid or unsupported
- THEN the supported provider list SHALL be derived from provider catalog metadata
- AND auth command code SHALL NOT maintain a second hard-coded provider list

### Requirement: Shared command output helpers SHALL own JSON rendering
TaskBridge commands that support machine-readable output SHALL route JSON rendering through a shared output helper instead of hand-writing `json.MarshalIndent` branches in each command.

#### Scenario: analyze output uses shared rendering
- GIVEN an operator runs an analyze subcommand with `--format json`
- WHEN the command renders the result
- THEN JSON serialization errors SHALL return `commandError`
- AND the command SHALL NOT ignore marshal errors
- AND stdout SHALL contain only the JSON payload

#### Scenario: list-like commands use one output split
- GIVEN `lists`, `sync`, or `task show` renders JSON output
- WHEN the implementation is reviewed
- THEN the command SHALL use the shared output helper or a focused internal renderer
- AND it SHALL NOT duplicate stdout/stderr split logic in command handlers

#### Scenario: pipe and quiet mode are explicit in tests
- GIVEN stdout is a pipe or `--quiet` is set
- WHEN a command chooses compact or JSON output automatically
- THEN tests SHALL document the expected mode switch
- AND the implementation SHALL keep stdout parseable by scripts

### Requirement: Storage read APIs SHALL not expose mutable internal state
TaskBridge storage implementations SHALL return owned copies from read APIs and SHALL store owned copies from write APIs so callers cannot mutate store internals without an explicit save operation.

#### Scenario: GetTask result mutation does not alter store state
- GIVEN a task contains top-level fields, `Tags`, `SubtaskIDs`, `Metadata.CustomFields`, and time pointer fields
- WHEN a caller reads it with `GetTask` and mutates the returned object without calling `SaveTask`
- THEN a second `GetTask` SHALL return the original stored values
- AND the store SHALL NOT be marked dirty by the read-side mutation

#### Scenario: List and query results are deep copies
- GIVEN tasks are returned by `ListTasks` or `QueryTasks`
- WHEN a caller mutates returned nested slices, maps, metadata, or time pointers
- THEN later storage reads SHALL NOT observe those mutations unless the caller explicitly saves the changed task

#### Scenario: SaveTask stores an owned copy
- GIVEN a caller passes a task pointer to `SaveTask`
- WHEN the caller mutates that pointer after `SaveTask` returns
- THEN later reads SHALL continue to show the saved value, not the caller's post-save mutation

### Requirement: Persistence errors SHALL be surfaced and counted accurately
TaskBridge sync, project sync, and TUI mutation paths SHALL surface local persistence errors instead of ignoring them, and success counts SHALL reflect operations that were actually recorded locally when local persistence is part of the operation.

#### Scenario: project sync local write failure is reported
- GIVEN a provider create/update succeeds but `TaskStore.SaveTask` fails while syncing a project
- WHEN project sync returns its result
- THEN the result SHALL include a provider/task-specific error
- AND pushed or updated counters SHALL NOT count the task as successful local state

#### Scenario: provider pull local write failure is reported
- GIVEN provider tasks are pulled into local storage
- WHEN `SaveTaskList` or `SaveTask` fails for a pulled item
- THEN the pull operation SHALL report the failed list or task
- AND it SHALL NOT silently continue as if all local state was updated

#### Scenario: TUI write failures are visible
- GIVEN the TUI attempts to delete or toggle a task
- WHEN the storage write fails
- THEN the TUI model SHALL retain a visible error state
- AND it SHALL NOT optimistically remove or mark the task as completed in the filtered task list

