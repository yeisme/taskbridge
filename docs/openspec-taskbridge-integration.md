# TaskBridge 与 OpenSpec 任务集成设计

更新时间：2026-06-08

## 1. CEO 判断

TaskBridge 后续应把 OpenSpec 作为一等任务信号接入，但不应新增一套 `taskbridge openspec *` 来复制 OpenSpec CLI。

原因：OpenSpec 里的 task 不只是待办事项，它同时承载了变更范围、依赖、验证命令、证据、验收标准和归档状态。TaskBridge 的价值不是替代 OpenSpec，而是把这些工程任务带进每日执行控制面。

一句话定位：

> OpenSpec 管变更事实源和变更命令，TaskBridge 管每日执行、提醒、聚焦和安全动作推荐。

## 2. 产品目标

用户已经更多使用 OpenSpec tasks 后，TaskBridge 的核心价值应升级为：

- 每天在 `taskbridge today` 看到 OpenSpec change 的下一步任务。
- 在 `taskbridge next` 中优先推荐 unblock 当前 change 的最小任务。
- 在 `taskbridge review` 中发现长期卡住、依赖未满足、缺少 evidence、接近 closeout 的 change。
- Agent 能通过 TaskBridge 读取 OpenSpec 执行上下文，但真实 OpenSpec 操作仍回到 `openspec` CLI。
- OpenSpec closeout 证据能出现在 TaskBridge audit/evidence 视图中。

非目标：

- 不重写或包一层 OpenSpec CLI。
- 不直接编辑 OpenSpec 内部 metadata 或 schema 文件。
- 不把所有 OpenSpec artifacts 同步到远端 Todo 平台。
- 不让 Agent 绕过 OpenSpec CLI 或 OpenSpec workflow 直接改结构化文件。

## 3. 责任边界

```text
┌──────────────────────────────┐
│ OpenSpec                     │
│ - proposal/design/tasks/spec │
│ - validate/archive/status    │
│ - change lifecycle           │
└──────────────┬───────────────┘
               │ read via CLI/files, write via openspec CLI where possible
               ▼
┌──────────────────────────────┐
│ TaskBridge OpenSpec Adapter  │
│ - scan active changes        │
│ - parse tasks.md             │
│ - derive executable refs     │
│ - surface risk/evidence      │
└──────────────┬───────────────┘
               │ projection
               ▼
┌──────────────────────────────┐
│ TaskBridge Control Plane     │
│ - today/next/review          │
│ - agent/json/explain output  │
│ - next command recommendation│
└──────────────────────────────┘
```

边界规则：

- OpenSpec 是 change lifecycle 的真源。
- TaskBridge 可以读取 `openspec list --json`、`openspec validate --json` 和本地 `tasks.md`。
- TaskBridge 不新增 OpenSpec 列表、查看、验证、归档的 wrapper 命令；这些仍使用原生 `openspec` CLI。
- TaskBridge 可以在输出中推荐 `openspec show/status/validate/archive` 等下一步命令。
- TaskBridge 不直接让 Agent 修改 `tasks.md`、`.openspec.yaml`、spec delta 或 archive 目录。

## 4. 数据映射

### 4.1 Change -> Project

OpenSpec change 映射为 TaskBridge project-like item。

| OpenSpec | TaskBridge |
| --- | --- |
| change name | project_id / source_raw_id |
| proposal.md title | project name |
| status | project status / risk |
| completedTasks / totalTasks | progress |
| lastModified | updated_at |
| archive state | archived/completed |

建议 source：

```text
source=openspec
source_raw_id=<change-name>
```

### 4.2 tasks.md line -> TaskRef

OpenSpec task 映射为控制面 `TaskRef`，不是立即写入普通 Todo store。

