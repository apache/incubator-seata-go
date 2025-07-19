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
<div style="align: center">
<img src="https://img.alicdn.com/imgextra/i1/O1CN011z0JfQ2723QgDiWuH_!!6000000007738-2-tps-1497-401.png"  height="100" width="426"/>
</div>

# Seata-go: Simple Extensible Autonomous Transaction Architecture(Go version)

[![CI](https://github.com/apache/incubator-seata-go/actions/workflows/build.yml/badge.svg)](https://github.com/apache/incubator-seata-go/actions/workflows/build.yml)  [![license](https://img.shields.io/github/license/apache/incubator-seata-go.svg)](https://www.apache.org/licenses/LICENSE-2.0.html) [![codecov](https://codecov.io/gh/apache/incubator-seata-go/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/incubator-seata-go) [简体中文 ZH](./README_ZH.md)

## What is Seata-go?

Apache Seata(incubating) is a very mature distributed transaction framework, and is the de facto standard platform for distributed transaction technology in the Java field. Seata-go is the implementation version of go language in Seata multilingual ecosystem, which realizes the interoperability between Java and Go, so that Go developers can also use seata-go to realize distributed transactions. Please visit the [official website of Seata](https://seata.apache.org/) to view the quick start and documentation.

The principle of seata-go is consistent with that of Seata-java, which is composed of TM, RM and TC. The functions of TC are reused in Java, and the functions of TM and RM have been connected with Seata-java.
### Distributed Transaction Problem in Microservices

Let's imagine a traditional monolithic application. Its business is built up with 3 modules. They use a single local data source.

Naturally, data consistency will be guaranteed by the local transaction.

![Monolithic App](https://img.alicdn.com/imgextra/i3/O1CN01FTtjyG1H4vvVh1sNY_!!6000000000705-0-tps-1106-678.jpg)

Things have changed in a microservices architecture. The 3 modules mentioned above are designed to be 3 services on top of 3 different data sources ([Pattern: Database per service](http://microservices.io/patterns/data/database-per-service.html)). Data consistency within every single service is naturally guaranteed by the local transaction.

**But how about the whole business logic scope?**

![Microservices Problem](https://img.alicdn.com/imgextra/i1/O1CN01DXkc3o1te9mnJcHOr_!!6000000005926-0-tps-1268-804.jpg)

### How Seata-go do?

Seata-go is just a solution to the problem mentioned above.

![Seata solution](https://img.alicdn.com/imgextra/i1/O1CN01FheliH1k5VHIRob3p_!!6000000004632-0-tps-1534-908.jpg)

Firstly, how to define a **Distributed Transaction**?

We say, a **Distributed Transaction** is a **Global Transaction** which is made up with a batch of **Branch Transaction**, and normally **Branch Transaction** is just **Local Transaction**.

![Global & Branch](https://cdn.nlark.com/lark/0/2018/png/18862/1545015454979-a18e16f6-ed41-44f1-9c7a-bd82c4d5ff99.png)

There are three roles in Seata-go:

- **Transaction Coordinator(TC):** Maintain status of global and branch transactions, drive the global commit or rollback.
- **Transaction Manager(TM):** Define the scope of global transaction: begin a global transaction, commit or rollback a global transaction.
- **Resource Manager(RM):** Manage resources that branch transactions working on, talk to TC for registering branch transactions and reporting status of branch transactions, and drive the branch transaction commit or rollback.

![Model](https://cdn.nlark.com/lark/0/2018/png/18862/1545013915286-4a90f0df-5fda-41e1-91e0-2aa3d331c035.png)

A typical lifecycle of Seata-go managed distributed transaction:

1. TM asks TC to begin a new global transaction. TC generates an XID representing the global transaction.
2. XID is propagated through microservices' invoke chain.
3. RM registers local transaction as a branch of the corresponding global transaction of XID to TC.
4. TM asks TC for committing or rollbacking the corresponding global transaction of XID.
5. TC drives all branch transactions under the corresponding global transaction of XID to finish branch committing or rollbacking.

![Typical Process](https://cdn.nlark.com/lark/0/2018/png/18862/1545296917881-26fabeb9-71fa-4f3e-8a7a-fc317d3389f4.png)

For more details about principle and design, please go to [Seata wiki page](https://github.com/apache/incubator-seata/wiki).

### History

##### Alibaba

- **TXC**: Taobao Transaction Constructor. Alibaba middleware team started this project since 2014 to meet the distributed transaction problems caused by application architecture change from monolithic to microservices.
- **GTS**: Global Transaction Service. TXC as an Aliyun middleware product with new name GTS was published since 2016.
- **Fescar**: we started the open source project Fescar based on TXC/GTS since 2019 to work closely with the community in the future.


##### Ant Financial

- **XTS**: Extended Transaction Service. Ant Financial middleware team developed the distributed transaction middleware since 2007, which is widely used in Ant Financial and solves the problems of data consistency across databases and services.

- **DTX**: Distributed Transaction Extended. Since 2013, XTS has been published on the Ant Financial Cloud, with the name of DTX .


##### Seata Community

- **Seata** :Simple Extensible Autonomous Transaction Architecture. Ant Financial joins Fescar, which make it to be a more neutral and open community for distributed transaction, and Fescar be renamed to Seata.



## How to run？

```go
go get seata.apache.org/seata-go@v2.0.0

```

if you want to know how to use and integrate seata-go, please refer to [apache/seata-go-samples](https://github.com/apache/incubator-seata-go-samples)

## How to find the latest version
Visit Seata-Go's GitHub Releases page

Open:
https://github.com/seata/seata-go/releases

The latest tag / release is the latest stable version.


## Documentation


You can view the full documentation from Seata Official Website: [Seata Website page](https://seata.apache.org/zh-cn/docs/overview/what-is-seata).

## Reporting bugs

Please follow the [template](.github/ISSUE_TEMPLATE/BUG_REPORT_TEMPLATE.md) for reporting any issues.

## Security

Please do not use our public issue tracker but refer to our [security policy](https://github.com/apache/incubator-seata/blob/2.x/SECURITY.md)

## Contributing

Seata-go is currently in the construction stage. Welcome colleagues in the industry to join the group and work with us to promote the construction of seata-go! If you want to contribute code to seata-go, you can refer to the  [**code contribution Specification**](./CONTRIBUTING_CN.md)  document to understand the specifications of the community, or you can join our community DingTalk group: 33069364 and communicate together!


## Contact

* Mailing list:
  * dev@seata.apache.org , for dev/user discussion. [subscribe](mailto:dev-subscribe@seata.apache.org), [unsubscribe](mailto:dev-unsubscribe@seata.apache.org), [archive](https://lists.apache.org/list.html?dev@seata.apache.org)
* Online chat:

|                                                       Dingtalk group                                                        |                                                   Wechat official account                                                    |                                                       QQ group                                                        |                                                  Wechat assistant                                                   |
|:---------------------------------------------------------------------------------------------------------------------------:|:----------------------------------------------------------------------------------------------------------------------------:|:---------------------------------------------------------------------------------------------------------------------:|:-------------------------------------------------------------------------------------------------------------------:|
| <img src="https://seata.apache.org/zh-cn/assets/images/dingtalk-group-67f42c9466fb2268b6927bb16b549d6c.jpg"  width="150" /> | <img src="https://seata.apache.org/zh-cn/assets/images/wechat-official-467d10305f5449e6b2096e65d23a9d02.jpg"  width="150" /> | <img src="https://seata.apache.org/zh-cn/assets/images/qq-group-8d8a89699cdb9ba8818364069475ba96.jpg"  width="150" /> | <img src="https://seata.apache.org/zh-cn/assets/images/wechat-f8a87a96973942b826e32d1aed9bc8d9.jpg"  width="150" /> |

## Seata ecosystem

* [Seata Website](https://github.com/apache/incubator-seata.github.io) - Seata official website
* [Seata](https://github.com/apache/incubator-seata)- Seata client and server
* [Seata GoLang](https://github.com/apache/incubator-seata-go) - Seata GoLang client and server
* [Seata Samples](https://github.com/apache/incubator-seata-samples) - Samples for Seata
* [Seata GoLang Samples](https://github.com/apache/incubator-seata-go-samples) - Samples for Seata GoLang
* [Seata K8s](https://github.com/apache/incubator-seata-k8s) - Seata integration with k8s
* [Seata CLI](https://github.com/apache/incubator-seata-ctl) - CLI tool for Seata

## Contributors

This project exists thanks to all the people who contribute. [[Contributors](https://github.com/apache/incubator-seata-go/graphs/contributors)].

## License

Seata-go is under the Apache 2.0 license. See the [LICENSE](https://github.com/apache/incubator-seata-go/blob/master/LICENSE) file for details.
