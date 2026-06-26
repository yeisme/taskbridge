# Design: TaskBridge Cross-Provider Daily Hub

## Context

TaskBridge 当前已经有多 Provider、统一 task model、daily control plane、sync diff/audit 和 Agent action file。下一阶段产品问题不是“能不能连接更多平台”，而是“连接之后，用户是否一眼看到工作和生活的全部任务，并知道下一步做什么”。

这个设计把核心路径收敛为：

```bash
taskbridge sync pull --all
taskbridge today
taskbridge next
taskbridge review
taskbridge sync push --confirm
```

## Product Model

### Source vs Domain

- `source` 表示任务来自哪个系统，例如 `microsoft`、`todoist`、`feishu`、`ticktick`、`dida`、`google`、`local`。
- `domain` 表示任务属于哪个生活上下文，例如 `work`、`life`、`personal`、`unknown`。
- 一个 Provider 里可以同时有 work/life 任务；不能假设某个 Provider 等于某个 domain。

### Domain Inference

初版采用保守规则：

1. 用户显式设置的 domain 优先。
2. list/project/tag 规则次之，例如清单名、项目名或 tag 命中配置。
3. Provider/source 只能作为弱信号，不能单独决定危险写入或隐藏任务。
4. 无法判断时使用 `unknown`，并在 `inbox` 或 review 中建议整理。

### Daily Hub Sections

`today` 的默认 human output 和 machine data 都围绕决策分区：

- `work`: 工作任务，含 due today、overdue、project next。
- `life`: 生活/家庭/健康/个人事项。
- `inbox`: 缺少 domain、project、date 或 next step 的任务。
- `overdue`: 跨 domain 的逾期风险摘要。
- `recommended_next`: 默认 3-5 个跨 Provider 推荐动作。
- `sync_warnings`: Provider 拉取失败、冲突、过期凭证或未同步提示。

## Command Design

### `sync pull --all`

`sync pull --all` 遍历已启用且已认证 Provider。单个 Provider 失败不应吞掉其它 Provider 结果；整体状态可以是 `partial`。

机器输出需要包含：

- `providers_attempted`
- `providers_succeeded`
- `providers_failed`
- per-provider counts: pulled/created/updated/skipped/conflicts/errors
- `next_action`

### `today`

`today` 不再只是任务列表，而是跨 Provider 汇总页面。默认 human output 要短，重点展示计数、分区和下一条命令。JSON/agent 输出保留完整结构。

### `next`

`next` 默认返回 3-5 项，不退化成全量列表。推荐排序考虑：

1. due today / overdue
2. priority / quadrant
3. active project next step
4. domain balance，避免生活任务长期被工作任务完全淹没
5. sync risk，有冲突或 Provider 状态不确定的任务只能建议 review/resolve，不建议直接 complete/defer

### `review`

`review` 输出一周视角：工作/生活覆盖率、逾期积压、unknown domain 比例、Provider 同步健康和建议动作。普通 `review` 仍只读；写入通过 action file dry-run/confirm。

## Data Shape

新增字段应优先落在统一 task projection 和 control-plane result 中：

```json
{
  "id": "task_123",
  "source": "todoist",
  "domain": "work",
  "title": "Prepare launch checklist",
  "due_date": "2026-06-23",
  "project_id": "proj_launch",
  "sync_state": "ok"
}
```

`domain` 的合法值：

- `work`
- `life`
- `personal`
- `unknown`

## Safety

- `sync pull --all` 是远端读、本地写入口；必须报告每个 Provider 发生了什么。
- 远端写回仍走 `sync push --confirm` 或 action file confirm gate。
- Agent/MCP 只能消费 machine output 和 action file，不直接写 store 或 Provider。
- JSON stdout 不混入 progress、human hints 或日志。

## Rollout

1. 先做 domain model 和 projection，不改变 Provider 写入。
2. 做 `sync pull --all` 和 per-provider summary。
3. 做 `today` work/life/inbox/recommended sections。
4. 做 `next` ranking 和 `review` coverage analysis。
5. 更新 README/commands docs 和 onboarding smoke。

## CEO Review

这个 change 的价值不是“多一个分类字段”，而是把 TaskBridge 的主叙事改成：用户不用打开多个 Todo app，就能在 CLI 里看到工作和生活所有任务，并获得安全下一步。

优先级上，这比 MCP adapter 和新 Provider 更靠前。MCP 是放大器，新 Provider 是覆盖面；daily hub 是留存理由。
