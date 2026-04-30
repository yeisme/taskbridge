# TaskBridge 子项目指令

## 技术栈

- Go 1.21+
- Cobra CLI
- 文件存储为默认运行时存储，MongoDB 为可选存储
- 统一任务模型位于 `internal/model`

## 架构边界

- CLI 命令入口放在 `cmd/`。
- 业务逻辑优先放在 `internal/`，CLI 层只做参数解析、调用服务和输出。
- 用户可见稳定输出优先使用 JSON 结构体或 map，避免拼接不可解析文本。
- 本地持久化应复用 `internal/persistence/atomicjson.go` 或既有 store 模式。

## 禁止事项

- 不绕过 Provider 接口直接写远端 Todo 平台。
- 不让 Agent 或 MCP adapter 直接读写 `~/.taskbridge` 数据文件。
- 不在 `--format json` 输出中混入进度条、提示语或日志。
- 不静默执行删除、批量完成、批量改期、远端覆盖和冲突丢弃。

## 测试与质量门禁

修改代码后至少运行：

```bash
go test ./...
```

涉及 CLI 编译或命令面时运行：

```bash
go build ./...
```

提交前检查：

- 新命令有单元测试或可重复的 CLI 行为测试。
- `--dry-run` 不写本地存储，不调用远端写 API。
- JSON 模式只写 stdout；日志和提示写 stderr。
