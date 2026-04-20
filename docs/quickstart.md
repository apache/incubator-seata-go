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

# Seata-go Quick Start

## Prerequisites

- Go >= 1.20
- Java >= 8
- MySQL >= 8.0

### Start Seata Server (Binary)

1. Download the binary distribution from the official [Seata-Server Release History](https://seata.apache.org/release-history/seata-server/) page and extract it.
2. Enter the extracted Seata Server directory.
3. Start the server with `file` storage mode:

```bash
sh ./seata-server/bin/seata-server.sh -p 8091 -h 127.0.0.1 -m file
```

> - `-p 8091`: specifies the Seata Server port.
> - `-h 127.0.0.1`: specifies the registry or advertised server address.
> - `-m file`: stores transaction logs in `file` mode, which is suitable for a local quick start.

4. Confirm that Seata Server is running and listening on `127.0.0.1:8091`.

### Start Seata Server (Docker)

1. Pull the image from the official Docker Hub repository [`apache/seata-server`](https://hub.docker.com/r/apache/seata-server):

```bash
docker pull apache/seata-server:<seata-version>
```

2. Start a container from the image you just pulled:

```bash
docker run --name seata-server \
  -p 8091:8091 \
  -e STORE_MODE=file \
  apache/seata-server:<seata-version>
```

> If local Docker has limited available memory and startup fails with `There is insufficient memory for the Java Runtime Environment to continue` or `Cannot allocate memory`, you can explicitly lower the JVM heap size, for example:
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
> Add `-d` if you want to run it in the background.

3. Check the logs and confirm that the service is ready:

```bash
docker logs -f seata-server
```

4. Confirm that the client can reach `127.0.0.1:8091` before continuing.

If you later need to switch Seata Server to Nacos or another registry center, adjust the Seata Server-side `registry.conf` and `application.yaml`.

If your Seata Server uses Nacos, Seata Server 1.4.x and earlier server-side configuration is usually written in `registry.conf`, while Seata Server 1.5.0 and later usually uses `application.yaml`. For 1.4.x and earlier, for example:

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

> `registry.conf`: official parameter reference and Nacos config-center example: [Parameter Configuration](https://seata.apache.org/docs/user/configurations) · [Nacos Configuration Center](https://seata.apache.org/docs/user/configuration/nacos/)

Seata Server 1.5.0 and later usually uses `application.yaml` as the server-side configuration file in the Nacos scenario:

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

> `application.yaml`: official parameter reference and Nacos registry example: [Parameter Configuration](https://seata.apache.org/docs/user/configurations) · [Nacos Registry Center](https://seata.apache.org/docs/user/registry/nacos/)

`seatago.yml` is the Seata Go client configuration file in this repository. Use it to adjust the client-side registry center, transaction group, and server address. For example:

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

References for the configuration structure and a full sample:

- `seatago.yml`: [Config struct](../pkg/client/config.go) · [Full sample](../testdata/conf/seatago.yml)

## Quick Integration

Install the dependency:

```bash
go get seata.apache.org/seata-go/v2@latest
```

Initialize the Seata client:

```go
package main

import (
	"context"

	"seata.apache.org/seata-go/v2/pkg/client"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

func main() {
	// Initialize the Seata client
	client.InitPath("./conf/seatago.yml")
	// Or set SEATA_GO_CONFIG_PATH=/path/to/your/seatago.yml first, then call:
	// client.Init()

	ctx := context.Background()

	// Run business logic in a global transaction
	err := tm.WithGlobalTx(ctx, &tm.GtxConfig{Name: "my-tx"}, func(ctx context.Context) error {
		// Business logic
		return nil
	})
	if err != nil {
		panic(err)
	}
}
```

## Example Scenarios

### AT Example

AT mode wraps business logic with `tm.WithGlobalTx(...)` and uses the `seata-at-mysql` driver to intercept database access:

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

Before using AT mode, make sure the business database has already created the [undo_log](../testdata/sql/undo_log.sql) table.

> Full example: [AT Example](https://github.com/apache/incubator-seata-go-samples/tree/main/at/basic).

### TCC Example

TCC mode uses `tcc.NewTCCServiceProxy(...)` to register Try/Confirm/Cancel actions, then chains them with `tm.WithGlobalTx(...)`:

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

If you need fence mode, set `seata.tcc.fence.enable: true` in `seatago.yml` and create the `tcc_fence_log` table.

> Full example: [TCC Example](https://github.com/apache/incubator-seata-go-samples/tree/main/tcc/local).

### XA Example

The current repository supports MySQL XA. The transaction entrypoint is the same as AT; you only need to switch the driver to `seata-xa-mysql`:

```go
db, err := sql.Open(
	"seata-xa-mysql",
	"root:password@tcp(127.0.0.1:3306)/seata_demo?charset=utf8mb4&parseTime=True&multiStatements=true",
)
```

> Full example: [XA Example](https://github.com/apache/incubator-seata-go-samples/tree/main/xa/basic).
