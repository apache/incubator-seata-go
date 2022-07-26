
# Seata-go: Simple Extensible Autonomous Transaction Architecture(Go version)

[![Build Status](https://github.com/seata/seata/workflows/build/badge.svg?branch=develop)](https://github.com/seata/seata/actions)
[![license](https://img.shields.io/github/license/seata/seata.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

[English 🇺🇸](./README.md)

## What is seata-go?

Seata is a very mature distributed transaction framework, and is the de facto standard platform for distributed transaction technology in the Java field. Seata-go is the implementation version of go language in Seata multilingual ecosystem, which realizes the interoperability between Java and Go, so that Go developers can also use seata-go to realize distributed transactions. Please visit the [official website of Seata](https://seata.io/en-us) to view the quick start and documentation.

The principle of seata-go is consistent with that of Seata-java, which is composed of TM, RM and TC. The functions of TC reuse Java, and the functions of TM and RM will be aligned with Seata-java later. The overall process is as follows:

![](https://user-images.githubusercontent.com/68344696/145942191-7a2d469f-94c8-4cd2-8c7e-46ad75683636.png)

## TODO list

- [x] TCC
- [ ] XA
- [ ] AT
- [ ] SAGA
- [ ] TM
- [x] RPC communication
- [ ] Transaction anti suspension
- [ ] Null compensation
- [ ] Configuration center
- [ ] Registration Center
- [ ] Metric monitoring
- [x] Examples


## How to run？

1. First download [**seata java**](https://github.com/seata/seata/tree/v1.5.2) and  Start the TC service. For the specific process, refer to  [**seata deployment guide**](https://seata.io/zh-cn/docs/ops/deploy-guide-beginner.ht ) Documentation
2. Just execute the main function under samples/ in the root directory


## How to join us？

Seata-go is currently in the construction stage. Welcome colleagues in the industry to join the group and work with us to promote the construction of seata-go! If you want to contribute code to seata-go, you can refer to the  [**code contribution Specification**](./CONTRIBUTING_CN.md)  document to understand the specifications of the community, or you can join our community DingTalk group: 33069364 and communicate together!


## Licence

Seata-go uses Apache license version 2.0. Please refer to the license file for more information.
