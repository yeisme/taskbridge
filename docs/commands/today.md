# today 命令

`taskbridge today` 是跨 Todo 软件的 daily hub。它把所有已同步 Provider 的任务按 Work、Life、Inbox、Overdue、Recommended next 和 Sync warnings 汇总，让用户不必分别打开多个 Todo app 才知道今天该做什么。

## 什么时候用

适合用 `today` 的情况：

- 每日开始工作时想快速了解工作和生活任务的整体状态。
- 想在一个入口看到跨 Provider 任务、风险和推荐下一步。
- 想用 mock 数据体验 TaskBridge 功能。

不适合用 `today` 的情况：

- 想看完整的任务列表：用 `list`。
- 想只看推荐下一步：用 `next`。
- 想做任务健康复盘：用 `review`。

## 子命令

`today` 当前没有子命令。

## 常用流程

```bash
taskbridge today
taskbridge today --json
taskbridge sync pull --all && taskbridge today
taskbridge today --source microsoft
```

`--mock` 使用内置模拟数据（保留给测试和兼容路径）；新用户应使用 `taskbridge demo today` 体验控制面。真实日常路径建议先运行 `taskbridge sync pull --all`，再运行 `taskbridge today`。

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文摘要 | 突出状态、重点、风险和推荐下一步。 |
| `--format json` / `--json` | 输出 AI-native envelope，`data` 内含 `taskbridge.today.v1`。 |
| `--agent` | 输出低 token key=value facts，供 Agent 和 shell glue 使用。 |

## 边界

- 只读命令，不写本地 storage，不调用远端写 API。
- `--mock` 使用内置模拟数据，不读取真实 token 或任务。
- `domain` 是任务上下文，合法值为 `work`、`life`、`personal`、`unknown`；不会替代 `source`、`provider`、`list_id` 或远端 ID。
- 旧任务缺少 domain 时仍会显示，并在 JSON/agent 输出里渲染为 `unknown`。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 无任务 | 本地 storage 为空。 | 先运行 `taskbridge sync pull --all` 拉取任务。 |
| Provider 未认证 | 指定了 `--source` 但未登录。 | 先运行 `auth login <provider>`。 |

## 最短可用流程

```bash
taskbridge today
```
