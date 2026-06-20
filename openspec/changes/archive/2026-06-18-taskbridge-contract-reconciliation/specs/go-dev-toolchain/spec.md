## MODIFIED Requirements

### Requirement: Go 开发工具链标准化落地
TaskBridge SHALL implement the root Go development toolchain baseline with golangci-lint v2 configuration, aligned Taskfile task semantics, reproducible build output, release checks, and optional hot reload.

#### Scenario: lint configuration covers the baseline
- **WHEN** TaskBridge completes `.golangci.yaml` configuration
- **THEN** the configuration SHALL cover baseline linters including `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `misspell`, and `revive`
- **AND** it SHALL cover formatters including `gofmt` and `goimports`
- **AND** `golangci-lint run --new-from-rev=HEAD~` SHALL exit with code 0

#### Scenario: Taskfile task semantics are aligned
- **WHEN** TaskBridge completes the Taskfile update
- **THEN** the Taskfile SHALL provide `tools:install`, `deps`, `mod-check`, `fmt`, `fmt-check`, `lint`, `lint:new`, `test`, `test:race`, `build`, `snapshot`, `release:check`, `release:local`, and `ci/check` tasks
- **AND** `task ci` SHALL run at least dependency download, module checks, format checks, lint, tests, and build
- **AND** `task build` SHALL write the current-platform binary to `dist/taskbridge` and smoke-run `dist/taskbridge --help`
- **AND** optional Air hot reload SHALL be exposed through `dev:watch` and SHALL NOT be a dependency of `ci` or `check`

#### Scenario: project-specific tasks are preserved
- **WHEN** TaskBridge already has project-specific tasks
- **THEN** the toolchain rollout MUST preserve those tasks
- **AND** it MAY add aliases or supplemental descriptions only for naming consistency
