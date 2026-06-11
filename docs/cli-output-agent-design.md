# TaskBridge CLI 输出与 Agent 体验设计

更新时间：2026-06-08

## 1. 设计判断

TaskBridge 的 CLI 输出不应继续按命令各自美化。下一阶段应先统一输出契约，再统一视觉组件。

核心判断：

- `today`、`next`、`review`、`doctor` 是人类日常入口，默认输出必须短、可扫读、能推动下一步动作。
- `agent *` 是安全自动化入口，stdout 必须继续保持 `taskbridge.agent-result.v1` JSON。
- 普通命令也需要 Agent 友好输出，但不应复用人类文本；应提供稳定低 token 的 `--agent` key=value 模式。
- `--json` 应使用稳定 envelope，不能把命令私有字段散落在 top-level。
- 视觉美化只服务 human summary，不能影响 `--json`、`--agent`、`--events` 和错误契约。

一句话目标：

> 同一个命令结果只生成一次投影，再分别渲染给人、脚本、Agent 和审计流。

## 2. 当前问题

### 2.1 输出路径已经分叉

当前代码至少存在三类输出路径：

- `printStructured(format, value, renderText)`：用于 `doctor`、控制面、sync control、project execution。
- `printResult(value)`：用于 project、governance 等旧命令，默认仍偏 JSON dump。
- `printAgent(result)`：用于 `agent *`，保持专用 JSON envelope。

这说明项目已经具备统一入口，但还没有统一 projection 类型。

### 2.2 自动 JSON 与默认 human 的边界不清

`wantsJSON()` 当前把 `--format json` 和 `IsQuietMode()` 合并判断。非 TTY 或管道环境会自动变成 JSON，这对脚本友好，但会让默认输出语义不够显式。

建议保留兼容行为，但在新契约中明确：

- 显式 `--json` 永远输出 envelope JSON。
- 显式 `--format json` 作为兼容别名。
- `--quiet` 或 pipe 自动 JSON 属于兼容策略，文档中标记为 legacy auto-machine mode。
- 新脚本和 Agent 文档只推荐 `--json` 或 `--agent`，不依赖 pipe 检测。

### 2.3 文档和命令面不一致

路线文档使用 `taskbridge demo today`，当前实现主要是 `taskbridge today --mock`。这影响新用户路径和 smoke test。

建议新增 `demo today` 作为别名命令，内部复用 `today --mock` 的服务能力。

### 2.4 美化计划偏视觉，缺少契约层

`docs/cli-lipgloss-beautify.md` 已解决表格和中文宽度问题，但它没有定义 machine 输出、Agent 输出、错误 envelope、stdout/stderr 分离和兼容迁移。

视觉方案应下沉为 human renderer 的实现细节，而不是项目级输出设计的主轴。

## 3. 输出模式

TaskBridge 统一支持五种公共输出模式。当前实现以 `internal/clioutput` 为普通命令输出合同真源，视觉美化只属于 human renderer。

| 模式 | 入口 | 受众 | stdout 约束 |
| --- | --- | --- | --- |
| summary | 默认 | 人类操作者 | English short summary/table/stats panel，不是 JSON dump |
| json | `--json`；部分 legacy `--format json` 保留兼容 payload | 脚本、CI、SDK | 单个 JSON object；新路径使用 envelope |
| agent | `--agent` | Agent、shell glue | 稳定 key=value 行 |
| events | `--events` | 长任务自动化 | NDJSON 事件流；只给适合的长任务 |
| explain | `--explain` | 决策复核 | English conclusion/evidence/confidence/risks/next step 摘要

旧格式保留兼容：

- `--format text` 等价 summary。
- `--format json` 对已迁移命令可能仍是 legacy payload；新脚本优先使用显式 `--json` envelope。
- `list --format table|compact|tsv|markdown` 继续保留为数据浏览格式；全局 `--json`/`--agent` 不输出 human table。

## 4. 统一 Projection

内部投影类型落在 `internal/clioutput`。命令层只构建 projection 或 legacy adapter，再由 renderer 负责 summary/json/agent/events/explain。

```go
type Projection struct {
    SpecVersion string                 `json:"spec_version"`
    Mode        string                 `json:"mode"`
    Command     string                 `json:"command"`
    Status      string                 `json:"status"`
    Summary     string                 `json:"summary,omitempty"`
    Facts       map[string]interface{} `json:"facts,omitempty"`
    Actions     []Action               `json:"actions,omitempty"`
    Evidence    []string               `json:"evidence,omitempty"`
    Confidence  *float64               `json:"confidence,omitempty"`
    Data        interface{}            `json:"data,omitempty"`
    Error       *OutputError           `json:"error,omitempty"`
}

type Action struct {
    Name    string `json:"name"`
    Command string `json:"command"`
}

type OutputError struct {
    Code       string                 `json:"code"`
    Message    string                 `json:"message"`
    Suggestion string                 `json:"suggestion,omitempty"`
    Retryable  bool                   `json:"retryable,omitempty"`
    Details    map[string]interface{} `json:"details,omitempty"`
}
```

