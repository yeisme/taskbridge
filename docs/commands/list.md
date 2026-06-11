# list 命令

`taskbridge list` 按来源、状态、象限、优先级、标签、清单、关键词等条件列出任务。它是任务浏览的主入口，支持多种输出格式。

## 什么时候用

适合用 `list` 的情况：

- 想按条件筛选和浏览任务。
- 想用 JSON 输出接入脚本或 Agent。
- 想在看列表前自动拉取最新数据。

不适合用 `list` 的情况：

- 想查看任务清单结构：用 `lists`。
- 想操作单个任务：用 `task`。
- 想看每日工作台：用 `today`。

## 子命令

`list` 当前没有子命令。

## 常用流程

### 基本浏览

```bash
taskbridge list
taskbridge list --all
taskbridge list --format table
```

### 按条件筛选

```bash
taskbridge list --source microsoft --status todo
taskbridge list --query "今天"
taskbridge list --priority 3
taskbridge list --source todoist --status completed
```

### 机器输出

```bash
taskbridge list --format json
taskbridge list --fields id,title,status,due_date --format json
taskbridge list --format compact
taskbridge list --format markdown
taskbridge list --format tsv
```

### 拉取后列表

```bash
taskbridge list --sync-now --source microsoft
```

## 输出模式

| 格式 | 用途 |
| --- | --- |
| `table`（默认） | 人类表格浏览。 |
| `json` | 机器可解析，不能混入中文提示。 |
| `markdown` | 文档或笔记集成。 |
| `compact` | 终端紧凑浏览。 |
| `tsv` | Shell glue 或 Excel 导入。 |

## 边界

- 默认只读。
- `--sync-now` 会先拉取远端任务到本地 storage；不做远端写入。
- 分页、total、has_more 等机器事实必须进入 JSON 字段，不能只出现在中文提示里。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| 无任务 | 本地 storage 为空或筛选条件过严。 | 放宽筛选条件或先 `sync pull`。 |
| Provider 无数据 | 未同步或 Provider 未启用。 | 先运行 `sync pull <provider>`。 |

## 最短可用流程

```bash
taskbridge list
taskbridge list --format json
```
