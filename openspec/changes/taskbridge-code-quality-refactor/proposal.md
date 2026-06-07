# Proposal: TaskBridge 代码质量和 CLI 输出契约重构

## 背景

TaskBridge 已经完成第一轮结构拆分：`taskbridge list` 的查询逻辑进入 `internal/taskquery`，任务渲染被集中到 `pkg/output`，provider catalog 和 filestore 查询也开始复用共享实现。当前剩余问题集中在质量边界而不是功能缺失：命令层仍会把人类提示写入 JSON stdout，公开 `pkg/output` 包反向暴露 `internal/model.Task`，命令级输出缺少契约测试，渲染层和 CLI 层的职责仍不够清楚。

二轮 review 继续发现两类更底层问题：多处 `getStore()` 调用在检查错误前 `defer cleanup()`，而 `getStore()` 失败时返回 nil cleanup，会把可解释的存储初始化错误变成 panic；`cmd/auth.go` 中 Microsoft/Feishu 缺凭证和 TickTick/Dida refresh 读取 token 失败会打印错误但返回成功，脚本和 CI 会误判命令执行成功。

全仓 review 继续发现持久化完整性问题：file store 的 `GetTask`/`GetTaskList` 直接返回内部 map 指针，调用方可以绕过 `SaveTask` 修改内存状态；`ListTasks`/`QueryTasks` 返回的是浅拷贝，`Metadata`、slice、map 等嵌套字段仍共享；项目同步和 TUI 中存在多处 `_ = SaveTask`/`_ = DeleteTask`，远程或 UI 操作可能显示成功但本地状态没有可靠落盘。

用户已允许破坏性改造，因此本 change 不保留旧的内部 API 或 `pkg/output` 兼容面；优先把边界收紧、测试补齐、输出契约固定下来。

## 目标

- 让 `taskbridge list` 和后续可复用 CLI 输出路径符合机器可解析输出契约：JSON stdout 必须是纯 JSON，提示、分页说明、诊断信息走 stderr 或结构化字段。
- 移除公开包对 `internal/model` 的依赖，重新定义任务输出 projection 的归属，避免不可用或误导性的 public API。
- 将 list 查询、projection、render、Cobra 命令组装拆成小模块，降低单文件膨胀和回归风险。
- 建立命令级契约测试，覆盖空结果、分页提示、JSON/table/compact/tsv/markdown 等关键输出模式。
- 锁定 provider catalog 和 filestore 查询复用行为，避免后续重构重新引入硬编码列表或双实现漂移。
- 修复存储初始化失败时的 nil cleanup panic，所有 CLI 命令必须返回稳定错误而不是崩溃栈。
- 修复 auth 命令的成功/失败退出码契约，错误场景不能返回 exit code 0。
- 收敛 analyze、lists、sync、task show 等命令的 JSON/text 输出路径，减少手写 marshal 和 stdout 分流。
- 建立 storage ownership 契约，读接口返回深拷贝，写接口保存自有副本，避免外部指针直接污染 store 内存。
- 清理同步、项目同步和 TUI 中被吞掉的持久化错误，成功计数和 UI 状态必须以本地写入结果为准。

## 非目标

- 不新增真实第三方 provider 能力，不改 Feishu、Google、Notion、Todoist、Slack、GitHub、MongoDB 的外部协议。
- 不重写 TUI、不引入新的 UI 框架。
- 不把所有命令一次性迁移到完整 `--agent`、`--events`、`--explain`；本 change 只为 list 输出建立可扩展的渲染边界。
- 不保留旧 `pkg/output.RenderTasks` public API 兼容；如确有复用需求，后续用稳定 DTO 重新开放。

## 风险与取舍

- 破坏性移动 `pkg/output` 会影响仓库内旧引用；本 change 要求一次性迁移并用 `go test ./...` 兜底。
- JSON 输出修正可能改变用户脚本依赖的人类提示；取舍上优先机器可解析契约，人类提示保留在 table/compact 或 stderr。
- 引入命令级测试可能增加 fixture 成本；只覆盖关键 stdout/stderr 契约，不把所有业务分支升级成 process e2e。