状态枚举使用：

- `success`
- `partial`
- `failed`

命令名使用点分格式：

- `doctor.check`
- `demo.today`
- `today.view`
- `next.list`
- `review.health`
- `sync.diff`
- `agent.today`

## 5. Human Summary 结构

默认输出只解决一个问题：用户现在该做什么。CLI chrome 默认 English；用户任务内容、Provider payload、第三方字段和值不被翻译。

统一结构：

```text
Status
Ready for today with 1 overdue risk.

Highlights
- P1 Finish TaskBridge output contract migration
- P2 Review overdue tasks and decide next action

Risks
- mock_overdue is 2 days overdue and needs a date or split decision

Recommended next step
taskbridge review
```

规则：

- 只展示一条主推荐下一步。
- 分析类命令使用 stats panel，例如 `taskbridge analyze priority`。
- 列表/搜索类命令 table-first，详情/治理/写入回执使用固定 section。
- 详细列表放 `--json` 或专门 detail/list 命令。
- 不依赖颜色表达含义；`NO_COLOR=1` 后仍可读。
- 命令、flag、schema key、provider id 和 machine protocol field 保持英文稳定。

## 6. Agent 输出

普通命令的 `--agent` 输出为 key=value：

```text
spec_version=1.0
mode=agent
command=today.view
status=partial
fact.must_do=3
fact.overdue=1
fact.at_risk=2
action.review="taskbridge review"
evidence.date=2026-06-08
```

规则：

- 必须包含 `spec_version`、`mode=agent`、`command`、`status`。
- key 使用英文稳定字段。
- value 单行输出；包含空格时加引号。
- 不输出 ANSI、表格、中文段落、raw provider payload、raw prompt 或推理链。

`taskbridge agent *` 继续保持现有 `taskbridge.agent-result.v1` JSON，不改成 key=value。它是安全执行 API，不是普通命令的 agent renderer。

## 7. JSON Envelope

普通命令 `--json` 输出：

```json
{
  "spec_version": "1.0",
  "mode": "json",
  "command": "today.view",
  "status": "partial",
  "summary": "今日可执行，有 1 个逾期风险",
  "facts": {
    "must_do": 3,
    "overdue": 1,
    "at_risk": 2
  },
  "actions": [
    {
      "name": "review",
      "command": "taskbridge review"
    }
  ],
  "data": {
    "schema": "taskbridge.today.v1",
    "date": "2026-06-08"
  }
}
```

现有命令私有 schema 放在 `data` 中，避免 top-level 持续膨胀。

兼容策略：

- 第一批命令可在 `--format json` 下保留旧 payload，但新增 `--json` 输出新 envelope。
- 或者统一切 envelope，但必须补测试并记录为 schema major 变更。
- 建议先采用第一种，降低脚本破坏面。

## 8. Explain 输出

`--explain` 用于复核判断，不暴露完整推理链。

格式：

```text
结论：建议先处理 mock_overdue，再进入今日重点任务。
证据：mock_overdue 已逾期 2 天；must_do=3；at_risk=2。
置信度：0.82
风险：推迟决策会继续污染 today 列表。
推荐下一步：taskbridge review
```

适用命令：

- `today`
- `next`
- `review`
- `project review`
- `sync diff`

不适用命令：

- 纯列表浏览，如 `provider list`、`lists`。
- 纯 CRUD 成功回执，除非包含策略判断。

## 9. Events 输出

`--events` 只用于长任务：

- `sync pull/push/diff`
- `agent execute`
- 后续 provider 批量写入
- 后台服务可观测命令

最小序列：

```json
{"type":"start","spec_version":"1.0","command":"sync.diff","seq":1}
{"type":"fact","key":"operations","value":3,"seq":2}
{"type":"end","status":"success","seq":3}
```

日志和提示继续写 stderr。

## 10. 命令迁移顺序

### Batch 1：体验地基

目标：新用户两分钟内看到价值。

- `doctor`
- `quickstart`
- 新增 `demo today`
- `today --mock` 保留为别名路径

交付：summary、`--json` envelope、`--agent`、错误 envelope 测试。

