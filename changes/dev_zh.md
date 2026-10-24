<!--
 Licensed to the Apache Software Foundation (ASF) under one or more
 contributor license agreements.  See the NOTICE file distributed with
 this work for additional information regarding copyright ownership.
 The ASF licenses this file to You under the Apache License, Version 2.0
 (the "License"); you may not use this file except in compliance with
 the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
 -->

### 开发记录


### dev version

Seata-go 是一款开源的分布式事务解决方案，提供高性能和简单易用的分布式事务服务。

此版本更新如下：

### feature：

- [[#123](https://github.com/apache/incubator-seata-go/pull/123)] 添加二阶段事务接口，以及dubbo集成
- 支持全局事务中 autoCommit 语句的 XA 分支注册：每条 autoCommit 语句都作为一个完整的 XA 分支单独注册并 prepare（注意：N 条 autoCommit 语句会在 TC 侧产生 N 个分支）；带参数的语句（默认 go-sql-driver DSN 会返回 `driver.ErrSkip`）通过分支内 Prepare+Exec 回退执行，从而保证仍在分支内完成
- 支持基于 pgx 驱动的 PostgreSQL XA
- [[#1130](https://github.com/apache/incubator-seata-go/issues/1130)] 支持 AT 模式下 MySQL 多值 INSERT 的复合主键与混合主键场景

### bugfix：

- [[#128](https://github.com/apache/incubator-seata-go/issues/128)] 修复 remoting session 重连后未重新注册缓存 RM 资源的问题
- [[#1190](https://github.com/apache/incubator-seata-go/issues/1190)] 修复 UPDATE JOIN 后镜像查询，使用前镜像主键定位更新记录，并正确处理外连接中未匹配的目标表
- [[#1207](https://github.com/apache/incubator-seata-go/pull/1207)] 修复同类型多数据源的元数据缓存覆盖问题，并保留 PostgreSQL 表身份中的 schema。
- [[#1201](https://github.com/apache/incubator-seata-go/issues/1201)] TCC 二阶段方法返回 false 时报告可重试的分支失败状态
- [[#904](https://github.com/apache/incubator-seata-go/issues/904)] 修复 XA autoCommit 下 `SELECT ... FOR UPDATE` 后紧接其他语句导致的 "busy buffer" / "driver: bad connection"：将分支提交（XA END + XA PREPARE）延迟到查询结果集关闭之后再执行
- [[#130](https://github.com/apache/incubator-seata-go/pull/130)] 修复getty session自动关闭的bug
- [[#1157](https://github.com/apache/incubator-seata-go/issues/1157)] 修复 TCC 分支提交和回滚处理 malformed applicationData 时 RM panic 的问题
- [[#1159](https://github.com/apache/incubator-seata-go/issues/1159)] 修复同步请求超时、以及无人等待回复的消息发送后 message future 泄漏，并避免重复响应阻塞传输层接收循环
- [[#1153](https://github.com/apache/incubator-seata-go/issues/1153)] 修复 context 已取消时全局事务二阶段的处理：提交请求从未发出却返回成功，回滚则对空响应做类型断言而 panic
- [[#1143](https://github.com/apache/incubator-seata-go/issues/1143)] 按 MySQL 驱动实际返回的 `[]byte` 读取 `XA RECOVER` 的 xid，并关闭结果集
- 修复 `loadbalance.Select()` 在 `load-balance.type` 配置为 `ConsistentHashLoadBalance` / `LeastActiveLoadBalance` 时静默回退为随机负载均衡的问题：两个策略均已实现且配置层已接受，仅分发 `switch` 缺少对应分支
- [[#1073](https://github.com/apache/incubator-seata-go/issues/1073)] 修复上述两个策略在真正可达之后暴露出的问题：负载均衡 key 从整个 `RpcMessage` 而非其 body 中提取，导致一致性哈希把所有请求固定到同一台 TC；`rpc.Status.GetActive` / `GetTotal` 未用 `sync/atomic` 读取在途计数；首次选择时若还没有 session，哈希环会一直为空，且 key 越过最后一个虚拟节点时不会环绕到环首
- [[#1161](https://github.com/apache/incubator-seata-go/pull/1161)] 修复嵌套事务中的上下文隔离，以恢复外层发起者的提交

### 破坏性变更：

- [[#1104](https://github.com/apache/incubator-seata-go/issues/1104)] 将最低 Go 版本从 1.20 升级到 1.25（上游 `github.com/pingcap/tidb/pkg/parser` 要求）。所有下游用户需升级到 Go 1.25+。

### optimize：

- [[#125](https://github.com/apache/incubator-seata-go/pull/125)] 优化resourceManagerApi和tccResource功能
- [[#1104](https://github.com/apache/incubator-seata-go/issues/1104)] 将 `github.com/arana-db/parser` 替换为上游 `github.com/pingcap/tidb/pkg/parser`，避免依赖可能停更的 fork

### test:

- [[#xxx](https://github.com/apache/incubator-seata-go/pull/xxx)] 添加xxx的单元测试


### docs：

  - [[#1179](https://github.com/apache/incubator-seata-go/issues/1179)] 修复 README 安装命令使用 `/v2` module 路径（`go get seata.apache.org/seata-go/v2@latest`），并修正失效的 release/issue/PR 模板链接

### contributors:

非常感谢以下 contributors 的代码贡献。若有无意遗漏，请报告。

- [slievrly](https://github.com/slievrly)

同时，我们收到了社区反馈的很多有价值的issue和建议，非常感谢大家。
