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

### Development notes

### dev version

  Seata-go is an easy-to-use, high-performance, open source distributed transaction solution.

  The version is updated as follows:	

### feature：

  - [[#123](https://github.com/apache/incubator-seata-go/pull/123)] add two phase and dubbo
  - support XA branch enrollment for autoCommit statements in a global transaction: each autoCommit statement is registered and prepared as its own complete XA branch (note: N autoCommit statements create N branches at the TC); parameterized statements (which the default go-sql-driver DSN answers with `driver.ErrSkip`) are executed via an in-branch Prepare+Exec fallback so they stay inside the branch
  - support PostgreSQL XA via pgx driver
  - [[#1130](https://github.com/apache/incubator-seata-go/issues/1130)] support MySQL multi-value INSERT in AT mode for composite and mixed primary keys

### bugfix：

  - [[#904](https://github.com/apache/incubator-seata-go/issues/904)] fix "busy buffer" / "driver: bad connection" when a `SELECT ... FOR UPDATE` is followed by another statement under XA autoCommit, by deferring the branch commit (XA END + XA PREPARE) until the query rows are closed
  - [[#130](https://github.com/apache/incubator-seata-go/pull/130)] getty session auto close bug
  - [[#991](https://github.com/apache/incubator-seata-go/issues/991)] fix connection leaks and prevent nil pointer panic in async worker
  - [[#887](https://github.com/apache/incubator-seata-go/issues/887)] make DayValue serialization timezone-stable

### optimize：

  - [[#125](https://github.com/apache/incubator-seata-go/pull/125)] optimize named for the resource manager api and tcc resource

### test:

  - [[#958](https://github.com/apache/incubator-seata-go/issues/958)] improve test coverage for pkg/rm/tcc/fence/store/db/sql (100%)
  - [[#957](https://github.com/apache/incubator-seata-go/issues/957)] improve test coverage for pkg/rm/tcc/fence/handler (84.5%)
  - [[#955](https://github.com/apache/incubator-seata-go/issues/955)] improve test coverage for pkg/rm/tcc/fence (95.1%)
  - [[#947](https://github.com/apache/incubator-seata-go/issues/947)] improve test coverage for pkg/discovery/mock
  - [[#948](https://github.com/apache/incubator-seata-go/issues/948)] improve test coverage for pkg/integration	

### contributors:

Thanks to these contributors for their code commits. Please report an unintended omission.  

- [slievrly](https://github.com/slievrly)

Also, we receive many valuable issues, questions and advices from our community. Thanks for you all.	