### Batch 2：日常控制面

目标：用户每天使用。

- `today`
- `next`
- `inbox`
- `review`

交付：summary、`--json` envelope、`--agent`、`--explain`。

### Batch 3：可信写入与审计

目标：写入前可预览，写入后可追踪。

- `sync diff`
- `sync audit`
- `review --apply-file`
- `agent execute`

交付：`--events`、audit evidence、dry-run/confirm 契约测试。

### Batch 4：项目执行闭环

目标：项目不止拆分，还进入每日推进。

- `project review`
- `project next`
- `project adjust`
- `project done`
- `project archive`

交付：`--explain`、project evidence、危险动作确认门禁。

### Batch 4.5：OpenSpec 工程任务控制面

目标：把当前仓库的 OpenSpec active changes 纳入现有 `today/next/review`，让工程变更任务成为每日执行的一等来源，同时不复制 OpenSpec CLI。

- `today` 自动包含 OpenSpec 区块
- `next --source openspec`
- `review --source openspec`
- `agent today` 包含 OpenSpec section

交付：只读扫描、`--json` envelope、`--agent` key=value、`--explain` 风险说明。详细查看、验证和归档仍推荐原生 `openspec show/status/validate/archive` 命令；不新增 `taskbridge openspec *` wrapper。

### Batch 5：浏览与配置命令

目标：完成风格统一，不打破已有列表工作流。

- `provider list/info`
- `auth status/show`
- `list`
- `lists`
- `config show`
- `version`

交付：human renderer 美化、JSON 兼容说明、必要时补 `--json` envelope。

## 11. 实现边界

CLI 层只做：

- 解析 flag。
- 构造 service input。
- 调用 service。
- 把 service result 转为 projection。
- 调用 renderer。

业务层不做：

- 拼接本地化输出文本。
- 依赖 ANSI/terminal 宽度。
- 为了输出格式改变行为。

Agent 和 MCP adapter 不做：

- 直接读写 `~/.taskbridge` 数据文件。
- 绕过 Provider 接口写远端。
- 没有 `--confirm` 时执行危险动作。

## 12. 测试策略

每个迁移命令至少覆盖：

- 默认 summary 不是 JSON dump。
- summary 有且只有一个主推荐下一步。
- `--json` stdout 是单个 JSON object，且 top-level envelope 字段完整。
- `--agent` stdout 只包含 key=value 行，必填 key 存在。
- `--explain` 不含 raw prompt、provider payload 或推理链。
- `--events` 是合法 NDJSON，包含 start/end 或 start/error。
- JSON stdout 不混入日志、进度条、人类提示。
- 错误路径也能输出稳定 error code 和 next action。
- `--dry-run` 不写本地存储，不调用远端写 API。

命令行为变更时建议加入临时二进制 smoke test：

```bash
tmp=$(mktemp -d)
go build -o "$tmp/taskbridge" .
"$tmp/taskbridge" --storage-path "$tmp/storage" doctor --json
"$tmp/taskbridge" --storage-path "$tmp/storage" demo today --json
"$tmp/taskbridge" --storage-path "$tmp/storage" today --mock --agent
```

## 13. 验收标准

Phase 0 完成时：

- `taskbridge demo today` 可运行，无需 Provider 认证。
- `doctor` 默认输出可读，`--json` 可解析，`--agent` 可被 shell 稳定消费。
- `quickstart` 永远输出一条主下一步。

Phase 1 完成时：

- `today` summary 可以在 20 行以内给出状态、重点、风险、下一步。
- `today --json` 和 `agent today` 字段语义一致，但 envelope 允许不同。
- `review` 输出 action file 建议，但不默认写入。

Phase 2 完成时：

- 所有写入前都有 dry-run 或 action file。
- 所有危险动作没有确认时返回需要确认。
- sync/audit 输出能说明比较了什么、准备写什么、实际写了什么。

## 14. Open Questions

1. 是否把 `--json` 作为新显式 flag，同时保留 `--format json`？推荐：是。
2. pipe 自动 JSON 是否保留？推荐：短期保留，文档标为兼容行为，新文档不推荐依赖。
3. 普通命令的 `--agent` 是否复用 `taskbridge.agent-result.v1`？推荐：不复用。普通命令使用 key=value，`agent *` 保持 JSON 安全 API。
4. 现有 `taskbridge.today.v1` 是否包进新 envelope 的 `data`？推荐：是，避免破坏命令私有 schema。
5. 是否把输出模块放 `pkg/output`？推荐：先在 `internal/output` 或 `cmd` 内收敛，稳定后再考虑公开包。
