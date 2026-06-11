# Tasks: TaskBridge CLI 输出美化与合同收敛

## 0. Preflight and Active Change Alignment

- [x] 0.1 Owner: `cli/taskbridge`; Lane: sequential; Scope: 对齐 active OpenSpec changes：`taskbridge-code-quality-refactor`、`taskbridge-control-plane-hardening`、`english-cli-output-contract`。Acceptance: 明确 shared renderer、control-plane projection、English CLI chrome 的唯一归属，避免重复实现。Failure recheck: 如果多个 change 定义同一 projection，合并到一个 `internal/clioutput` 真源或写清 adapter 边界。
- [x] 0.2 Owner: `cli/taskbridge`; Lane: sequential; Depends on: 0.1; Scope: 审计整个 Cobra tree 的输出入口：root/help/error、analyze、list/lists、provider/auth、task、sync、doctor/quickstart、today/next/inbox/review、project、governance、config/version、serve。Acceptance: 每个命令族被归类为 analysis stats panel、result table、detail card、governance report、write receipt、decision summary、long-running/service 或 explain report。Failure recheck: 如果发现命令直接 `fmt.Println` 输出 machine payload，列入迁移任务而不是继续手写。
- [x] 0.3 Owner: `cli/taskbridge`; Lane: sequential; Depends on: 0.2; Scope: 采集当前输出基线。Commands: `go test ./cmd/... -run 'Output|Provider|Auth|List|Analyze|Config|Version' -count=1`，并用临时 `--storage-path` smoke `taskbridge analyze priority`、`taskbridge provider list`、`taskbridge config show --format json`。Acceptance: 记录 stdout/stderr、JSON 污染、宽度错位、颜色行为、legacy `--format json` 兼容点。Failure recheck: 如果命令依赖真实凭证，使用空 store、mock 数据或已有 test helper。

## 1. Renderer Foundation

- [x] 1.1 Owner: `cli/taskbridge`; Lane: renderer; Depends on: 0.1; Scope: 新增或收敛 `internal/clioutput` projection 类型，字段覆盖 `spec_version`、`mode`、`command`、`status`、`summary`、`facts`、`actions`、`evidence`、`confidence`、`data`、`error`、human-only preview/table/stat panels/risks/hint。Acceptance: unit tests 覆盖空 command、默认 status、error projection、action filtering、stable facts ordering。Failure recheck: 如果已有 shared helper 可复用，不新建平行类型。
- [x] 1.2 Owner: `cli/taskbridge`; Lane: renderer; Depends on: 1.1; Scope: 实现 JSON envelope renderer 和 agent key=value renderer。Acceptance: `--json` stdout 是单个 object；`--agent` 必含 `spec_version`、`mode=agent`、`command`、`status`，facts/actions/evidence 排序稳定。Failure recheck: 如果 JSON 测试失败，先检查 stdout 是否混入日志、提示或 ANSI。
- [x] 1.3 Owner: `cli/taskbridge`; Lane: renderer; Depends on: 1.1; Scope: 实现 human renderer primitives：stats panel、result table、detail card、governance report、write receipt、decision summary、explain report、empty state、hint。Acceptance: 默认 human 输出 English、短、可扫读；推荐下一步最多一条；颜色不是唯一语义。Failure recheck: 如果输出过长，先把大列表移到 `--json` 或专门 list/detail 命令。
- [x] 1.4 Owner: `cli/taskbridge`; Lane: renderer; Depends on: 1.2, 1.3; Scope: 增加 display width、truncate、padding、ASCII/Unicode fallback、`NO_COLOR` 和 root `--color auto|always|never`。Acceptance: CJK、emoji、长英文、长路径在表格和 stats panel 中不破坏列对齐；machine modes 永远无 ANSI。Failure recheck: 如果颜色进入 machine stdout，先检查 renderer mode gate 和 lipgloss color profile 设置。
- [x] 1.5 Owner: `cli/taskbridge`; Lane: renderer; Depends on: 1.2; Scope: 实现 redaction helper，覆盖 token、secret、password、Authorization、cookie、raw prompt、provider payload、hidden system prompt、private tool args。Acceptance: renderer unit tests 证明 human、json、agent、events、explain 均脱敏。Failure recheck: 如果测试只覆盖 stdout，补 sidecar/evidence/golden artifact 路径。

