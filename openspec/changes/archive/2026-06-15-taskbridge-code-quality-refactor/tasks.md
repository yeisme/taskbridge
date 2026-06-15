# Tasks: TaskBridge 代码质量和 CLI 输出契约重构

- [x] 0.1 Owner: `cli/taskbridge`。建立 OpenSpec change 骨架，记录 review 发现的 JSON stdout 污染、public API 边界和命令级测试缺口。
- [x] 0.2 Owner: `cli/taskbridge`。完成二轮 review，确认新增缺陷：`getStore()` 失败路径 nil cleanup panic、auth 缺凭证返回成功、analyze/lists/sync/task show 输出分流重复。Lane: review-intake。
- [x] 1.1 Owner: `cli/taskbridge`。补 `taskbridge list --format json` RED 契约测试，覆盖空结果和带分页提示的结果页；测试必须解析 stdout 为单个 JSON object，并断言 stderr 只包含允许的人类提示。依赖: 0.1。Lane: output-contract。
- [x] 1.2 Owner: `cli/taskbridge`。修正 `cmd/list.go` 输出分流：JSON mode 不输出 `📭 没有找到任务` 或分页中文提示，分页、total、limit、offset、has_more 进入 JSON 字段或稳定 metadata。依赖: 1.1。Lane: output-contract。
- [x] 1.3 Owner: `cli/taskbridge`。为 table、compact、markdown、tsv 补关键 golden/contract 测试，确认人类提示只出现在 human mode，机器模式 stdout 可被稳定解析。依赖: 1.2。Lane: output-contract。
- [x] 2.1 Owner: `cli/taskbridge`。把 `pkg/output/tasks.go` 移到 `internal/taskoutput` 或等价 internal 包，删除公开 API 对 `internal/model.Task` 的依赖；迁移仓库内全部引用。依赖: 0.1。Lane: module-boundary。
- [x] 2.2 Owner: `cli/taskbridge`。定义 `TaskProjection` DTO 和 projection 函数，projection 处保留必要中文注释说明字段取舍、日期格式和截断策略。依赖: 2.1。Lane: module-boundary。
- [x] 2.3 Owner: `cli/taskbridge`。拆分 task output 文件，建议分为 `projection.go`、`json.go`、`table.go`、`delimited.go`、`markdown.go`、`format.go`；单文件目标小于 250 行，避免重新形成大文件。依赖: 2.2。Lane: module-boundary。
- [x] 3.1 Owner: `cli/taskbridge`。为 `provider.GetAllProviderNames()` 或后续 catalog API 增加顺序、包含项和去重测试；覆盖 loader、auth/list/sync/provider/storage 的默认 provider 遍历入口。依赖: 0.1。Lane: catalog-storage。
- [x] 3.2 Owner: `cli/taskbridge`。补 filestore 查询一致性测试，使用同一组 fixture 验证 `FileStorage` 与 `ProviderStorage` 的过滤、排序、分页行为一致。依赖: 0.1。Lane: catalog-storage。
- [x] 3.3 Owner: `cli/taskbridge`。清理残留硬编码 provider 列表和重复 query 分支；新增复杂条件时写中文注释解释排序和分页边界。依赖: 3.1, 3.2。Lane: catalog-storage。
- [x] 4.1 Owner: `cli/taskbridge`。补 store lifecycle RED 测试：创建临时文件并作为 `--storage-path` 运行 `taskbridge list --format json`，期望命令返回非零错误、stderr 包含存储初始化说明、stdout 不含 panic stack。依赖: 0.2。Lane: store-lifecycle。
- [x] 4.2 Owner: `cli/taskbridge`。修复 `getStore()` cleanup contract：失败时返回非 nil noop cleanup；所有直接调用点在 `err` 检查通过后再 `defer cleanup()`，覆盖 `cmd/list.go`、`cmd/analyze.go`、`cmd/lists.go`、`cmd/sync.go`、`cmd/task.go`、`cmd/serve.go` 和 `getCLIStores()` 调用链。依赖: 4.1。Lane: store-lifecycle。
- [x] 4.3 Owner: `cli/taskbridge`。增加命令级回归测试，至少覆盖 `list`、`lists`、`task add` 任一写路径在存储初始化失败时不 panic；错误输出走 stderr，stdout 保持空或机器可解析错误 envelope。依赖: 4.2。Lane: store-lifecycle。
- [x] 5.3 Owner: `cli/taskbridge`。抽出 auth provider 指引 helper，复用 `provider.GetProviderDefinition()`/catalog 元数据，删除重复的"支持的 Provider"硬编码文本；复杂分支加中文注释说明交互式登录与脚本模式差异。依赖: 5.2, 3.1。Lane: auth-exit-contract。
- [x] 6.1 Owner: `cli/taskbridge`。补共享输出 helper 契约测试，覆盖 `printStructured`、`printResult`、`wantsJSON` 的 stdout/stderr 分流、marshal error 和 pipe/quiet mode 行为。依赖: 0.2。Lane: shared-output。
- [x] 6.2 Owner: `cli/taskbridge`。将 `cmd/analyze.go` 的 quadrant/priority/time/trend/report 数据计算迁入 `internal/analyze` 或等价 internal 包，命令层只调用 service 和 shared output helper；时间边界逻辑必须有中文注释说明本地日期截断。依赖: 6.1。Lane: shared-output。
- [x] 6.3 Owner: `cli/taskbridge`。将 `cmd/lists.go`、`cmd/sync.go`、`cmd/task.go` 的手写 JSON 分支迁入共享输出 helper；所有 `json.MarshalIndent` 错误必须返回 `commandError`，不能忽略。依赖: 6.1, 2.1。Lane: shared-output。
- [x] 7.1 Owner: `cli/taskbridge`。补 storage ownership RED 测试：`FileStorage` 和 `ProviderStorage` 的 `GetTask`、`ListTasks`、`QueryTasks`、`GetTaskList` 返回值被修改后，未调用 `SaveTask`/`SaveTaskList` 时再次读取应保持原值；测试必须覆盖 `Tags`、`SubtaskIDs`、`Metadata.CustomFields` 和 `*time.Time` 字段。依赖: 0.2。Lane: persistence-integrity。
- [x] 7.2 Owner: `cli/taskbridge`。实现 `model.CloneTask`、`model.CloneTaskList` 或等价 internal clone helper；`SaveTask`、`GetTask`、`ListTasks`、`QueryTasks`、`SaveTaskList`、`GetTaskList`、`ListTaskLists` 均使用深拷贝，复杂 clone 逻辑写中文注释说明 map/slice/time pointer 的所有权。依赖: 7.1。Lane: persistence-integrity。
- [x] 7.3 Owner: `cli/taskbridge`。清理被吞掉的持久化错误：`internal/sync/engine.go`、`internal/sync/projectsync.go`、`internal/projectservice/sync.go`、`cmd/tui.go` 中的 `_ = SaveTask`、`_ = SaveTaskList`、`_ = DeleteTask`、`_ = SaveProject`、`_ = SetLastSyncTime`、`_ = Flush` 必须改为记录/返回错误；成功计数只能在本地写入成功后增加。依赖: 7.2。Lane: persistence-integrity。
- [x] 7.4 Owner: `cli/taskbridge`。补 fake failing store 测试，覆盖普通 sync、project sync、TUI delete/toggle 任一写路径；期望写失败时 result/UI 有错误信息，任务状态不被乐观改写为成功。依赖: 7.3。Lane: persistence-integrity。
- [x] 8.1 Owner: `cli/taskbridge`。决定命令级 harness：优先引入 `github.com/rogpeppe/go-internal/testscript`；如果继续使用现有 Go test harness，必须在测试文件说明原因和覆盖边界。依赖: 1.1, 4.1, 5.1, 7.1。Lane: test-evidence。
- [x] 8.2 Owner: `cli/taskbridge`。为命令级、process e2e 或 golden 测试增加 `temp/integration-test-runs/<run-id>/` evidence 输出，至少包含 `summary.json`、`command.txt`、`stdout.log`、`stderr.log`、`env.json` 和 `artifacts/`，并做 secret/provider payload 脱敏。依赖: 8.1。Lane: test-evidence。
- [x] 9.1 Owner: `cli/taskbridge`。执行全量验证：`gofmt -w` 涉及文件、`go test ./...`、`go test -race ./internal/storage/... ./internal/sync/... ./internal/projectservice/...`、`go vet ./...`、`go build ./...`、`gofmt -l .`、`git diff --check`。依赖: 1.3, 2.3, 3.3, 4.3, 5.3, 6.3, 7.4, 8.2。Lane: verification。
- [x] 9.2 Owner: `cli/taskbridge`。执行 OpenSpec 验证：`openspec validate taskbridge-code-quality-refactor --strict --no-interactive`；修正所有 proposal/design/tasks/spec 格式问题。依赖: 9.1。Lane: verification。
- [x] 9.3 Owner: `cli/taskbridge`。Closeout 记录证据和剩余风险；如果后续要把 TUI、serve 或 provider client 大文件继续拆分，另开 change，不扩大本 change。依赖: 9.2。Lane: verification。

