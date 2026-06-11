# lists 命令

`taskbridge lists` 列出任务清单和任务数量，用于查看 Provider/List 结构。它和 `list` 的区别是：`lists` 展示清单（文件夹/项目）级别的结构和统计，`list` 展示任务级别的筛选和浏览。

## 什么时候用

适合用 `lists` 的情况：

- 想查看某个 Provider 下有哪些清单（如 Microsoft Todo 的任务列表）。
- 想知道每个清单里有多少任务。
- 想了解远端清单结构再做同步。

不适合用 `lists` 的情况：

- 想看具体任务：用 `taskbridge list`。
- 想管理清单：清单管理在对应 Provider App 中完成。

## 子命令

`lists` 当前没有子命令。

## 常用流程

```bash
taskbridge lists
taskbridge lists --source microsoft
taskbridge lists --source todoist --format json
taskbridge lists --format json
```

## 输出模式

| 格式 | 用途 |
| --- | --- |
| `table`（默认） | 人类表格浏览清单结构。 |
| `json` | 机器可解析清单摘要数组。 |

## 边界

- 默认只读。
- `--sync-now` 如存在时只做拉取，不做远端写入。
- 清单的增删改需要在对应 Provider App 中完成，TaskBridge 不管理清单生命周期。

## 常见错误

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| Provider 未认证 | `--source` 指定的 Provider 未登录。 | 先运行 `taskbridge auth login <provider>`。 |
| 无清单数据 | 未同步过。 | 先运行 `sync pull <provider>`。 |

## 最短可用流程

```bash
taskbridge lists --source microsoft --format json
```