## 2. Output Control Wiring

- [x] 2.1 Owner: `cli/taskbridge`; Lane: wiring; Depends on: 1.1; Scope: 改造 `cmd/output_control.go`，让命令层通过 projection renderer 输出，不再从 human 文本构造 JSON。Acceptance: `printStructured` 或替代 helper 能显式处理 summary/json/agent/events/explain/legacy format。Failure recheck: 如果旧命令仍需要 legacy payload，保留命令级 adapter 并加兼容测试。
- [x] 2.2 Owner: `cli/taskbridge`; Lane: wiring; Depends on: 2.1; Scope: 在 `cmd/root.go` 增加全局或约定的 `--json`、`--agent`、`--events`、`--explain`、`--color` 处理策略；不能破坏已有命令专属 flag。Acceptance: help text 只展示用户可运行真实命令，machine mode 不受 TTY/颜色影响。Failure recheck: 如果 flag 命名冲突，先在具体命令局部接入，再规划全局化。
- [x] 2.3 Owner: `cli/taskbridge`; Lane: wiring; Depends on: 2.1; Scope: 标准化错误输出。Human mode 错误走 stderr 并返回 exit code；machine mode 输出 envelope/error object，stderr 只放安全诊断。Acceptance: invalid provider、storage init failure、marshal failure 均有 stable `error.code` 和 English suggestion。Failure recheck: 如果 Cobra 默认错误污染 stdout，检查 `SilenceErrors`、`SilenceUsage` 和 command return path。

## 3. Analyze Stats Panel Migration

- [x] 3.1 Owner: `cli/taskbridge`; Lane: analyze; Depends on: 1.3, 2.1; Scope: 迁移 `cmd/analyze.go` 的 `priority` text output 到 stats panel。Acceptance: `taskbridge analyze priority` 输出 title、boxed rows `Urgent/High/Medium/Low/No priority`、counts、percentages、`Total | Active | Completed` footer；空 store 输出稳定。Failure recheck: 如果 emoji 宽度错位，修复 stats panel width 计算，不在命令里手补空格。
- [x] 3.2 Owner: `cli/taskbridge`; Lane: analyze; Depends on: 3.1; Scope: 迁移 `analyze quadrant/time/trend/report` 到 shared analysis renderer。Acceptance: 所有 analyze 子命令默认 human 输出来自 stats panel/detail primitives，不再手写 box/table；`--format json` legacy payload 兼容。Failure recheck: 如果 report payload 是 map，先定义 command-local projection adapter，不改变业务分析结构。
- [x] 3.3 Owner: `cli/taskbridge`; Lane: analyze; Depends on: 3.2; Scope: 补 analyze renderer 和 command contract tests。Commands: `go test ./cmd/... -run 'Analyze|Output|Width|JSON' -count=1`。Acceptance: zero-data、non-zero percentages、NO_COLOR、legacy JSON parseability、machine stdout no ANSI 均覆盖。Failure recheck: 不测试 ANSI 颜色值；只测试语义、parseability 和 display width invariants。

## 4. Low-risk Identity and Config Migration

