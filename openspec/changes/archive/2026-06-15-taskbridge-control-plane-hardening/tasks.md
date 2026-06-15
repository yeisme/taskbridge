## 0. Preflight and Dependency Alignment

- [x] 0.1 Owner: `cli/taskbridge`; Lane: sequential; Scope: 检查当前工作区和 open changes。运行 `git status --short`、`openspec list`，确认只修改本 change 文件和后续实现文件，不碰无关 runtime skill 同步改动。
- [x] 0.2 Owner: `cli/taskbridge`; Lane: sequential; Depends on: 0.1; Scope: 与 `taskbridge-code-quality-refactor` 对齐。确认 shared output helper、命令级 harness、evidence 任务是否已完成；如未完成，先在该 change 收口或在本 change 中只复用已稳定接口。
- [x] 0.3 Owner: `cli/taskbridge`; Lane: sequential; Depends on: 0.2; Scope: 锁定当前行为基线。为 `today/next/review/agent execute` 当前 stdout/stderr、exit code 和 dry-run 行为补 RED contract 测试。

## 1. Control-plane Entry Experience

- [x] 1.1 Owner: `cli/taskbridge`; Lane: entry; Depends on: 0.3; Scope: 实现 `taskbridge demo today`，复用现有 mock control-plane service，禁止读写 Provider token、远端 API 和真实 task store。Verify: `go test ./cmd/... -run 'DemoToday|ControlPlaneMock' -count=1`。
- [x] 1.2 Owner: `cli/taskbridge`; Lane: entry; Depends on: 1.1; Scope: 强化 `doctor` next action。无 Provider 认证或空存储时输出 `taskbridge demo today`、`taskbridge provider list`、`taskbridge auth status` 等真实可运行命令。Verify: `go test ./cmd/... -run 'Doctor.*NextAction|DoctorJSON' -count=1`。
- [x] 1.3 Owner: `cli/taskbridge`; Lane: entry; Depends on: 1.1; Scope: 调整 `today`、`next`、`review` 默认 summary，使其突出状态、重点、风险和一个推荐下一步，不退化为完整任务列表。Verify: golden/contract 测试覆盖 default stdout 不为 JSON dump。
- [x] 1.4 Owner: `cli/taskbridge`; Lane: entry; Depends on: 1.1, 1.2, 1.3; Scope: 更新 README 和 `docs/cli-design.md` 主路径为 `doctor -> demo today -> today -> next -> review`，弱化多 Provider 列表作为第一入口。Verify: `rg -n "demo today|taskbridge today|任务执行控制面" README.md docs`。

## 2. CLI Output Contract

- [x] 2.1 Owner: `cli/taskbridge`; Lane: output; Depends on: 0.2, 0.3; Scope: 设计或复用 command projection，字段覆盖 status、facts、actions、evidence、confidence、data、error；不要从中文输出反解析机器输出。复杂 projection 字段取舍写中文注释。Verify: unit tests 覆盖 projection nil/empty/error paths。
- [x] 2.2 Owner: `cli/taskbridge`; Lane: output; Depends on: 2.1; Scope: 为 `today/next/review` 增加 `--json` envelope，保留 `--format json` 兼容输出。Verify: stdout 解析为单个 JSON object，stderr 不含 payload 必需字段。
- [x] 2.3 Owner: `cli/taskbridge`; Lane: output; Depends on: 2.1; Scope: 为 `today/next/review` 增加 `--agent` key=value 输出，必含 `spec_version`、`mode=agent`、`command`、`status`，不输出中文段落、表格或 ANSI。Verify: agent mode contract tests。
- [x] 2.4 Owner: `cli/taskbridge`; Lane: output; Depends on: 2.2, 2.3; Scope: 统一输出错误 envelope 和 stdout/stderr 分流。marshal error、invalid provider、store init error 必须返回 command error；机器 stdout 只保留机器对象。Verify: `go test ./cmd/... -run 'OutputContract|StdoutStderr|JSONEnvelope|AgentMode' -count=1`。

## 3. Action Execution Audit

