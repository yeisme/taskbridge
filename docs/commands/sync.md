# sync 命令

`taskbridge sync` 同步本地任务和远端 Provider。可信写入、diff、冲突和 audit 都属于这个命令族。它和 `list --sync-now` 的区别是：`sync` 提供完整的双向同步控制、冲突处理和审计记录，`list --sync-now` 只在列表前做一次性拉取。

## 什么时候用

适合用 `sync` 的情况：

- 需要把所有已认证 Provider 的远端任务统一拉取到本地，或把本地任务推送到远端。
- 需要双向同步，保留双方最新变更。
- 需要预览同步差异再决定是否写入。
- 需要处理同步冲突，选择保留哪一方的变更。
- 需要创建本地数据快照或恢复历史快照。
- 需要查看同步审计记录，确认上次同步写了什么。

不适合用 `sync` 的情况：

- 只想在列表前自动拉取最新任务：用 `taskbridge list --sync-now`。
- 只想查看当前 Provider 认证状态：用 `taskbridge auth status`。
- 想自动定时同步：用 `taskbridge serve --sync`。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge sync pull <provider>` | 从单个远端 Provider 拉取到本地。 | 写本地 storage。 |
| `taskbridge sync pull --all` | 聚合拉取所有已认证 Provider。 | 写本地 storage；单个 Provider 失败不阻断其它 Provider。 |
| `taskbridge sync push <provider>` | 从本地推送到远端。 | 写远端 Provider。 |
| `taskbridge sync bidirectional <provider>` | 双向同步。 | 写本地和远端。 |
| `taskbridge sync watch <provider>` | 持续定时同步。 | 写本地和/或远端。 |
| `taskbridge sync status [provider]` | 查看同步状态。 | 不写入。 |
| `taskbridge sync diff <source> --target <provider>` | 预览同步差异。 | 写 audit session；不写任务。 |
| `taskbridge sync conflicts` | 列出同步冲突。 | 不写任务。 |
| `taskbridge sync resolve <conflict-id>` | 解决冲突。 | 写冲突状态；高风险策略需确认。 |
| `taskbridge sync backup create` | 创建本地数据快照。 | 写 backup。 |
| `taskbridge sync backup restore <backup-id>` | 恢复快照。 | 写本地 storage。 |
| `taskbridge sync audit <session-id>` | 查看同步审计记录。 | 不写入。 |

## 常用流程

### 单向拉取

```bash
taskbridge sync status
taskbridge sync pull microsoft
taskbridge sync pull --all
taskbridge sync pull --all --dry-run --json
taskbridge sync pull todoist --format json
```

`sync pull --all --json` 的 `data` 会包含 `providers_attempted`、`providers_succeeded`、`providers_failed`、`providers_skipped` 和 per-provider `pulled/created/updated/skipped/conflicts/errors/next_action`。无认证 Provider 的环境会返回 skipped 汇总和下一步认证建议，便于新环境 smoke test。

### 双向同步

```bash
taskbridge sync diff microsoft --target todoist --format json
taskbridge sync bidirectional microsoft
```

### 冲突处理

```bash
taskbridge sync conflicts
taskbridge sync resolve <conflict-id>
```

### 备份与恢复

```bash
taskbridge sync backup create
taskbridge sync backup restore <backup-id>
```

### 审计

```bash
taskbridge sync audit <session-id> --format json
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文摘要 | 人类查看同步结果。 |
| `--format json` / `--json` | 机器解析同步结构。 |
| `--agent` | 输出低 token key=value facts。 |

## 边界

- `--dry-run` 不写本地 storage，不调用远端写 API。
- `sync pull --all` 只调用 Provider 读取接口；远端写 API 仍只能通过 `sync push`、双向 push 阶段或 action file confirm gate 触发。
- 远端覆盖、删除、冲突丢弃必须显式确认或进入 action/audit gate。
- sync/audit 输出必须说明比较了什么、准备写什么、实际写了什么。
- 不让 sync 命令绕过 Provider 接口直接写远端 Todo 平台。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| Provider 未认证 | `sync pull/push` 需要有效 token；`sync pull --all` 会把该 Provider 标记为 skipped。 | 先运行 `taskbridge auth login <provider>`。 |
| 冲突未解决 | 双向同步发现同一任务双方都有修改。 | 用 `sync conflicts` 查看，`sync resolve` 选择保留策略。 |
| Token 过期 | OAuth token 已失效。 | 运行 `taskbridge auth refresh <provider>`。 |
| 同步中断 | 网络或 Provider 限流。 | 检查 `sync status`，确认部分写入状态后重试。 |

## 最短可用流程

```bash
taskbridge auth login microsoft
taskbridge sync pull --all
taskbridge sync status
```
