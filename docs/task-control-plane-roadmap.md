# TaskBridge 任务控制面四阶段路线设计

更新时间：2026-04-30

## 1. 核心判断

TaskBridge 的下一步不是继续横向堆 Todo Provider，也不是先追求完美双向同步，而是先成为用户每天会打开的本地任务控制面。

原路线把“可信同步层”放在第一阶段，工程上正确，但产品启动偏慢。用户不会因为 conflict resolver 完整就每天运行工具，用户会因为 `taskbridge today` 真能告诉他今天该做什么而留下来。

因此路线调整为：

```text
体验地基
  -> 只读任务控制面
  -> 可信写入与同步审计
  -> 项目执行闭环
  -> Agent 安全执行层
```

一句话定位：

> TaskBridge 是人和 AI 共用的本地任务驾驶舱：先帮用户看清今天，再安全地把决策写回各 Todo 平台。

## 2. 北极星指标

主指标：

> 用户每个工作日运行 `taskbridge today`，并从输出中完成至少一个任务决策。

任务决策包括：

- 开始一个推荐任务。
- 推迟一个逾期任务。
- 拆分一个过大的任务。
- 清理一个 inbox 任务。
- 解决一个同步冲突。
- 推进一个项目的下一步。

辅助指标：

- `today` 运行成功率。
- `today -> action` 转化率。
- 每周被 `next` 推荐后完成的任务数。
- 每周被用户确认的写入动作数。
- 同步冲突被解决的平均时间。
- Agent dry-run 到 confirmed execute 的转化率。

## 3. 目标用户

- 同时使用多个 Todo 平台的人：Microsoft To Do、Todoist、飞书、TickTick、Google Tasks 等。
- 主要在终端、脚本和 agent 工作流中管理任务的人。
- 希望 AI 帮忙拆任务、做每日计划、处理逾期，但不希望 AI 静默改掉全部任务的人。
- 希望任务数据保留在本地，有明确同步记录、冲突记录和回滚路径的人。

## 4. 产品原则

1. `today` 是第一公民。同步、治理、项目、Agent 都服务于每日决策。
2. 先读后写。先让用户安全地看清任务，再逐步开放写入和双向同步。
3. CLI 是核心入口。未来可以有 MCP adapter，但 adapter 只能薄封装 CLI。
4. 自动化必须可解释。任何写入都要能回答：为什么、改了什么、能否撤回。
5. AI 只做建议和编排，不默认静默执行危险动作。
6. JSON stdout 必须稳定。日志、人类提示、调试信息只能写 stderr。

## 5. 非目标

- 不做完整 SaaS Web App。
- 不把核心重构成常驻 MCP Server 或 HTTP Server。
- 不在当前四阶段实现复杂日程自动排程。
- 不承诺所有 Provider 能力完全一致。
- 不让 Agent 绕过 CLI 直接操作本地存储。
- 不把 AI 生成结果直接当成事实执行。

## 6. 当前基础

已有能力：

- Provider：Microsoft To Do、Google Tasks、飞书任务、TickTick、滴答清单、Todoist。
- CLI：`auth`、`provider`、`list`、`lists`、`task`、`sync`、`project`、`analyze`、`governance`、`tui`、`serve`。
- 项目拆分：`project create -> split -> confirm -> sync`。
- 治理：逾期健康分析、长期任务调配、复杂任务识别、任务拆分建议、成就分析。
- 文档：已有多 Provider 存储、目标拆分 MVP、CLI 命令设计。

主要缺口：

1. 新用户很难快速看到价值，必须先理解 provider、auth、storage、sync。
2. `today` 还不是主入口，日常任务决策分散在 `list/analyze/governance/project`。
3. 写入动作的预览、审计、回滚和冲突处理不够显性。
4. 项目拆分还偏一次性生成，缺少执行中 review、next、adjust、done。
5. Agent 可调用性还没有形成统一契约。

## Phase 0：体验地基

### 目标

让用户在不完成复杂 OAuth 和同步配置前，也能看到 TaskBridge 的核心价值。

Phase 0 不是正式产品阶段，而是所有后续阶段的体验前置条件。它解决的是“用户第一次运行时能不能在 2 分钟内理解 TaskBridge 值得继续配置”。