- [x] 4.1 Owner: `cli/taskbridge`; Lane: low-risk; Depends on: 2.1; Scope: 迁移 `cmd/version.go`。Acceptance: `taskbridge version` 默认是 detail card；`taskbridge version --json` 是 envelope 或记录为兼容 alias；`--agent` 输出低 token facts。Failure recheck: 如果 build metadata 为空，不伪造值，facts 标记 `unknown`。
- [x] 4.2 Owner: `cli/taskbridge`; Lane: low-risk; Depends on: 2.1; Scope: 迁移 `cmd/config.go` 的 `show/get/validate`。Acceptance: `taskbridge config show --format json` stdout 不再追加人类文本；敏感字段默认脱敏，配置来源进入 `facts.source` 或 stderr。Failure recheck: 如果 JSON 后追加提示，先修复 renderer gate。
- [x] 4.3 Owner: `cli/taskbridge`; Lane: low-risk; Depends on: 4.1, 4.2; Scope: 为 version/config 补 contract tests。Commands: `go test ./cmd/... -run 'Version|Config.*Output|JSONEnvelope|AgentMode' -count=1`。Acceptance: JSON 可解析、agent key=value 稳定、human 非 JSON dump。Failure recheck: 如果测试依赖全局 flag 状态，重置 Cobra command 和 package globals。

## 5. Browsing and Provider/Auth Migration

- [x] 5.1 Owner: `cli/taskbridge`; Lane: browsing; Depends on: 1.4, 2.1; Scope: 迁移 `provider list/info/test` 到 shared renderer，保留 provider catalog 为唯一真源。Acceptance: 默认表格 CJK 对齐；`provider info` 使用 detail/能力表；`--json` envelope 包含 provider data；`--agent` 不输出表格或 localized prose。Failure recheck: 如果 provider 名称列表漂移，先检查 `provider.GetAllProviderNames()` 和 definition metadata。
- [x] 5.2 Owner: `cli/taskbridge`; Lane: browsing; Depends on: 1.4, 2.1; Scope: 迁移 `auth status/show/whoami/login/logout` 的非交互结果输出，交互提示保持 stderr/stdin 边界清晰。Acceptance: 默认输出是 governance/detail/write receipt；认证错误不会被颜色或 emoji 隐藏；machine mode 不泄露 token path 中敏感内容。Failure recheck: 如果 auth 状态读取失败但 exit 0，先接入 `commandError` 并补非零 exit 测试。
- [x] 5.3 Owner: `cli/taskbridge`; Lane: browsing; Depends on: 2.1; Scope: 迁移 `lists` 和 `list` 浏览输出，与 `internal/taskoutput` 对齐。Acceptance: result table 只展示主结果；空结果给一条下一步；分页 facts 进入 JSON/agent，不污染 table。Failure recheck: 如果 task renderer 与 shared renderer 都算宽度，保留一个宽度真源。
- [x] 5.4 Owner: `cli/taskbridge`; Lane: browsing; Depends on: 5.1, 5.2, 5.3; Scope: 补 provider/auth/list/lists contract tests。Commands: `go test ./cmd/... ./internal/taskoutput/... ./pkg/ui/... -run 'Provider|Auth|List|Lists|Output|Width' -count=1`。Acceptance: CJK 表格对齐、`NO_COLOR=1` 可读、machine stdout 无 ANSI。Failure recheck: 如果 golden 在不同终端宽度不稳定，固定测试 width 并把 terminal width 注入 renderer。

## 6. Task, Sync, Control-plane, Project, and Governance Tree

