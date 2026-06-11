# agent 命令

`taskbridge agent` 是 Agent 安全执行入口。`agent *` stdout 必须保持 `taskbridge.agent-result.v1` JSON；普通命令如需机器输出，当前以各自已实现的 `--format json` 或命令专属 flag 为准。

## 什么时候用

适合用 `agent` 的情况：

- Agent 需要读取当前任务状态，获取结构化数据。
- Agent 需要生成目标计划或执行 action file。
- Agent 需要查询可调用能力和 schema。

不适合用 `agent` 的情况：

- 人类在终端操作任务：用普通命令（`list`、`task`、`sync` 等）。
- 只想在普通命令中获取机器输出：使用该命令 help 中声明的 `--format json` 或命令专属 flag。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge agent capabilities` | 输出 Agent 可调用能力。 | 不写入。 |
| `taskbridge agent today` | 输出 Agent 友好的 today。 | 不写入。 |
| `taskbridge agent plan <goal>` | 生成目标计划预览。 | 默认 dry-run；`--dry-run=false` 写本地 project store。 |
| `taskbridge agent execute --action-file actions.json` | 执行 action file。 | 默认 dry-run；`--confirm` 后写本地任务。 |
| `taskbridge agent schemas` | 输出 Agent schema 名称。 | 不写入。 |

## 常用流程

### 查询能力

```bash
taskbridge agent capabilities
taskbridge agent schemas
```

### 读取状态

```bash
taskbridge agent today
```

### 生成计划

```bash
taskbridge agent plan "学习 OpenClaw" --dry-run
taskbridge agent plan "学习 OpenClaw" --dry-run=false
```

### 执行动作

```bash
taskbridge agent execute --action-file actions.json --dry-run
taskbridge agent execute --action-file actions.json --confirm
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| `taskbridge.agent-result.v1` JSON | stdout 永远是这个结构。 |
| stderr | 脱敏诊断信息。 |

## 边界

- Agent 不直接读写 `~/.taskbridge` 数据文件。
- Agent 不持有 Provider token，不绕过 Provider 接口。
- 危险动作没有 `--confirm` 时必须返回 `requires_confirmation=true`，不能写入。
- `execute` 只执行 action file 中声明的动作，不执行未声明操作。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| `requires_confirmation` | 执行危险动作但未传 `--confirm`。 | 审核后追加 `--confirm`。 |
| action file 格式错误 | JSON 结构不符合 schema。 | 用 `agent schemas` 查看正确格式。 |
| 认证缺失 | 尝试操作需要 Provider 认证。 | 先完成 `auth login`。 |

## 最短可用流程

```bash
taskbridge agent capabilities
taskbridge agent today
taskbridge agent plan "学习 Go" --dry-run
```