- [x] 3.1 Owner: `cli/taskbridge`; Lane: audit; Depends on: 0.3; Scope: 新增 `internal/actionaudit` 或等价 service，使用 TaskBridge storage path 写 CLI-authored receipt，字段包含 schema_version、session_id、command、dry_run、confirm、status、stats、operations、errors、redaction。复杂 receipt 写入和脱敏逻辑写中文注释。Verify: unit tests 覆盖 success/failure/redaction。
- [x] 3.2 Owner: `cli/taskbridge`; Lane: audit; Depends on: 3.1; Scope: 将 `review --apply-file` 和 `taskbridge agent execute` 统一到同一个 execution facade，dry-run 不写 task store，confirm 写入成功后记录操作结果。Verify: `go test ./internal/actionfile ./cmd -run 'ActionAudit|ReviewApply|AgentExecute' -count=1`。
- [x] 3.3 Owner: `cli/taskbridge`; Lane: audit; Depends on: 3.2; Scope: 写入失败时返回 failed/partial result 和非零 exit code，audit receipt 记录失败 action，成功计数只统计已持久化操作。Verify: fake failing store 测试覆盖 SaveTask 失败。
- [x] 3.4 Owner: `cli/taskbridge`; Lane: audit; Depends on: 3.1, 3.2; Scope: 暴露 receipt 查询入口，例如 `taskbridge audit show <session-id> --json` 或等价只读命令；不要求用户手写或修改 receipt JSON。Verify: process test 读取刚生成的 receipt。

## 4. Agent Contract Hardening

- [x] 4.1 Owner: `cli/taskbridge`; Lane: agent; Depends on: 2.4, 3.2; Scope: 修正 `taskbridge agent *` 错误路径：stdout 仍输出合法 JSON，失败返回非零 exit code，stderr 只写安全诊断。Verify: store init failed、invalid action file、unsupported schema 测试。
- [x] 4.2 Owner: `cli/taskbridge`; Lane: agent; Depends on: 4.1; Scope: 更新 `agent capabilities`，只声明已实现命令、schema version、危险 action、audit support 和 output behavior，不提前承诺 MCP 或远端写。Verify: JSON contract test 检查字段和排序稳定性。
- [x] 4.3 Owner: `cli/taskbridge`; Lane: agent; Depends on: 4.1; Scope: 更新 `agent schemas --json`，输出 validator-friendly schema index；至少覆盖 `taskbridge.agent-result.v1`、`taskbridge.actions.v1`、control-plane envelope。Verify: 用 Go JSON schema validator 或项目内等价校验 fixture。
- [x] 4.4 Owner: `cli/taskbridge`; Lane: agent; Depends on: 4.1, 4.2, 4.3; Scope: 更新 `docs/agent-contract.md`，写清 exit code、confirmation-required、audit receipt、schema index 和 MCP adapter 延期边界。Verify: `rg -n "exit code|audit|requires_confirmation|agent schemas" docs/agent-contract.md`。

## 5. Integration Evidence and Command Harness

- [x] 5.1 Owner: `cli/taskbridge`; Lane: evidence; Depends on: 0.2; Scope: 决定命令级 harness，优先使用 `github.com/rogpeppe/go-internal/testscript`；如果沿用 Go test harness，在测试注释或 design closeout 中说明覆盖边界。Verify: harness 能运行真实 `taskbridge` 二进制或 test main。
- [x] 5.2 Owner: `cli/taskbridge`; Lane: evidence; Depends on: 5.1; Scope: 新增 `task test:integration`，每次运行写 `temp/integration-test-runs/<run-id>/summary.json`、`command.txt`、`stdout.log`、`stderr.log`、`env.json` 和 `artifacts/`；失败保留 evidence 并返回原 exit code。Verify: 人为失败用例或 wrapper unit test 覆盖。
- [x] 5.3 Owner: `cli/taskbridge`; Lane: evidence; Depends on: 5.2; Scope: 对 env/stdout/stderr/artifacts 做默认脱敏，覆盖 token、Authorization、raw prompt、hidden system prompt、Provider payload、完整 chain-of-thought 和 private tool args。Verify: redaction tests 使用 fixture 字符串断言敏感值不落盘。
- [x] 5.4 Owner: `cli/taskbridge`; Lane: evidence; Depends on: 2.4, 3.3, 4.1, 5.2; Scope: 增加 process e2e 覆盖 `demo today --json`、`today --json`、`next --agent`、`review --apply-file --dry-run`、`agent today`、`agent execute --dry-run`。Verify: `task test:integration` 通过并生成 evidence。

## 6. Documentation and Compatibility

