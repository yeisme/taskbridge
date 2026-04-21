# TaskBridge Token 自动刷新

## 目标

确保 OAuth2 Provider 的 token 不会因为过期而让 CLI 同步或后台服务中断。

## 当前入口

```bash
taskbridge auth status
taskbridge auth refresh <provider>
taskbridge serve --enable-auto-refresh
```

## 设计原则

1. 状态查看走 CLI，不依赖协议服务。
2. 自动刷新由 `serve` 后台服务负责。
3. Provider 刷新逻辑仍由各自实现持有。

## 后续演进方向

1. 在 `auth status` 中显示更丰富的过期预警信息。
2. 在 `serve` 中补充健康检查与刷新统计。
3. 将定时同步与 token 刷新整合成更清晰的后台服务生命周期。