### 用户工作流

```bash
taskbridge doctor
taskbridge demo today
taskbridge quickstart
taskbridge provider list
taskbridge auth status
```

### 命令设计

| 命令 | 作用 |
| --- | --- |
| `doctor` | 检查配置、存储路径、Provider 启用状态、凭证状态、写权限 |
| `demo today` | 用内置示例数据展示 `today` 效果，不需要登录 Provider |
| `quickstart` | 给出下一步命令，不做交互式重配置 |
| `auth status` | 显示各 Provider 的认证状态和下一步动作 |

### `doctor` 输出契约

```json
{
  "schema": "taskbridge.doctor.v1",
  "status": "warning",
  "checks": [
    {
      "id": "storage_path",
      "status": "ok",
      "message": "storage path is writable"
    },
    {
      "id": "provider_auth",
      "status": "warning",
      "message": "no provider authenticated",
      "next_action": "taskbridge demo today"
    }
  ]
}
```

### 验收标准

- 用户无需认证任何 Provider，也能运行 `taskbridge demo today`。
- `doctor` 能明确指出缺什么，并给出下一条命令。
- `quickstart` 根据当前状态输出不同路径：demo、auth、sync、today。
- 所有命令支持 `--format json`。

### 降级策略

如果时间不足，保留：

- `doctor`
- `demo today`

暂缓：

- `quickstart`
- 更复杂的环境修复建议。

### 阶段门禁

Phase 0 完成后，新用户从 clone/build 到看到 `today` 示例输出不超过 2 分钟。

## Phase 1：只读任务控制面

### 目标

在不写入远端 Provider 的前提下，把 TaskBridge 变成每天能用的任务控制面。

这阶段只做安全读、聚合、推荐和解释。写入能力只允许本地 dry-run 或明确的本地任务管理，不做跨 Provider 自动 push。

### 用户工作流

```bash
taskbridge sync pull microsoft
taskbridge sync pull todoist
taskbridge today
taskbridge next
taskbridge inbox
taskbridge review
taskbridge today --format json
```

### 命令设计

| 命令 | 作用 |
| --- | --- |
| `today` | 每日任务工作台，聚合今日、逾期、风险、项目下一步、同步健康 |
| `next` | 给出当前最值得推进的 1-5 个任务 |
| `inbox` | 列出无项目、无日期、无下一步归属的待整理任务 |
| `review` | 任务健康复盘，只输出建议，不默认写入 |
| `sync pull` | 从 Provider 拉取到本地，仍按现有能力执行 |

### `today` 输出结构

```json
{
  "schema": "taskbridge.today.v1",
  "date": "2026-04-30",
  "status": "ok",
  "summary": {
    "must_do": 4,
    "overdue": 7,
    "at_risk": 3,
    "inbox": 9,
    "project_next": 2,
    "sync_warnings": 1
  },
  "sections": [
    {
      "id": "must_do",
      "title": "今日必须做",
      "tasks": ["task_1", "task_2"]
    },
    {
      "id": "at_risk",
      "title": "即将失控",
      "tasks": ["task_3"]
    },
    {
      "id": "next",
      "title": "建议下一步",
      "tasks": ["task_4", "task_5"]
    }
  ],
  "suggested_actions": [
    {
      "action_id": "act_001",
      "type": "defer",
      "task_id": "task_3",
      "reason": "任务已逾期 5 天且无完成记录",
      "requires_confirmation": true
    }
  ],
  "warnings": []
}
```

### 排序原则

1. 明确今天到期或已逾期的任务优先。
2. 属于活跃项目的任务优先于孤立任务。
3. 可在 30-180 分钟内完成的任务优先。
4. 任务过大时不推荐直接执行，推荐拆分。
5. 有同步冲突或 Provider 状态不确定的任务只进入提醒区。

### 验收标准

- `today` 能在只有本地存储、无 Provider 认证时运行。
- `today --format json` 的字段稳定，可被脚本和 Agent 消费。
- `next` 默认只输出少量任务，不退化成完整列表。
- `review` 只输出建议动作，不写入。
- text 输出适合人读，json 输出适合机器读。
- stdout/stderr 分离：JSON 只出现在 stdout。

