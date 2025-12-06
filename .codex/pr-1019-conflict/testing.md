## 测试记录（2025-12-06）

- 命令：`go test ./pkg/client/... ./pkg/remoting/loadbalance/... ./pkg/rm/... ./pkg/tm/... ./pkg/util/...`
  - 结果：失败。`pkg/tm` 在 macOS ARM64 下 gomonkey 打补丁触发 SIGBUS（`TestTransactionExecutorBegin`），同时 `SQLUndoDirtyError` 枚举偏移导致断言失败（已修正）。
- 命令：`go test ./pkg/util/errors`
  - 结果：通过，验证错误码顺序调整后恢复。
- 命令：`go test ./pkg/client/... ./pkg/remoting/loadbalance/... ./pkg/rm/... ./pkg/util/...`
  - 结果：通过（仅 ld_classic 库链接警告），覆盖改动相关包（不含 `pkg/tm` 以规避 gomonkey 崩溃）。
