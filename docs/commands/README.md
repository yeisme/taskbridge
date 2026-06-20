# TaskBridge 命令手册

本目录管理 TaskBridge CLI 命令说明。根 README 和路线文档只保留主路径、定位和阶段规划；每个 root command 的职责、子命令、输出模式和写入边界都必须放在本目录的独立页面中。

`taskbridge --help` 和当前 Cobra command tree 是二进制事实来源；本目录负责把命令放进用户能理解的工作流、Agent 契约和安全边界里。

## 怎么读

- 第一次使用：先看 [`doctor`](./doctor.md)、[`quickstart`](./quickstart.md)、[`today`](./today.md)、[`next`](./next.md) 和 [`review`](./review.md)。
- 要接入 Agent 或脚本：先看 [`agent`](./agent.md)，普通命令优先使用已实现的 `--json`、`--agent`、`--format json` 或命令专属机器输出，不要解析 human output。
- 要新增或重写命令文档：先看 [命令文档预设](./preset-template.md)。
- 只想查当前参数：运行 `taskbridge <command> --help`，help 是当前实现的事实来源。

## 命令地图

| 分组 | 命令 | 什么时候用 |
| --- | --- | --- |
| 每日控制面 | [`taskbridge today`](./today.md) | 打开每日任务工作台，未来默认包含 OpenSpec 工程任务信号。 |
| 每日控制面 | [`taskbridge next`](./next.md) | 获取当前最值得推进的下一步。 |
| 每日控制面 | [`taskbridge inbox`](./inbox.md) | 查看无归属、缺日期或待整理任务。 |
| 每日控制面 | [`taskbridge review`](./review.md) | 做任务健康复盘，生成建议动作，不默认写入。 |
| 新手入口 | [`taskbridge demo`](./demo.md) | 无需 Provider 认证，用 demo 数据体验控制面能力。 |
| 新手入口 | [`taskbridge doctor`](./doctor.md) | 检查本地环境、存储和 Provider 认证状态。 |
| 任务浏览 | [`taskbridge list`](./list.md) | 按筛选条件列出任务。 |
| 任务浏览 | [`taskbridge lists`](./lists.md) | 列出任务清单和任务数量。 |
| 本地任务 | [`taskbridge task`](./task.md) | 管理本地任务 CRUD 和完成状态。 |
| Provider | [`taskbridge provider`](./provider.md) | 查看、启用、禁用、测试 Provider。 |
| Provider | [`taskbridge auth`](./auth.md) | 管理 Provider 认证、token 状态和刷新。 |
| 同步 | [`taskbridge sync`](./sync.md) | 拉取、推送、diff、冲突、审计和备份。 |
| 项目 | [`taskbridge project`](./project.md) | 项目草稿、拆分、确认、同步和执行闭环。 |
| 治理 | [`taskbridge governance`](./governance.md) | 逾期健康、长期任务调配、拆分建议和成就分析。 |
| 分析 | [`taskbridge analyze`](./analyze.md) | 四象限、优先级、时间和趋势分析。 |
| Agent | [`taskbridge agent`](./agent.md) | Agent 安全读取、计划预览和 action file 执行入口。 |
| 配置 | [`taskbridge config`](./config.md) | 查看当前配置和兼容旧配置命令。 |
| 服务 | [`taskbridge serve`](./serve.md) | 启动后台服务、token 刷新和定时同步。 |
| 交互 | [`taskbridge tui`](./tui.md) | 启动交互式终端界面。 |
| 维护 | [`taskbridge version`](./version.md) | 查看版本和构建信息。 |
| 维护 | [`taskbridge completion`](./completion.md) | 生成 shell completion；`help` 也归到这里说明。 |

## 常见流程

| 目标 | 推荐入口 |
| --- | --- |
| 首次体验 | `taskbridge doctor` -> `taskbridge quickstart` -> `taskbridge demo today` |
| 日常开始工作 | `taskbridge today` -> `taskbridge next` |
| 处理风险 | `taskbridge review` |
| 查看 OpenSpec 工程信号 | `taskbridge today` 或 `taskbridge next --source openspec`；详细操作继续用原生 `openspec ...`。 |
| 列出任务 | `taskbridge list --format table` 或 `taskbridge list --format json` |
| 同步 Provider | `taskbridge auth status` -> `taskbridge sync pull <provider>` |
| Agent 读取状态 | `taskbridge agent today` 或已实现的 `--format json` 输出 |
| Agent 执行动作 | `taskbridge agent execute --action-file actions.json --dry-run`，确认后才用 `--confirm`。 |

## 输出模式

| 模式 | 用途 |
| --- | --- |
| 默认英文摘要 | 人类操作，快速扫状态、重点、风险和推荐下一步。 |
| `--format json` | Agent、CI、脚本读取稳定 JSON；仅适用于 help 中声明该 flag 的命令。 |
| `taskbridge version --json` | 版本命令的专属 JSON 输出。 |
| `taskbridge agent *` | Agent 安全入口，stdout 是 `taskbridge.agent-result.v1` JSON。 |

机器模式不能混入进度条、人类提示、ANSI 或日志。stderr 只放脱敏诊断。

## 写入规则

| 类型 | 示例 | 写入边界 |
| --- | --- | --- |
| 只读查看 | `today`、`next`、`inbox`、`review`、`list`、`doctor`、`sync diff` | 不写本地任务，不调用远端写 API。 |
| 本地任务写入 | `task add/edit/delete/done/undo`、`project confirm/done/archive` | 写本地 storage/project store；高风险动作需要确认或后续 action/audit。 |
| Provider 写入 | `sync push`、项目同步到 Provider | 必须经过 Provider 接口；危险写入需要 dry-run/confirm/audit。 |
| Agent 写入 | `agent execute --confirm` | 只执行 action file；无确认时不能写。 |
| OpenSpec 信号 | `today`、`next --source openspec`、`review --source openspec` | 首阶段只读，不新增 `taskbridge openspec *` wrapper，不写 OpenSpec 文件。 |