### 降级策略

如果时间不足，保留：

- `today`
- `next`
- JSON 输出契约。

暂缓：

- `inbox`
- `review --apply`
- 精细同步健康评分。

### 阶段门禁

Phase 1 完成后，用户能每天先运行 `taskbridge today`，并从输出中决定下一步做什么。

## Phase 2：可信写入与同步审计

### 目标

在用户已经依赖 `today` 后，逐步开放可信写入：先预览，再确认，再审计，再可恢复。

这阶段的核心不是“做完美同步系统”，而是让用户敢把 `today/review/project` 产生的决策写回本地和远端 Provider。

### 用户工作流

```bash
taskbridge sync diff microsoft --target todoist
taskbridge sync push todoist --dry-run
taskbridge review --apply-file actions.json --dry-run
taskbridge review --apply-file actions.json --confirm
taskbridge sync conflicts
taskbridge sync resolve <conflict-id> --strategy source-wins
taskbridge sync backup create
taskbridge sync audit <session-id>
```

### 命令设计

| 命令 | 作用 |
| --- | --- |
| `sync diff <source> [--target <provider>]` | 基于本地 source/target 快照预览新增、更新、删除、跳过和冲突 |
| `sync push <provider> --dry-run` | 预览写回远端 Provider |
| `sync conflicts` | 列出待处理冲突 |
| `sync resolve <id>` | 用指定策略处理冲突 |
| `sync backup create` | 创建本地数据快照 |
| `sync backup restore <id>` | 从快照恢复本地数据 |
| `sync audit <session-id>` | 查看一次写入或同步的完整记录 |

### Action file 契约

Phase 2 开始引入结构化 action file，Phase 4 再统一到 `agent execute`。

```json
{
  "schema": "taskbridge.actions.v1",
  "source": "review",
  "created_at": "2026-04-30T10:30:00+08:00",
  "actions": [
    {
      "id": "act_001",
      "type": "defer_task",
      "task_id": "task_123",
      "due_date": "2026-05-02",
      "reason": "任务逾期且仍有价值，建议推迟到最近可执行日期",
      "requires_confirmation": true
    }
  ]
}
```

### 同步审计结构

```json
{
  "schema": "taskbridge.sync-session.v1",
  "id": "sync_20260430_101500",
  "mode": "push",
  "source": "local",
  "target": "todoist",
  "dry_run": false,
  "started_at": "2026-04-30T10:15:00+08:00",
  "completed_at": "2026-04-30T10:15:08+08:00",
  "status": "completed",
  "stats": {
    "created": 3,
    "updated": 8,
    "deleted": 0,
    "skipped": 41,
    "conflicts": 1,
    "errors": 0
  },
  "operations": [
    {
      "op_id": "op_001",
      "type": "update",
      "local_id": "task_abc",
      "provider_id": "123456",
      "title": "完成项目报告",
      "before_hash": "sha256:old",
      "after_hash": "sha256:new",
      "requires_confirmation": false
    }
  ]
}
```

`sync diff` 不实时调用远端 Provider；它比较最近一次同步后保存在本地的 source/target 快照。需要最新远端状态时，先运行对应 Provider 的 `sync pull`，再运行 `sync diff`。

### 冲突结构

```json
{
  "schema": "taskbridge.conflict.v1",
  "id": "conflict_001",
  "local_id": "task_abc",
  "providers": ["microsoft", "todoist"],
  "field_conflicts": ["title", "due_date", "status"],
  "detected_at": "2026-04-30T10:15:08+08:00",
  "strategies": ["latest-wins", "source-wins", "target-wins", "manual"],
  "status": "open"
}
```

### 验收标准

- `sync diff` 不写入任何本地或远端数据。
- `--dry-run` 不改变本地存储和远端 Provider。
- 每次真实写入都生成 audit session。
- 批量删除、批量完成、覆盖远端数据必须 `requires_confirmation=true`。
- 冲突不会静默解决，默认进入 `sync conflicts`。
- 用户可以恢复到最近一次写入前的本地快照。

### 降级策略

如果时间不足，保留：

- `sync diff`
- `--dry-run`
- audit session。

暂缓：

