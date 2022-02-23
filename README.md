# seata-golang
[![LICENSE](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](https://github.com/opentrx/seata-golang/blob/v2/LICENSE)

## Introduction | [中文](https://github.com/opentrx/seata-golang/blob/v2/docs/README_ZH.md)
seata-golang is a distributed transaction middleware based on Golang.
### difference between seata-glang and [seata](https://github.com/seata/seata)
| feature   | seata | seata-golang | remark                                                         |
|-----------|:-----:|:------------:|----------------------------------------------------------------|
| AT mode   |   ✅   |      ✅       |                                                                |
| TCC mode  |   ✅   |      ✅       |                                                                |
| SAGA mode |   ✅   |      ☑️      |                                                                |
| rpc       |   ✅   |      ✅       | [dev branch](https://github.com/opentrx/seata-golang/tree/dev) |
| grpc      |  ☑️   |      ✅       | [v2 branch](https://github.com/opentrx/seata-golang/tree/v2)   |

## Architecture
<img alt="seata-flow" width="500px" src="https://github.com/opentrx/seata-golang/blob/v2/docs/images/seata-flow.png" />  

A typical lifecycle of Seata managed distributed transaction:

- TM asks TC to begin a new global transaction. TC generates an XID representing the global transaction.
- XID is propagated through microservices' invoke chain.
- RM registers local transaction as a branch of the corresponding global transaction of XID to TC.
- TM asks TC for committing or rolling back the corresponding global transaction of XID.
- TC drives all branch transactions under the corresponding global transaction of XID to finish branch committing or rolling back.

## Directory structure
- cmd: to startup TC server
	- profiles/dev/config.yml: TC config file
	- tc/main.go: TC startup entrance
- dist: to build in docker container
- docs: documentations
- pkg: TC + RM + TM implementation
	- server/db/*.sql: sql scripts to create DB and tables for TC

## Getting started
- ### TC server
```bash
cd ${projectpath}/cmd/tc
go build -o tc_server
# create database `seata` for TC server
# update storage.dsn.mysql in ${projectpath}/cmd/profiles/dev/config.yml
./tc_server start -config ${projectpath}/cmd/profiles/dev/config.yml
```

- ### Client
Please refer to demo [seata-go-samples](https://github.com/opentrx/seata-go-samples)

- ### Prerequisites
  - MySQL server
  - Golang version >= 1.15
  - Business tables that require primary key

## Design and implementation
The seata-golang AT and TCC design are actually the same as [seata](https://github.com/seata/seata).  
Please refer to [what-is-seata](https://seata.io/en-us/docs/overview/what-is-seata.html) for more details.

## Reference
- [what is seata AT mode?](https://seata.io/en-us/docs/dev/mode/at-mode.html)
- [what is seata TCC mode?](https://seata.io/en-us/docs/dev/mode/tcc-mode.html)
- [GRPC](https://grpc.io/)
- [dubbogo](https://github.com/dubbogo)
- [mysql-driver](https://github.com/opentrx/mysql)
- [seata-go-samples](https://github.com/opentrx/seata-go-samples)

## Contact
Please contact us via DingTalk app if you have any issues. The chat group ID is 33069364.  
<img alt="DingTalk Group" src="https://github.com/opentrx/seata-golang/blob/dev/docs/pics/33069364.png" width="200px" />

## Contributing
Welcome to raise up issue or pull-request to seata-golang!  
To contribute, fork from opentrx/seata-golang and push branch to your repo, then open a pull-request.

## License
seata-golang software is licenced under the Apache License Version 2.0. See the [LICENSE](https://github.com/opentrx/seata-golang/blob/v2/LICENSE) file for details.
