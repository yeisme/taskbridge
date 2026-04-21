# TaskBridge 版本与发布流程

当前稳定版本：`v1.0.3`（截至 2026-03-07）

## 目标

- 统一版本来源（CLI、后台服务、发行产物使用同一版本号）
- 标准化发布步骤（tag + GoReleaser）
- 让用户直接安装和运行 `taskbridge` CLI

## 版本号来源

统一在 [pkg/buildinfo/buildinfo.go](/Users/yeshugen/workspace/taskbridge/pkg/buildinfo/buildinfo.go)：

- `Version`
- `GitCommit`
- `BuildDate`

## 当前打包矩阵

- Linux: `amd64` / `arm64`
- macOS: `amd64` / `arm64`
- Windows: `amd64` / `arm64`
- 源码包：`taskbridge_source.tar.gz`
- Linux 包管理格式：`.deb` / `.rpm` / `.apk`

## 发布前检查

```bash
go test ./...
go build ./...
goreleaser check
```

## 发布步骤

```bash
git add .
git commit -m "release: v1.0.3"
git tag -a v1.0.3 -m "Release v1.0.3"
git push origin main
git push origin v1.0.3
goreleaser release --clean
```