- 双向同步。
- 自动冲突解决。
- 全量 backup restore UI。

### 阶段门禁

Phase 2 完成后，用户可以放心把 `today/review` 里的明确决策写回一个 Provider，并能审计和回滚。

## Phase 3：项目执行闭环

### 目标

把项目从“一次性拆分”升级为“执行中持续调整”，并把项目下一步接入 `today`。

项目模块不应是独立岛。它应该成为 `today` 和 `next` 的高质量信号来源：哪个项目在推进，哪个项目失控，哪个项目只需要下一步。

### 用户工作流

```bash
taskbridge project create "学习 OpenClaw" --goal-text "我希望学习 openclaw"
taskbridge project split <project-id> --max-tasks 10
taskbridge project confirm <project-id>
taskbridge project review <project-id>
taskbridge project next <project-id>
taskbridge project adjust <project-id> --reason "本周只完成了 2 个任务" --dry-run
taskbridge project adjust <project-id> --reason "本周只完成了 2 个任务" --confirm
taskbridge project done <project-id>
taskbridge project archive <project-id>
taskbridge today
```

### 命令设计

| 命令 | 作用 |
| --- | --- |
| `project review <id>` | 输出项目进度、风险、卡点和建议动作 |
| `project next <id>` | 输出该项目当前最该做的一步 |
| `project adjust <id>` | 基于执行状态重排剩余任务 |
| `project done <id>` | 标记项目完成并生成收口摘要 |
| `project archive <id>` | 归档项目，不再出现在默认工作台 |

### 项目状态机

```text
draft
  -> split_suggested
  -> confirmed
  -> active
  -> completed
  -> archived

confirmed -> active: 首个项目任务开始、完成或同步成功
active -> completed: 用户显式执行 project done
active -> archived: 用户显式归档
completed -> archived: 用户显式归档
```

### 项目 review 输出结构

```json
{
  "schema": "taskbridge.project-review.v1",
  "project_id": "proj_001",
  "status": "active",
  "progress": {
    "total_tasks": 12,
    "completed_tasks": 4,
    "overdue_tasks": 2,
    "blocked_tasks": 1
  },
  "risk": {
    "level": "medium",
    "reasons": ["2 个任务逾期", "剩余任务平均粒度超过 180 分钟"]
  },
  "next_task_id": "task_007",
  "suggested_actions": [
    {
      "type": "split_task",
      "task_id": "task_009",
      "reason": "估算时长超过 180 分钟",
      "requires_confirmation": true
    }
  ]
}
```

### 与 `today` 的集成

`today` 需要新增项目区块：

```json
{
  "id": "project_next",
  "title": "项目下一步",
  "items": [
    {
      "project_id": "proj_001",
      "project_name": "学习 OpenClaw",
      "next_task_id": "task_007",
      "risk_level": "medium"
    }
  ]
}
```

### 验收标准

- `project review` 不修改项目和任务。
- `project next` 默认只返回一个下一步，可用 `--limit` 返回多个。
- `project adjust --dry-run` 输出变更预览，不落库。
- `project adjust` 不能删除用户手写任务，只能新增建议、修改 metadata 或标记需要确认。
- `project done` 生成完成摘要，保留 `completed_at`。
- `project archive` 不删除历史数据。
- `today` 能展示活跃项目下一步和项目风险。

### 降级策略

如果时间不足，保留：

- `project review`
- `project next`
- `today` 项目区块。

暂缓：

- `project adjust` 自动重排。
- `project done` 复杂摘要。
- 项目级历史趋势。

### 阶段门禁

Phase 3 完成后，项目不是拆完就丢，而是能持续出现在 `today` 中，并推动用户完成下一步。

## Phase 4：Agent 安全执行层

### 目标

让 Codex、Claude、脚本和未来 MCP adapter 可以安全调用 TaskBridge，同时保持 CLI 核心简单。

Phase 4 不是“给 Agent 开后门”。相反，它把 Agent 的能力限制在稳定 schema、dry-run、action file 和确认门禁内。

### 用户工作流