- [x] 6.1 Owner: `cli/taskbridge`; Lane: task-tree; Depends on: 2.1, 5.3; Scope: 迁移 `task show/add/edit/done/undo/delete`。Acceptance: read/detail 使用 detail card；写入结果使用 write receipt；删除/危险操作保留确认 gate；machine modes parseable。Failure recheck: 不通过弱化确认提示来修复测试，改用 explicit confirm/dry-run test path。
- [x] 6.2 Owner: `cli/taskbridge`; Lane: sync-tree; Depends on: 2.1; Scope: 迁移 `sync pull/push/bidirectional/status/diff/conflicts/resolve/backup/watch`。Acceptance: read/preview 输出描述比较或写入结果；长任务只在适合时支持 `--events`；dry-run 不写本地、不调用远端写 API。Failure recheck: 如果 events 变成空流，拒绝给非长任务加 `--events`。
- [x] 6.3 Owner: `cli/taskbridge`; Lane: control-plane; Depends on: 2.1; Scope: 对齐 `today/next/inbox/review/doctor/quickstart` 与 `taskbridge-control-plane-hardening` projection，不重复实现控制面合同。Acceptance: 默认输出视觉与 shared renderer 一致；`--json`、`--agent` 字段来自同一 projection。Failure recheck: 如果两个 change 都定义 projection，合并到 `internal/clioutput` 或明确 adapter 边界。
- [x] 6.4 Owner: `cli/taskbridge`; Lane: project-governance; Depends on: 2.1; Scope: 迁移 `project *` 和 `governance *`。Acceptance: planning/review/adjust/achievement 使用 decision summary 或 governance report；写入/确认结果使用 write receipt；dry-run/confirm 语义不变。Failure recheck: 如果 command currently dumps JSON by default, add compatibility adapter and tests before changing default.
- [x] 6.5 Owner: `cli/taskbridge`; Lane: service; Depends on: 2.1; Scope: 迁移 `serve` startup/status/shutdown text and token/sync progress boundaries。Acceptance: human progress is readable; logs/diagnostics go stderr; service mode does not emit machine-looking partial JSON on stdout. Failure recheck: if serve is long-running in tests, use command-level renderer tests and focused startup validation instead of sleeping.

## 7. Documentation and Migration Notes

- [x] 7.1 Owner: `cli/taskbridge`; Lane: docs; Depends on: 3.3, 5.4, 6.3; Scope: 更新 `docs/cli-output-agent-design.md`，把本 OpenSpec 的 CLI tree migration order、mode contract、analysis stats panel 和 GitPulse/skillctl 参考结论同步进去。Acceptance: 文档明确视觉美化只是 human renderer，不影响 machine modes。Failure recheck: 如果文档和实现不一致，先以 OpenSpec spec 和 tests 为准。
- [x] 7.2 Owner: `cli/taskbridge`; Lane: docs; Depends on: 7.1; Scope: 更新 `docs/cli-lipgloss-beautify.md`，标记为视觉组件参考或合并到输出合同文档，避免继续指导按命令手写 UI。Acceptance: 文档不再建议“所有命令直接使用 lipgloss 美化”作为主方案。Failure recheck: 如果仍需要保留历史设计，增加“superseded by OpenSpec change”说明。
- [x] 7.3 Owner: `cli/taskbridge`; Lane: docs; Depends on: 7.1; Scope: 更新 README 和 `docs/cli-design.md` 示例，命令示例必须是真实用户可运行命令。Acceptance: 示例包含 `taskbridge analyze priority`、`taskbridge provider list --agent`、`taskbridge config show --json`、`taskbridge today --explain` 等真实入口。Failure recheck: 搜索文档中是否出现本地 wrapper、alias 或 agent-only 前缀。

## 8. Verification and Closeout

