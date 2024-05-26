
# Seata-go: Simple Extensible Autonomous Transaction Architecture(Go version)

[![CI](https://github.com/apache/incubator-seata-go/actions/workflows/license.yml/badge.svg)](https://github.com/apache/incubator-seata-go/actions/workflows/license.yml)
[![license](https://img.shields.io/github/license/apache/incubator-seata-go.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

[简体中文 ZH](./README_ZH.md)

## What is seata-go?

Apache Seata(incubating) is a very mature distributed transaction framework, and is the de facto standard platform for distributed transaction technology in the Java field. Seata-go is the implementation version of go language in Seata multilingual ecosystem, which realizes the interoperability between Java and Go, so that Go developers can also use seata-go to realize distributed transactions. Please visit the [official website of Seata](https://seata.apache.org/) to view the quick start and documentation.

The principle of seata-go is consistent with that of Seata-java, which is composed of TM, RM and TC. The functions of TC reuse Java, and the functions of TM and RM will be aligned with Seata-java later. The overall process is as follows:

![](https://user-images.githubusercontent.com/68344696/145942191-7a2d469f-94c8-4cd2-8c7e-46ad75683636.png)

## TODO list

- [x] TCC
- [ ] XA
- [x] AT
  - [x] Insert SQL
  - [x] Delete SQL
  - [x] Insert on update SQL
  - [x] Multi update SQL
  - [x] Multi delete SQL
  - [x] Select for update SQL
  - [x] Update SQL
- [ ] SAGA
- [x] TM
- [x] RPC communication
- [x] Transaction anti suspension
  - [x] Manually way
  - [x] Proxy datasource way 
- [x] Null compensation
- [ ] Configuration center
  - [x] Configuration file
- [ ] Registration Center
- [ ] Metric monitoring
- [x] Compressor algorithm
- [x] Examples


## How to run？

if you want to know how to use and integrate seata-go, please refer to [apache/seata-go-samples](https://github.com/apache/incubator-seata-go-samples)

## How to join us？

Seata-go is currently in the construction stage. Welcome colleagues in the industry to join the group and work with us to promote the construction of seata-go! If you want to contribute code to seata-go, you can refer to the  [**code contribution Specification**](./CONTRIBUTING_CN.md)  document to understand the specifications of the community, or you can join our community DingTalk group: 33069364 and communicate together!

## Licence

Seata-go uses Apache license version 2.0. Please refer to the license file for more information.