```bash
taskbridge agent capabilities
taskbridge agent today
taskbridge agent plan "学习 OpenClaw" --horizon-days 14 --dry-run
taskbridge agent plan "学习 OpenClaw" --horizon-days 14 --dry-run=false
taskbridge agent execute --action-file actions.json --dry-run
taskbridge agent execute --action-file actions.json --confirm
taskbridge agent schemas
```

### 命令设计

| 命令 | 作用 |
| --- | --- |
| `agent capabilities` | 输出 provider、命令、schema version、危险动作规则 |
| `agent today` | Agent 友好的每日工作台输出 |
| `agent plan <goal>` | dry-run 时生成拆分预览；`--dry-run=false` 时写入本地 project draft 和 plan suggestion |
| `agent execute` | 执行结构化 action file |
| `agent schemas` | 输出 JSON schema，便于外部工具校验 |

### Agent result 契约

```json
{
  "schema": "taskbridge.agent-result.v1",
  "status": "ok",
  "request_id": "req_001",
  "dry_run": true,
  "requires_confirmation": false,
  "result": {},
  "warnings": [],
  "errors": []
}
```

错误输出：

```json
{
  "schema": "taskbridge.agent-result.v1",
  "status": "error",
  "request_id": "req_001",
  "dry_run": true,
  "requires_confirmation": false,
  "result": null,
  "warnings": [],
  "errors": [
    {
      "code": "provider_not_authenticated",
      "message": "microsoft Provider 未认证",
      "next_action": "taskbridge auth login microsoft"
    }
  ]
}
```

### 危险动作边界

默认需要确认的动作：

- 删除任务。
- 批量完成任务。
- 批量改 due date。
- 覆盖远端 provider 数据。
- 解决冲突时丢弃一侧数据。
- 项目归档。
- 执行跨 Provider 写入。

默认允许 dry-run 的动作：

- 生成计划。
- 生成拆分建议。
- 生成 today/review/next 输出。
- 生成同步 diff。
- 生成 action file。

### MCP adapter 边界

如果后续做 MCP adapter：

- adapter 只能调用 `taskbridge agent *` 和只读查询命令。
- adapter 不直接读写 `~/.taskbridge` 存储文件。
- adapter 不持有 provider token。
- adapter 必须把 `requires_confirmation=true` 原样返回给上层 Agent。

### 验收标准

- `agent capabilities` 能描述当前环境中可安全调用的能力。
- `agent schemas` 输出可被 JSON schema validator 校验的 schema。
- `agent plan --dry-run=false` 只写入本地 project draft 和 plan suggestion，不创建任务，不写远端 Provider。
- `agent execute --dry-run` 不修改任何任务。
- 所有危险动作在没有 `--confirm` 时返回 `requires_confirmation=true`。
- Agent 命令的 stdout 永远是 JSON。
- MCP adapter 只能薄封装 CLI，不成为第二套核心逻辑。

### 降级策略

如果时间不足，保留：

- `agent capabilities`
- `agent today`
- `agent execute --dry-run`
- schema 输出。

暂缓：

- MCP adapter。
- Agent plan 自动创建真实任务或写远端 Provider。
- 多 action 事务回滚。

### 阶段门禁

Phase 4 完成后，Agent 能安全读取 TaskBridge 状态、生成 action file、dry-run 执行，并在用户确认后写入。

## 7. 跨阶段契约

### 7.1 所有新命令共享字段

从 Phase 1 起，新 JSON 输出都应该尽量包含：

- `schema`
- `status`
- `request_id`
- `dry_run`
- `requires_confirmation`
- `result`
- `warnings`
- `errors`
- `next_action`

### 7.2 stdout/stderr 规则

- JSON 输出只写 stdout。
- 日志、提示、debug、进度条只写 stderr。
- `--format json` 下禁止混入人类提示文本。

### 7.3 写入安全规则

- 任何写入都应支持 `--dry-run`。
- 任何危险动作都应支持 `--confirm` 或 action file 确认。
- 批量写入必须生成 audit session。
- 跨 Provider 写入必须能追踪来源、目标和操作列表。

### 7.4 数据演进顺序

1. Phase 1：只读输出 schema。
2. Phase 2：action file、sync session、conflict、backup。
3. Phase 3：project review、project adjustment、completion summary。
4. Phase 4：agent result、agent schemas、capabilities。

