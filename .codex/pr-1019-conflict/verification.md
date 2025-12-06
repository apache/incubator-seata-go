## 验证结论（2025-12-06）

- **状态**：部分通过。
- **已验证**：
  - `go test ./pkg/util/errors` 通过，确认错误码顺序修正。
  - `go test ./pkg/client/... ./pkg/remoting/loadbalance/... ./pkg/rm/... ./pkg/util/...` 通过（ld_classic 警告），覆盖客户端、负载均衡、RM 及通用工具包。
- **未通过/风险**：
  - `go test ./pkg/tm/...` 在 macOS ARM64 上因 gomonkey 打补丁导致 SIGBUS（见 testing.md），需在支持环境或禁用补丁模式下复测。