| OpenSpec task | TaskBridge TaskRef |
| --- | --- |
| `- [ ] 1.1 ...` | id=`openspec:<change>:1.1` |
| checkbox | status=`todo/completed` |
| Owner | owner custom field |
| Lane | lane / tag |
| Depends on | dependency list |
| Scope | title/description |
| Verify | evidence requirement |
| Acceptance | project-level evidence |

OpenSpec task 的推荐 title 应从 `Scope:` 提取；如果没有 `Scope:`，使用整行清理后的文本。

### 4.3 Verify -> Evidence Gate

`Verify:` 不应只是描述文本。TaskBridge 应把它作为完成门槛：

```json
{
  "task_id": "openspec:taskbridge-control-plane-hardening:1.1",
  "verify": "go test ./cmd/... -run 'DemoToday|ControlPlaneMock' -count=1",
  "evidence_required": true
}
```

推荐 OpenSpec task 时，如果存在 Verify 命令，TaskBridge 应把验证命令作为 evidence gate 展示出来，但不自动执行。

## 5. 命令设计

### 5.1 不新增 OpenSpec wrapper 命令

不设计以下命令：

```bash
taskbridge openspec list
taskbridge openspec show <change>
taskbridge openspec validate <change>
taskbridge openspec archive <change>
```

这些命令已经由 OpenSpec 自己负责。TaskBridge 如果再做一套，会增加学习成本、文档重复和行为分叉。

TaskBridge 只在现有控制面中消费 OpenSpec 信号：

```bash
taskbridge today
taskbridge next
taskbridge review
taskbridge agent today
```

### 5.2 接入通用控制面

```bash
taskbridge today
taskbridge next --source openspec
taskbridge review --source openspec
```

推荐默认策略：

- 如果当前目录存在 `openspec/changes` 且有 active changes，`today` 默认包含 OpenSpec 区块。
- `--source openspec` 只看 OpenSpec 来源的执行信号。
- `--exclude openspec` 可关闭 OpenSpec 区块。
- 详细变更查看、验证、归档都给出原生 `openspec` 下一步命令。

### 5.3 写入边界

第一阶段不设计 TaskBridge 写 OpenSpec 的命令。

原因：

- OpenSpec task 状态目前由 OpenSpec workflow 和 `tasks.md` 表达。
- TaskBridge 贸然提供 `complete/defer/archive` 会形成第二套 lifecycle。
- 如果未来确实需要写入，只能作为 action/audit 稳定后的单独 change，并且调用原生 OpenSpec 能力或受控 workflow。

## 6. today/next/review 体验

### 6.1 today 区块

默认 human summary：

```text
状态：OpenSpec 有 3 个活跃 change，1 个需要先解阻塞

OpenSpec 下一步：
- taskbridge-control-plane-hardening 1.1 实现 demo today
- taskbridge-code-quality-refactor 1.3 补 table/compact/markdown/tsv 契约测试

风险：
- taskbridge-control-plane-hardening 依赖 shared output helper 尚未收口

推荐下一步：
openspec show taskbridge-control-plane-hardening
```

### 6.2 next 排序

OpenSpec task 推荐排序：

1. 当前活跃 change 且无未完成 Depends on。
2. Lane 是 `entry`、`output`、`verification` 等阻塞后续工作的任务。
3. Verify 命令明确、可在本地执行。
4. 最近修改的 change 优先。
5. 接近完成的 change 优先于刚开始的 change。
6. 如果 task 需要用户决策或外部状态，标记为 blocked，不推荐直接执行。

### 6.3 review 风险

`review --source openspec` 应识别：

- active change 超过 N 天未修改。
- completedTasks=0 但 totalTasks 很多。
- 任务依赖指向不存在的 task id。
- Verify 命令缺失或不可执行。
- change 任务已完成但未运行 `openspec validate --strict`。
- change 可能可归档但仍留在 active。
- 与其他 active change 改同一输出 helper、命令或文档，存在合并冲突风险。

## 7. Agent 契约

普通命令 `--agent` 示例：