- `output-contract`: 1.1 到 1.3，优先处理用户可见和机器可解析行为。
- `module-boundary`: 2.1 到 2.3，可在 1.1 RED 测试建立后并行推进。
- `catalog-storage`: 3.1 到 3.3，可与 output/module lanes 并行，但合并前要跑全量测试。
- `store-lifecycle`: 4.1 到 4.3，优先级高于大块拆文件，因为它是现存 panic 缺陷。
- `auth-exit-contract`: 5.1 到 5.3，可与 store-lifecycle 并行，依赖 provider catalog 测试收口硬编码文案。
- `shared-output`: 6.1 到 6.3，用同一输出 helper 吃掉 analyze/lists/sync/task show 的重复 JSON 分支。
- `persistence-integrity`: 7.1 到 7.4，处理对象所有权和持久化错误不能吞的问题。
- `test-evidence`: 8.1 到 8.2，依赖命令级测试入口明确后推进。
- `verification`: 9.1 到 9.3，所有实现 lanes 合并后执行。

## Acceptance

- `taskbridge list --format json` 在空结果和分页场景 stdout 都能被解析为单个 JSON object，且不混入中文提示、emoji 或 table 文本。
- `taskbridge list --format json --storage-path <existing-file>` 不 panic，返回非零错误，stderr 有中文错误说明，stdout 不出现 panic stack。
- Microsoft/Feishu 缺凭证、TickTick/Dida refresh token 读取失败等 auth 错误路径返回非零 exit code，不能打印错误后返回成功。
- 仓库内不存在公开 `pkg/output` API 暴露 `internal/model.Task` 的情况；如保留 `pkg/output`，只能暴露稳定 DTO 或通用 renderer contract。
- `cmd/list.go`、`cmd/analyze.go`、`cmd/lists.go`、`cmd/sync.go`、`cmd/task.go` 保持薄命令层职责，不承载过滤、排序、projection 或具体格式渲染逻辑。
- provider 默认列表只有一个 catalog 真源；filestore 查询逻辑只有一个共享实现。
- storage 读 API 返回深拷贝，调用方 mutate 返回对象不会绕过 `SaveTask`/`SaveTaskList` 污染 store 内部状态。
- sync/project/TUI 写入失败会进入错误结果或 UI 错误状态，不能被成功计数掩盖。
- 命令级测试失败时保留脱敏 evidence，成功时也能查看 summary 和 stdout/stderr artifact。
- `go test ./...`、`go test -race ./internal/storage/... ./internal/sync/... ./internal/projectservice/...`、`go vet ./...`、`go build ./...`、`gofmt -l .`、`git diff --check` 和 `openspec validate taskbridge-code-quality-refactor --strict --no-interactive` 通过。

## Failure Recheck

- 如果 JSON contract 测试失败，先检查 stdout 是否被人类提示污染，再检查 JSON envelope 字段是否缺失。
- 如果 store lifecycle 测试失败，先检查 `getStore()` 是否仍可能返回 nil cleanup，再检查调用点是否在 `err` 前 defer。
- 如果 auth exit-contract 测试失败，先搜索 `cmd/auth.go` 中错误路径的 `return nil`，再确认 command error exit code 映射。
- 如果 module boundary 测试或 build 失败，先搜索 `pkg/output` 和 `internal/model.Task` 的跨包引用，不做 shim 掩盖问题。
- 如果 provider/storage 测试失败，先确认 catalog 顺序和 query helper 是否被绕过，再排查 fixture 本身。
- 如果 persistence-integrity 测试失败，先检查 clone helper 是否遗漏嵌套 map/slice/time pointer，再搜索 `_ = .*SaveTask|_ = .*DeleteTask|_ = .*Flush` 残留。
- 如果 evidence 测试失败，先检查输出目录是否在 `temp/integration-test-runs/<run-id>/`，再检查脱敏规则和原 exit code 保留。
