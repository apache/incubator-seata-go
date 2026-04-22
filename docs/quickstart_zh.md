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

# Seata-go 快速开始

## 环境准备

- Go >= 1.20
- Java >= 8
- MySQL >= 8.0

### 启动 Seata Server（二进制）

1. 从 Seata 官网的 [Seata-Server 版本历史](https://seata.apache.org/zh-cn/release-history/seata-server/) 页面下载二进制发行包并解压。
2. 进入解压后的 Seata Server 目录。
3. 使用 `file` 存储模式启动服务：

```bash
sh ./seata-server/bin/seata-server.sh -p 8091 -h 127.0.0.1 -m file
```

> - `-p 8091`：指定 Seata Server 端口。
> - `-h 127.0.0.1`：指定服务注册地址或对外暴露地址。
> - `-m file`：使用 `file` 模式存储事务日志，适合本地快速开始。

4. 确认 Seata Server 成功启动并监听 `127.0.0.1:8091`。

### 启动 Seata Server（Docker）

1. 从 Docker Hub 官方仓库 [`apache/seata-server`](https://hub.docker.com/r/apache/seata-server) 拉取镜像：

```bash
docker pull apache/seata-server:<seata-version>
```

2. 使用刚刚拉取的镜像启动容器：

```bash
docker run --name seata-server \
  -p 8091:8091 \
  -e STORE_MODE=file \
  apache/seata-server:<seata-version>
```

> 如果本地 Docker 可用内存较小，启动时出现 `There is insufficient memory for the Java Runtime Environment to continue` 或 `Cannot allocate memory`，可以显式调低 JVM 堆大小，例如：
>
> ```bash
> docker run --name seata-server \
>   -p 8091:8091 \
>   -e STORE_MODE=file \
>   -e JVM_XMS=512m \
>   -e JVM_XMX=512m \
>   apache/seata-server:<seata-version>
> ```
>
> 如需后台运行，可额外添加 `-d`。

3. 查看启动日志，确认服务已就绪：

```bash
docker logs -f seata-server
```

4. 确认客户端可以访问 `127.0.0.1:8091`。

如果后面需要将 Seata Server 切换到 Nacos 或其他注册中心，可对照 Seata Server 侧的 `registry.conf` 与 `application.yaml` 进行修改。

如果你的 Seata Server 使用 Nacos 部署，Seata Server 1.4.x 的服务端配置通常写在 `registry.conf` 中，例如：

```hocon
registry {
  type = "nacos"
  nacos {
    application = "seata-server"
    serverAddr = "127.0.0.1:8848"
    group = "SEATA_GROUP"
    namespace = ""
    cluster = "default"
    username = ""
    password = ""
  }
}

config {
  type = "nacos"
  nacos {
    serverAddr = "127.0.0.1:8848"
    group = "SEATA_GROUP"
    namespace = ""
    username = ""
    password = ""
  }
}
```

> `registry.conf`：官方参数总表与 Nacos 配置中心示例：[参数配置](https://seata.apache.org/zh-cn/docs/user/configurations) · [Nacos 配置中心](https://seata.apache.org/zh-cn/docs/user/configuration/nacos/)

Seata Server 1.5.0 及以后在 Nacos 场景下通常使用 `application.yaml` 作为服务端配置文件：

```yaml
seata:
  registry:
    type: nacos
    nacos:
      application: seata-server
      server-addr: 127.0.0.1:8848
      group: SEATA_GROUP
      namespace: ""
      cluster: default
      username: ""
      password: ""
  config:
    type: nacos
    nacos:
      server-addr: 127.0.0.1:8848
      group: SEATA_GROUP
      namespace: ""
      data-id: seataServer.properties
      username: ""
      password: ""
```

> `application.yaml`：官方参数总表与 Nacos 注册中心示例：[参数配置](https://seata.apache.org/zh-cn/docs/user/configurations) · [Nacos 注册中心](https://seata.apache.org/zh-cn/docs/user/registry/nacos/)

`seatago.yml` 是当前仓库的 Seata Go 客户端配置文件，用于调整客户端的注册中心、事务分组与服务地址。例如：

```yaml
seata:
  application-id: quickstart-demo
  tx-service-group: default_tx_group
  data-source-proxy-mode: AT

  service:
    vgroup-mapping:
      default_tx_group: default
    grouplist:
      default: 127.0.0.1:8091

  registry:
    type: file

  client:
    tm:
      default-global-transaction-timeout: 60s
    rm:
      lock:
        retry-interval: 30s
        retry-times: 10
        retry-policy-branch-rollback-on-conflict: true
    undo:
      log-serialization: json
      log-table: undo_log
      only-care-update-columns: true

  tcc:
    fence:
      enable: false
```

配置结构与完整样例参考如下：

- `seatago.yml`：[配置结构定义](../pkg/client/config.go) · [完整样例](../testdata/conf/seatago.yml)

## 快速集成

安装依赖：

```bash
go get seata.apache.org/seata-go/v2@latest
```

初始化 Seata 客户端：

```go
package main

import (
	"context"

	"seata.apache.org/seata-go/v2/pkg/client"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

func main() {
	// 初始化 Seata 客户端
	client.InitPath("./conf/seatago.yml")
	// 或者先设置 SEATA_GO_CONFIG_PATH=/path/to/your/seatago.yml，再调用：
	// client.Init()

	ctx := context.Background()

	// 使用分布式事务
	err := tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "my-tx"}, func(ctx context.Context) error {
		// 业务逻辑
		return nil
	})
	if err != nil {
		panic(err)
	}
}
```

## 示例场景

### AT 模式示例

AT 模式通过 `tm.WithGlobalTx(...)` 包裹业务逻辑，并使用 `seata-at-mysql` 驱动接管数据库访问：

```go
import (
	"context"
	"database/sql"

	"seata.apache.org/seata-go/v2/pkg/tm"
)

db, err := sql.Open(
	"seata-at-mysql",
	"root:password@tcp(127.0.0.1:3306)/seata_demo?charset=utf8mb4&parseTime=True&multiStatements=true",
)
if err != nil {
	return err
}

ctx := context.Background()
err = tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "create-order"}, func(ctx context.Context) error {
	_, err := db.ExecContext(ctx,
		"UPDATE account SET balance = balance - ? WHERE user_id = ?",
		100,
		1,
	)
	return err
})
```

使用 AT 模式前，请确认业务库已经创建 [undo_log](../testdata/sql/undo_log.sql) 表。

> 完整示例可参考：[AT 模式示例](https://github.com/apache/incubator-seata-go-samples/tree/main/at/basic)。

### TCC 模式示例

TCC 模式通过 `tcc.NewTCCServiceProxy(...)` 注册 Try/Confirm/Cancel 动作，再由 `tm.WithGlobalTx(...)` 串联全局事务：

```go
proxy, err := tcc.NewTCCServiceProxy(&InventoryTCC{})
if err != nil {
	return err
}

return tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "inventory-tcc"}, func(ctx context.Context) error {
	_, err := proxy.Prepare(ctx, ReserveRequest{OrderID: 1001})
	return err
})
```

如果需要开启 fence，可在 `seatago.yml` 中设置 `seata.tcc.fence.enable: true`，并创建 `tcc_fence_log` 表。

> 完整示例可参考：[TCC 模式示例](https://github.com/apache/incubator-seata-go-samples/tree/main/tcc/local)。

### XA 模式示例

当前仓库已支持 MySQL XA，事务入口与 AT 相同，只需要将驱动切换为 `seata-xa-mysql`：

```go
db, err := sql.Open(
	"seata-xa-mysql",
	"root:password@tcp(127.0.0.1:3306)/seata_demo?charset=utf8mb4&parseTime=True&multiStatements=true",
)
```

> 完整示例可参考：[XA 模式示例](https://github.com/apache/incubator-seata-go-samples/tree/main/xa/basic)。