```text
spec_version=1.0
mode=agent
command=next.list
status=success
fact.source=openspec
fact.active_changes=3
fact.next_task_id=openspec:taskbridge-control-plane-hardening:1.1
fact.next_change=taskbridge-control-plane-hardening
action.show="openspec show taskbridge-control-plane-hardening"
action.validate="openspec validate taskbridge-control-plane-hardening --strict"
```

Agent JSON API 暂不新增 OpenSpec 专属命令：

```bash
taskbridge agent today
```

`agent today` 的 result 可以包含 OpenSpec section；真实变更操作继续由上层 Agent 调用 `openspec` CLI 或进入 OpenSpec workflow。

## 8. Output Envelope

`taskbridge next --source openspec --json`：

```json
{
  "spec_version": "1.0",
  "mode": "json",
  "command": "next.list",
  "status": "success",
  "summary": "建议推进 taskbridge-control-plane-hardening 1.1",
  "facts": {
    "active_changes": 3,
    "next_change": "taskbridge-control-plane-hardening",
    "next_task_id": "1.1"
  },
  "actions": [
    {
      "name": "show",
      "command": "openspec show taskbridge-control-plane-hardening"
    },
    {
      "name": "validate",
      "command": "openspec validate taskbridge-control-plane-hardening --strict"
    }
  ],
  "data": {
    "schema": "taskbridge.openspec-next.v1",
    "tasks": []
  }
}
```

## 9. 实现建议

### Phase A：只读扫描

新增 internal package：

```text
internal/openspec/
  scanner.go      # 调 openspec list --json，读取 changes
  taskparser.go   # 解析 tasks.md checkbox/owner/lane/depends/scope/verify
  projection.go   # 生成 change/task DTO
  ranker.go       # next/review 排序和风险判断
```

第一阶段只读，不写任何 OpenSpec 文件。

### Phase B：控制面接入

把 OpenSpec task projection 接入：

- `today` 的 sections 增加 `openspec_next`。
- `next --source openspec` 返回 OpenSpec TaskRef。
- `review --source openspec` 返回风险和 suggested actions。

### Phase C：evidence 关联

第一阶段不新增写入 action。后续如果要把验证证据接回 TaskBridge audit，可以只记录证据引用：

```text
validate_openspec_change
record_openspec_evidence
```

这些 action 也不直接改 OpenSpec 文件，只记录命令、exit code、evidence path 和 redaction 状态。

## 10. 迁移优先级

建议先做最小可用闭环：

```bash
taskbridge today
taskbridge next --source openspec
taskbridge review --source openspec
openspec show taskbridge-control-plane-hardening
```

成功标准：

- 能识别 3 个 active changes。
- 能解析 `tasks.md` 的 checkbox、owner、lane、depends、scope、verify。
- 能推荐一个未完成且依赖满足的 next task。
- 不新增 `taskbridge openspec *` wrapper 命令。
- 不修改任何 OpenSpec 文件。
- `--json` 和 `--agent` 契约稳定。

第二步再做：

```bash
openspec validate <change> --json
taskbridge review --source openspec
```

第三步再评估是否需要 evidence 记录，不默认做 confirmed write。

## 11. Open Questions

1. OpenSpec task 完成是否允许 TaskBridge 改 `tasks.md` checkbox？推荐：短期不允许。
2. OpenSpec 是否默认进入 `today`？推荐：当前目录有 active changes 时默认进入；全局 TaskBridge 工作区可配置关闭。
3. Verify 命令是否由 TaskBridge 自动运行？推荐：不运行；只推荐原生 `openspec validate` 或项目验证命令。
4. 是否同步 OpenSpec tasks 到 Todoist/Microsoft Todo？推荐：不做。先在 TaskBridge 本地控制面展示，避免产生双事实源。
5. 是否新增 `model.SourceOpenSpec`？推荐：谨慎。优先作为 control-plane projection source，不进入 provider/store 模型。
