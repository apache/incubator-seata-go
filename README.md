# seata-golang
[![LICENSE](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](https://github.com/opentrx/seata-golang/blob/v2/LICENSE)

## Introduction | [中文](https://github.com/opentrx/seata-golang/blob/v2/docs/README_ZH.md)
seata-golang is a distributed transaction middleware based on Golang.
### difference between seata-glang and [seata](https://github.com/seata/seata)
- Currently, seata-golang only support AT mode and TCC mode
- seata-golang supports bidirectional grpc streaming while seata not

## Architecture
<img alt="seata-flow" src="https://github.com/opentrx/seata-golang/blob/v2/docs/images/seata-flow.png" />

## Features
- AT mode
- TCC mode
- GRPC (v2 branch, more friendly to cloud-native)
- RPC (dev branch)
- MySQL driver

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
# update storage.dsn.mysql in ${projectpath}/cmd/profiles/dev/config.yml
./tc_server start -config ${projectpath}/cmd/profiles/dev/config.yml
```
- ### Client
Please refer to [seata-go-samples](https://github.com/opentrx/seata-go-samples)

- ### Prerequisites
  - MySQL server
  - Golang applications that need distributed transaction

## Design and implementation
The seata-golang AT and TCC design are actually same as [seata](https://github.com/seata/seata).  
Please refer to [what-is-seata](https://seata.io/en-us/docs/overview/what-is-seata.html) for more details.

## Roadmap
- [what is seata AT mode?](https://seata.io/en-us/docs/dev/mode/at-mode.html)
- [what is seata TCC mode?](https://seata.io/en-us/docs/dev/mode/tcc-mode.html)
- [GRPC](https://grpc.io/)

## Built With
- [dubbogo](https://github.com/dubbogo)
- [msyql-driver](https://github.com/opentrx/mysql)
- [seata-go-samples](https://github.com/opentrx/seata-go-samples)

## Contact
Please contact us via DingTalk app if you have any issues. The chat group ID is 33069364.  
<img alt="DingTalk Group" src="https://github.com/opentrx/seata-golang/blob/dev/docs/pics/33069364.png" width="200px" />

## Contributing
Welcome to contribute to seata-golang!  
To contribute, fork from opentrx/seata-golang and push branch to your repo, then open a pull-request.

## License
seata-golang software is licenced under the Apache License Version 2.0. See the [LICENSE](https://github.com/opentrx/seata-golang/blob/v2/LICENSE) file for details.
