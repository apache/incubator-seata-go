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

- [[#904](https://github.com/apache/incubator-seata-go/issues/904)] 修复 XA autoCommit 下 `SELECT ... FOR UPDATE` 后紧接其他语句导致的 "busy buffer" / "driver: bad connection"：将分支提交（XA END + XA PREPARE）延迟到查询结果集关闭之后再执行
- [[#130](https://github.com/apache/incubator-seata-go/pull/130)] 修复getty session自动关闭的bug

### optimize：

- [[#125](https://github.com/apache/incubator-seata-go/pull/125)] 优化resourceManagerApi和tccResource功能

### test:

- [[#xxx](https://github.com/apache/incubator-seata-go/pull/xxx)] 添加xxx的单元测试


### contributors:

非常感谢以下 contributors 的代码贡献。若有无意遗漏，请报告。

- [slievrly](https://github.com/slievrly)

同时，我们收到了社区反馈的很多有价值的issue和建议，非常感谢大家。