- [x] 6.1 Owner: `cli/taskbridge`; Lane: docs; Depends on: 1.4, 2.4, 3.4, 4.4; Scope: 更新 README，将一句话定位改为"人和 Agent 共用的本地任务执行控制面"，给出真实命令示例，不展示本地执行包装器或 agent-only 前缀。
- [x] 6.2 Owner: `cli/taskbridge`; Lane: docs; Depends on: 6.1; Scope: 更新 `docs/task-control-plane-roadmap.md`，标记本 change 只补 Phase 0/1/2/4 的 hardening，不新增 MCP adapter、双向同步或 Provider。
- [x] 6.3 Owner: `cli/taskbridge`; Lane: docs; Depends on: 5.2; Scope: 更新 `AGENTS.md` 和 Taskfile 说明，把 `task test:integration` 纳入涉及 CLI/process e2e/golden 的完成标准。

## 7. Verification and Closeout

- [x] 7.1 Owner: `cli/taskbridge`; Lane: verification; Depends on: 1.4, 2.4, 3.4, 4.4, 5.4, 6.3; Scope: 运行 Go 验证。Commands: `gofmt -w` 涉及文件、`go test ./...`、`go test -race ./internal/storage/... ./internal/sync/... ./internal/projectservice/...`、`go vet ./...`、`go build ./...`、`gofmt -l .`、`git diff --check`。
- [x] 7.2 Owner: `cli/taskbridge`; Lane: verification; Depends on: 7.1; Scope: 运行集成 evidence 验证。Commands: `task test:integration`、`find temp/integration-test-runs -maxdepth 2 -type f | sort`，确认最新 run 含 required files 且脱敏。
- [x] 7.3 Owner: `cli/taskbridge`; Lane: verification; Depends on: 7.2; Scope: 运行 OpenSpec 验证。Command: `openspec validate taskbridge-control-plane-hardening --strict`。
- [x] 7.4 Owner: `cli/taskbridge`; Lane: verification; Depends on: 7.3; Scope: Closeout 记录证据、兼容风险和 deferred items；如要实现远端 Provider audit、MCP adapter、双向 sync conflict resolver 或项目自动 adjust，另开 change。

## Parallel Lanes

- `entry`: 1.1 到 1.4，优先解决用户两分钟内看到价值的问题。
- `output`: 2.1 到 2.4，依赖质量重构中的 shared output helper，优先保证机器 stdout 稳定。
- `audit`: 3.1 到 3.4，锁定 action 写入可信边界。
- `agent`: 4.1 到 4.4，依赖 output 和 audit，收口 agent 可调用性。
- `evidence`: 5.1 到 5.4，给命令级合同和失败复验提供本地证据。
- `docs`: 6.1 到 6.3，等接口和命令稳定后更新。
- `verification`: 7.1 到 7.4，所有 lanes 收口后执行。

## Acceptance

- 新用户无需 Provider 凭证即可运行 `taskbridge demo today --json` 并看到可解析的每日工作台。
- `taskbridge today --json` 输出 AI Native CLI envelope；`taskbridge next --agent` 输出稳定 key=value；legacy `--format json` 仍可解析。
- `review --apply-file` 和 `agent execute` 的 dry-run 不修改任务；confirm 写入生成 action audit receipt。
- agent 错误路径 stdout 是 JSON，失败 exit code 非零；`requires_confirmation=true` 路径不静默写入。
- `agent capabilities` 和 `agent schemas --json` 不宣称未实现能力，schema 可被测试校验。
- `task test:integration` 成功和失败都会在 `temp/integration-test-runs/<run-id>/` 写 required evidence，并脱敏敏感内容。
- `go test ./...`、`go build ./...`、`task test:integration`、`openspec validate taskbridge-control-plane-hardening --strict` 通过。

## Failure Recheck

- 如果 demo path 失败，先确认是否复用现有 mock service，再检查是否意外初始化真实 Provider 或 task store。
- 如果 JSON/agent contract 失败，先检查 stdout 是否混入中文提示、ANSI、日志或 cobra error text，再检查 projection 是否被多个 renderer 分叉。
- 如果 action audit 失败，先检查 receipt 是否由 service 写入、失败路径是否 finish receipt，再检查 redaction policy。
- 如果 agent exit contract 失败，先检查 `printAgent` 是否吞掉错误返回 nil，再确认 cobra 错误输出是否污染 stdout。
- 如果 integration evidence 失败，先检查 wrapper 是否始终写 summary/stdout/stderr/env/command/artifacts，再确认原 exit code 是否保留。
