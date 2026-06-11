# Proposal: TaskBridge CLI 输出美化与合同收敛

## Why

TaskBridge 已经有两条输出改造线索：`docs/cli-lipgloss-beautify.md` 解决表格和中文宽度，但偏视觉；`docs/cli-output-agent-design.md` 已提出 AI-native 输出合同，但还没有进入可执行 OpenSpec。当前代码也已经分裂成多条输出路径：`cmd/output_control.go` 的 `printStructured`、`cmd/project.go` 的 `printResult`、`pkg/output.RenderTasks`、`pkg/ui` 组件、以及 provider/auth/config/version 中的手写 `fmt.Println`。

如果只做“更好看”的表格，会继续把 human 文本、JSON、Agent 输出和错误处理绑在一起。正确方向应参考 GitPulse 和 skillctl：先统一 command projection，再从同一投影渲染默认中文摘要、`--json` envelope、`--agent` key=value、`--events` NDJSON 和 `--explain` 审查摘要；视觉美化只属于默认 human renderer，不能污染机器 stdout。

## What Changes

- 建立 TaskBridge 全 CLI tree 的统一 output projection 和 renderer contract，覆盖 root command、help/error、`analyze`、task browsing、provider/auth、sync、control-plane、project/governance、service/runtime 输出，以及 `summary`、`json`、`agent`、`events`、`explain` 和 legacy `--format json`。
- 收敛默认 human 输出类型：分析类使用 stats panel；列表/搜索类优先 table-first；治理/详情/写入预览类使用 `Status`、`Highlights`、`Facts`、`Preview`、`Risks`、`Recommended next step` 固定区块。
- 统一 panel/table/card 的 display width、截断、分页提示、颜色控制和 `NO_COLOR` 行为；机器模式永远无 ANSI、无表格边框、无本地化 prose。
- 迁移 analyze/provider/auth/list/lists/task/sync/config/version/doctor/today/next/inbox/review/project/governance/serve 等命令树，和既有 `taskbridge-control-plane-hardening`、`english-cli-output-contract` 输出合同保持同源。
- 固化 contract tests：默认输出不是 JSON dump；`--json` 是单个 envelope object；`--agent` 是稳定 key=value；错误路径 stdout/stderr 分离；颜色和宽度不会破坏解析；`analyze priority` 等 stats panel 在空数据和非空数据下稳定。

## Goals

- 默认 CLI 输出更短、更稳、更易扫读，中文和英文混排不再错位。
- 人类输出、脚本输出、Agent 输出和审查解释来自同一份 projection。
- `--format json` 兼容路径有明确迁移策略，新文档推荐显式 `--json` 和 `--agent`。
- 所有推荐下一步都是真实可运行命令，不出现本地 agent wrapper、alias 或调试前缀。
- 输出合同测试覆盖成功、partial、failed 和 sensitive redaction 路径。

## Non-Goals

- 不在本 change 中新增 Provider、同步协议、TUI 重写、MCP adapter 或后台服务。
- 不把所有命令压进一次不可审查的单个 patch；本 change 维护整个 CLI tree 的统一目标，但实现按命令族分 lane 迁移并逐族验证。
- 不改变 `taskbridge agent *` 已有 `taskbridge.agent-result.v1` 默认 JSON 安全 API；普通命令的 `--agent` 使用 key=value。
- 不为了美化引入复杂交互组件、全屏 UI 或依赖颜色表达语义。
- 不要求用户或 agent 手写 JSON/YAML/JSONL metadata；结构化 evidence 和 receipt 仍由 CLI/application service 生成。

## Impact

- 主要代码：`cmd/root.go`、`cmd/output_control.go`、`cmd/analyze.go`、`cmd/provider.go`、`cmd/auth.go`、`cmd/list.go`、`cmd/lists.go`、`cmd/task.go`、`cmd/sync.go`、`cmd/sync_control.go`、`cmd/controlplane.go`、`cmd/project*.go`、`cmd/governance.go`、`cmd/doctor.go`、`cmd/config.go`、`cmd/version.go`、`cmd/serve.go`、`internal/taskoutput/`、`internal/controlplane/render/`、`pkg/ui/`、`pkg/output/`，可能新增 `internal/clioutput/`。
- 关联 change：`taskbridge-code-quality-refactor` 已在收敛 `pkg/output` 和 shared output helper；`taskbridge-control-plane-hardening` 已覆盖控制面和 agent/write path；`english-cli-output-contract` 要求 CLI chrome 默认 English。实现时必须先对齐这些 active changes，避免重复建输出层或继续引入中文 chrome。
- 文档：`README.md`、`docs/cli-design.md`、`docs/cli-output-agent-design.md`、`docs/cli-lipgloss-beautify.md` 和 `docs/agent-contract.md` 需要在接口稳定后同步。
- 测试：新增 renderer unit tests、command contract tests、process/e2e evidence 覆盖和 `NO_COLOR`/`--color`/stats panel 断言。

## Compatibility

- `--json` 是新推荐机器入口，输出 AI Native CLI envelope。
- `--format json` 短期保留现有命令级 payload 或兼容 envelope；每个命令迁移时必须在任务中说明具体策略。
- `--format ai` 如历史存在，可作为 `--agent` alias；新文档只推荐 `--agent`。
- pipe/quiet 自动机器输出短期保留为 legacy auto-machine mode，但新脚本不应依赖 pipe 检测。
- 默认 human 输出可以改变排版，但不能改变写入行为、确认门禁、dry-run 语义或 machine mode 字段名。

## References

- GitPulse: `internal/output/projection`、`cmd/gitpulse/output_contract.go`、`docs/interfaces/ai-native-cli-output-contract.md`。
- skillctl: `internal/output/formatter.go`、`internal/output/width.go`、`docs/cli-output.md`。
- TaskBridge: `docs/cli-output-agent-design.md`、`docs/cli-lipgloss-beautify.md`、`pkg/ui`、`pkg/output`、`cmd/output_control.go`。
