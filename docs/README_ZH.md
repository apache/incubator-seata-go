# seata-golang
[![LICENSE](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](https://github.com/opentrx/seata-golang/blob/v2/LICENSE)

## 简介 | [English](https://github.com/opentrx/seata-golang/blob/v2/README.md)
seata-golang是一个用于解决分布式事务的中间件，是基于Go语言版本的seata。
### seata-golang与[seata](https://github.com/seata/seata) 的区别
- seata-golang支持AT模式与TCC模式，seata支持AT、TCC、Saga模式。
- seata-golang支持grpc，而seata不支持。

## 架构
<img alt="seata-flow" src="https://github.com/opentrx/seata-golang/blob/v2/docs/images/seata-flow.png" />

## 功能
- AT 模式
- TCC 模式
- GRPC (v2 分支, 更贴近云原生)
- RPC (dev 分支)
- MySQL 驱动

##目录结构
- cmd: 启动TC server的入口
	- profiles/dev/config.yml: TC 配置文件
	- tc/main.go: TC 启动文件
- dist: docker环境
- docs: 相关文档
- pkg: TC + RM + TM 核心模块实现
	- server/db/*.sql: 用于启动TC所必须的创建数据库表的SQL

## 启动方法
- ### TC server
```bash
cd ${projectpath}/cmd/tc
go build -o tc_server
# 修改配置文件 ${projectpath}/cmd/profiles/dev/config.yml 的配置项 storage.dsn.mysql
./tc_server start -config ${projectpath}/cmd/profiles/dev/config.yml
```
- ### Client
请查看demo演示[seata-go-samples](https://github.com/opentrx/seata-go-samples)

- ### 前提条件
    - MySQL服务器
    - Golang应用
    - 带主键的业务数据表

## 设计与实现
seata-golang的AT模式和TCC模式的设计与[seata](https://github.com/seata/seata) 是一致的。  
请参考[什么是seata](https://seata.io/en-us/docs/overview/what-is-seata.html)

## 路线图
- [什么是seata AT模式？](https://seata.io/en-us/docs/dev/mode/at-mode.html)
- [什么是seata TCC模式？](https://seata.io/en-us/docs/dev/mode/tcc-mode.html)
- [GRPC](https://grpc.io/)

## 相关项目
- [dubbogo](https://github.com/dubbogo)
- [msyql-driver](https://github.com/opentrx/mysql)
- [seata-go-samples](https://github.com/opentrx/seata-go-samples)

## 联系方式
如果对seata-golang有问题，可以通过钉钉联系我们。钉钉群号是 33069364。  
<img alt="DingTalk Group" src="https://github.com/opentrx/seata-golang/blob/dev/docs/pics/33069364.png" width="200px" />

## 贡献
欢迎来为seata-golang提交issue和pull-request！
要给seata-golang提交代码, 可以fork opentrx/seata-golang，然后提交代码到你fork的仓库分支上，最后提交pull request。

## 开源协议
seata-golang遵循Apache 2.0开源协议。 查看 [LICENSE](https://github.com/opentrx/seata-golang/blob/v2/LICENSE)