- [x] 8.1 Owner: `cli/taskbridge`; Lane: verification; Depends on: 3.3, 4.3, 5.4, 6.5, 7.3; Scope: 运行 Go 验证。Commands: `gofmt -w` 涉及 Go 文件、`task test`、`task build`。Acceptance: 命令通过，输出无格式错误。Failure recheck: 如果全量测试失败，先用 focused contract test 定位 stdout/stderr 或 global flag 状态。Evidence: `gofmt -w cmd internal/clioutput internal/taskoutput`; `task test` passed; `task build` passed; PTY smoke `go run . --storage-path "$tmp/storage" analyze priority` rendered the requested stats panel.
- [x] 8.2 Owner: `cli/taskbridge`; Lane: verification; Depends on: 8.1; Scope: 涉及命令面或发布配置时运行 release guard。Commands: `task release:check`。Acceptance: release 检查通过，buildinfo ldflags 不漂移。Failure recheck: 如果 release check 与输出无关失败，记录 blocker 并保留 focused verification evidence。Evidence: `task release:check` passed (`goreleaser check`, `.goreleaser.yaml` validated).
- [x] 8.3 Owner: `cli/taskbridge`; Lane: verification; Depends on: 8.2; Scope: 运行 OpenSpec 验证。Command: `openspec validate taskbridge-cli-output-polish --strict`。Acceptance: OpenSpec 通过 strict 验证。Failure recheck: 如果 spec 格式失败，先检查 `ADDED Requirements`、Scenario bullet 和 capability 路径。Evidence: OpenSpec validation passed.
- [x] 8.4 Owner: `cli/taskbridge`; Lane: verification; Depends on: 8.3; Scope: Closeout 记录证据、兼容风险和 per-command migration status。Acceptance: 整个 CLI tree 有统一维护矩阵；未完成的命令族必须有明确 deferred reason 和 follow-up OpenSpec task，不得静默遗漏。Failure recheck: 如果范围膨胀到 Provider/TUI/MCP behavior change，另开 OpenSpec change。Evidence: migration matrix in this tasks file is fully checked for renderer foundation, analyze, low-risk commands, browsing commands, remaining CLI tree, docs, and verification.

## Parallel Lanes

- `renderer`: 1.1 到 1.5，先建立 projection、renderer、宽度、颜色和脱敏真源。
- `wiring`: 2.1 到 2.3，把 Cobra 输出入口接入 renderer。
- `analyze`: 3.1 到 3.3，用 `taskbridge analyze priority` 首先验证 stats panel。
- `low-risk`: 4.1 到 4.3，迁移 version/config，降低回归风险。
- `browsing`: 5.1 到 5.4，迁移 provider/auth/list/lists。
- `task-tree`、`sync-tree`、`control-plane`、`project-governance`、`service`: 6.1 到 6.5，覆盖剩余 CLI tree。
- `docs`: 7.1 到 7.3，接口稳定后同步文档。
- `verification`: 8.1 到 8.4，所有 lanes 收口后执行。

## Acceptance

- `taskbridge analyze priority` 默认输出 stats panel，包含五个优先级、counts、percentages、`Total | Active | Completed` footer，空数据稳定。
- analyze/provider/auth/list/lists/task/sync/control-plane/project/governance/config/version/serve 等命令族都通过同一 output contract 维护，而不是各自手写输出风格。
- 默认 human CLI chrome 使用 English，可扫读，不依赖颜色表达含义；用户数据、第三方 payload 和已有 machine fields 不被翻译。
- `taskbridge config show --format json` 和新 `--json` 路径 stdout 均为单个可解析 JSON object，不追加人类提示。
- `taskbridge version --agent`、`taskbridge provider list --agent`、`taskbridge auth status --agent` 输出稳定 key=value，必含 required keys。
- `NO_COLOR=1` 和 `--color never` 下 human 输出可读；`--color always` 不会让 machine mode 输出 ANSI。
- 失败路径有稳定 `error.code` 和 English 修复建议；machine mode stdout 不混入 Cobra usage、日志、进度或 human prose。
- `task test`、`task build`、需要时 `task release:check` 和 `openspec validate taskbridge-cli-output-polish --strict` 通过。

## Failure Recheck

- 如果 human 输出错位，先检查 display width 计算和测试注入 width，再检查 emoji/CJK 截断。
- 如果 JSON 解析失败，先检查 stdout 是否有提示、日志、颜色或多对象输出。
- 如果 agent 输出不稳定，先检查 facts/actions/evidence 排序和 quote 规则。
- 如果 legacy `--format json` 破坏脚本，先恢复兼容 payload，再把 envelope 放到显式 `--json`。
- 如果某个 CLI subtree 没有迁移，更新维护矩阵和任务，不用“已统一”覆盖实际遗漏。
- 如果 docs 示例不真实，直接运行示例命令或改成已实现命令。
