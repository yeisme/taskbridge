# project 命令

`taskbridge project` 管理项目草稿、拆分建议、确认落库、Provider 同步和项目执行闭环。它把目标规划拆成可执行任务序列，不绕过 Provider 接口写远端。

## 什么时候用

适合用 `project` 的情况：

- 有一个学习目标或项目计划，想拆成具体可执行任务。
- 有 Markdown 格式的任务树，想导入为结构化项目。
- 想把本地项目任务同步到 Todo Provider。
- 想查看项目执行状态、下一步或调整计划。

不适合用 `project` 的情况：

- 只想添加单个任务：用 `taskbridge task add`。
- 只想查看所有任务列表：用 `taskbridge list`。
- 想分析任务健康度或逾期：用 `taskbridge governance`。

## 子命令

| 命令 | 用途 | 写入 |
| --- | --- | --- |
| `taskbridge project create <name>` | 创建项目草稿。 | 写 project store。 |
| `taskbridge project list` | 列出项目。 | 不写入。 |
| `taskbridge project split <project-id>` | 生成拆分建议。 | 写 project plan。 |
| `taskbridge project split-markdown <project-id>` | 从 Markdown 任务树生成拆分建议。 | 写 project plan。 |
| `taskbridge project confirm <project-id>` | 确认项目并落库任务。 | 写 task/project store。 |
| `taskbridge project sync <project-id>` | 同步项目任务到 Provider。 | 写远端 Provider。 |
| `taskbridge project review <project-id>` | 复盘项目执行状态。 | 不写入。 |
| `taskbridge project next <project-id>` | 输出项目下一步。 | 不写入。 |
| `taskbridge project adjust <project-id>` | 生成或应用项目调整。 | 默认 dry-run；confirm 后写任务。 |
| `taskbridge project done <project-id>` | 标记项目完成。 | 写 project store。 |
| `taskbridge project archive <project-id>` | 归档项目。 | 写 project store。 |

## 主流程

### 1. 创建项目草稿

```bash
taskbridge project create "学习 OpenClaw"
taskbridge project create "发布 TaskBridge 控制面" --goal-text "希望完成控制面四阶段"
```

### 2. 拆分项目

```bash
taskbridge project split <project-id> --max-tasks 10
taskbridge project split <project-id> --format json
```

### 3. 确认落库

```bash
taskbridge project confirm <project-id>
```

确认后会在本地 task store 创建实际任务。

### 4. 同步到 Provider

```bash
taskbridge project sync <project-id>
```

将项目任务推送到已启用的远端 Provider。

## 从 Markdown 创建

```bash
taskbridge project create "季度规划"
taskbridge project split-markdown <project-id> --file plan.md
taskbridge project confirm <project-id>
```

## 项目执行闭环

```bash
taskbridge project review <project-id>
taskbridge project next <project-id>
taskbridge project adjust <project-id>
taskbridge project done <project-id>
taskbridge project archive <project-id>
```

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认中文摘要 | 人类查看项目状态和拆分建议。 |
| `--format json` | 机器解析项目结构和任务列表。 |

## 边界

- `confirm` 会创建本地任务；远端同步必须显式 `project sync`。
- `adjust` 默认 dry-run，有 action 时必须确认后应用。
- `archive` 不删除历史数据，只标记归档状态。
- 不绕过 Provider 接口直接写远端 Todo 平台。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 项目不存在 | `<project-id>` 无效。 | 用 `project list` 查看当前项目。 |
| 未拆分就确认 | `confirm` 需要先完成 `split`。 | 先运行 `project split <id>`。 |
| Provider 未启用 | `project sync` 需要已启用的 Provider。 | 用 `provider enable <name>` 启用。 |

## 最短可用流程

```bash
taskbridge project create "学习 OpenClaw"
taskbridge project split <project-id> --max-tasks 10
taskbridge project confirm <project-id>
```