## 8. 工程拆分建议

### Phase 0 文件范围

- `cmd/doctor.go`：环境检查。
- `cmd/demo.go`：示例数据入口。
- `cmd/auth.go`：补强认证状态输出。
- `docs/provider-setup-guide.md`：补 quickstart 路径。

### Phase 1 文件范围

- `cmd/today.go`：每日工作台命令。
- `cmd/next.go`：下一步推荐命令。
- `cmd/inbox.go`：待整理任务命令。
- `cmd/review.go`：任务健康复盘命令。
- `internal/governance/`：聚合 today/next/inbox/review。
- `pkg/output/`：稳定 text/json 输出。

### Phase 2 文件范围

- `internal/sync/`：diff、conflict、session audit。
- `internal/storage/filestore/`：sync session、mapping、backup 存储。
- `cmd/sync.go`：新增 diff/conflicts/resolve/backup/audit。
- `internal/actionfile/`：action file 解析和校验。
- `docs/multi-provider-storage.md`：对齐实现后的同步数据结构。

### Phase 3 文件范围

- `internal/project/`：项目状态、完成摘要、归档字段。
- `internal/projectservice/`：review、next、adjust、done、archive。
- `cmd/project.go`：新增项目执行闭环命令。
- `cmd/today.go`：集成项目下一步区块。
- `docs/goal-decomposition-mvp.md`：升级为项目执行闭环说明。

### Phase 4 文件范围

- `cmd/agent.go`：Agent 命令组。
- `internal/agentcontract/`：schema、capabilities、危险动作门禁。
- `internal/actionfile/`：复用 Phase 2 action file。
- `pkg/output/`：强制 JSON stdout 契约。
- `docs/agent-contract.md`：Agent 和 MCP adapter 契约。

## 9. 里程碑

| 阶段 | 交付物 | 成功信号 |
| --- | --- | --- |
| Phase 0 | 体验地基 | 新用户 2 分钟内看到 `demo today` |
| Phase 1 | 只读任务控制面 | 用户每天先运行 `taskbridge today` |
| Phase 2 | 可信写入与同步审计 | 用户敢把明确决策写回一个 Provider |
| Phase 3 | 项目执行闭环 | 项目下一步进入 `today` 并推动执行 |
| Phase 4 | Agent 安全执行层 | Agent 能 dry-run、生成 action file、等待确认后执行 |

## 10. 推荐优先级

1. 先做 Phase 0 的 `doctor` 和 `demo today`。没有快速体验，后续能力很难被理解。
2. 再做 Phase 1 的 `today` 和 `next`。这是日常留存入口。
3. 再做 Phase 2 的 `sync diff`、`--dry-run` 和 audit。可信写入比双向同步更重要。
4. 再做 Phase 3 的 `project review/next`，并接入 `today`。
5. 最后做 Phase 4 的 `agent` 命令族。等字段、错误码和安全边界稳定后再统一封装。

## 11. 阶段砍线

| 阶段 | 必保 | 可砍 |
| --- | --- | --- |
| Phase 0 | `doctor`、`demo today` | `quickstart` 高级分支 |
| Phase 1 | `today`、`next`、JSON 契约 | `inbox`、`review --apply` |
| Phase 2 | `sync diff`、`--dry-run`、audit | 双向同步、自动冲突解决 |
| Phase 3 | `project review`、`project next`、today 集成 | 自动 adjust、复杂完成摘要 |
| Phase 4 | `agent capabilities`、`agent today`、`execute --dry-run` | MCP adapter、多 action 事务 |

## 12. 开放问题

1. `today` 是否顶级命令？推荐顶级，因为它是主入口。
2. `review` 是否顶级命令？推荐顶级，和 `today/next/inbox` 组成日常命令组。
3. Phase 1 是否允许写本地任务？推荐允许现有 `task` 命令继续工作，但新工作台不主动写入。
4. Phase 2 是否做双向同步？推荐先做单向可信写入，再做双向。
5. Phase 4 是否立即做 MCP adapter？推荐暂缓。CLI agent contract 先稳定，adapter 后做。
6. 是否引入 do date 和容量排程？推荐作为 Phase 5，不进入当前路线。
