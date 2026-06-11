# go-dev-toolchain Specification

## Purpose
定义 TaskBridge Go 辅助开发工具链的本地开发、质量门禁、构建输出、发布校验和可选热加载要求。

## Requirements
### Requirement: Go 开发工具链标准化落地

本项目 SHALL 按照根 go-dev-toolchain-quality 设计落地 golangci-lint v2 配置、Taskfile 任务语义对齐和可选热加载。

#### Scenario: lint 配置覆盖基线

- **WHEN** 本项目完成 .golangci.yaml 配置
- **THEN** 配置 SHALL 覆盖基线 linters（errcheck、govet、ineffassign、staticcheck、unused、misspell、revive）和 formatters（gofmt、goimports）
- **AND** golangci-lint run --new-from-rev=HEAD~ SHALL 退出码 0

#### Scenario: Taskfile 任务语义对齐

- **WHEN** 本项目完成 Taskfile 更新
- **THEN** Taskfile SHALL 提供 tools:install、deps、mod-check、fmt、fmt-check、lint、lint:new、test、test:race、build、snapshot、release:check、release:local、ci/check 任务
- **AND** task ci SHALL 至少执行依赖下载、模块检查、格式检查、lint、测试和构建
- **AND** task build SHALL 输出当前平台二进制到 `dist/taskbridge` 并 smoke-run `dist/taskbridge --help`
- **AND** 可选 Air 热加载 SHALL 通过 dev:watch 暴露，且不作为 ci/check 依赖

#### Scenario: 项目特有任务保留

- **WHEN** 本项目已有特有任务
- **THEN** 工具链落地 MUST 保留这些特有任务
- **AND** 只允许为命名一致性增加别名或补充说明

