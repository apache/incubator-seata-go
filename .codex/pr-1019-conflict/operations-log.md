## Operations Log
- 2025-12-06: Fetch PR #1019 (feature/saga) from origin，尝试合并 origin/master 暴露冲突文件（go.mod、client.go、loadbalance 等）。
- 2025-12-06: 完成阶段0上下文收集，生成 structured-request、context-scan、关键疑问与充分性检查文件。
- 2025-12-06: 统一模块路径为 seata.apache.org/seata-go，合并 go.mod/go.sum 与 goimports.sh，并批量替换旧导入路径。
- 2025-12-06: 对齐客户端 remoting 初始化为 InitGetty，保留 Saga/TCC 初始化；删除废弃 rpc_client；合并负载均衡新算法与测试、RM/TM 错误处理及错误码。
- 2025-12-06: 运行 gofmt 与 go test（排除 pkg/tm 时通过；包含 pkg/tm 时报 gomonkey 在 macOS 上 SIGBUS），记录测试结果与风险。
