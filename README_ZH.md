
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

<img src="https://img.alicdn.com/imgextra/i1/O1CN011z0JfQ2723QgDiWuH_!!6000000007738-2-tps-1497-401.png" height="100" width="426"/>

</div>
# Seata-go：简单可扩展的自主事务架构（Go 语言版本）

[![CI](https://github.com/apache/incubator-seata-go/actions/workflows/build.yml/badge.svg)](https://github.com/apache/incubator-seata-go/actions/workflows/build.yml)  [![license](https://img.shields.io/github/license/apache/incubator-seata-go.svg)](https://www.apache.org/licenses/LICENSE-2.0.html) [![codecov](https://codecov.io/gh/apache/incubator-seata-go/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/incubator-seata-go) [English](./README.md)


## 什么是 Seata-go？


Apache Seata（incubating）是一个非常成熟的分布式事务框架，是 Java 领域事实上的标准分布式事务平台。Seata-go 是其在多语言生态中 Go 语言的实现版本，实现了 Java 与 Go 之间的互操作，使得 Go 开发者也可以使用 seata-go 实现分布式事务。请访问 [Seata 官网](https://seata.apache.org/) 获取快速入门和文档。

Seata-go 的原理与 Seata-java 一致，由 TM、RM 和 TC 三部分组成。其中 TC 功能复用 Java 的实现，而 TM 和 RM 则已与 Seata-java 对接。

### 微服务中的分布式事务问题

假设我们有一个传统的单体应用，其业务由三个模块组成，使用一个本地数据源，自然由本地事务保障数据一致性。
![Monolithic App](https://img.alicdn.com/imgextra/i3/O1CN01FTtjyG1H4vvVh1sNY_!!6000000000705-0-tps-1106-678.jpg)
但在微服务架构中，这三个模块被设计成了三个不同的服务，分别使用不同的数据源（[数据库每服务模式](http://microservices.io/patterns/data/database-per-service.html)）。单个服务内的数据一致性可以通过本地事务保障。

但**整个业务范围的一致性如何保障？**
![Microservices Problem](https://img.alicdn.com/imgextra/i1/O1CN01DXkc3o1te9mnJcHOr_!!6000000005926-0-tps-1268-804.jpg)


### Seata-go 如何做？

Seata-go 就是为了解决上述问题而生。
![Seata solution](https://img.alicdn.com/imgextra/i1/O1CN01FheliH1k5VHIRob3p_!!6000000004632-0-tps-1534-908.jpg)
首先，如何定义一个**分布式事务**？
我们认为，分布式事务是一个**全局事务**，由一组**分支事务**组成，而**分支事务**通常就是**本地事务**。
![Global & Branch](https://cdn.nlark.com/lark/0/2018/png/18862/1545015454979-a18e16f6-ed41-44f1-9c7a-bd82c4d5ff99.png)


Seata-go 中有三个角色：

- **事务协调器（TC）**：维护全局和分支事务的状态，驱动全局提交或回滚。

- **事务管理器（TM）**：定义全局事务的范围，开始、提交或回滚全局事务。

- **资源管理器（RM）**：管理分支事务所处理的资源，与 TC 通信注册分支事务并报告状态，驱动分支事务提交或回滚。
  ![Model](https://cdn.nlark.com/lark/0/2018/png/18862/1545013915286-4a90f0df-5fda-41e1-91e0-2aa3d331c035.png)


Seata-go 分布式事务的典型生命周期如下：



1. TM 请求 TC 开启一个新的全局事务，TC 生成表示全局事务的 XID。

2. XID 在微服务调用链中传播。

3. RM 将本地事务注册为该 XID 对应的全局事务的分支事务。

4. TM 请求 TC 提交或回滚该 XID 对应的全局事务。

5. TC 驱动该 XID 对应的所有分支事务进行提交或回滚。

![Typical Process](https://cdn.nlark.com/lark/0/2018/png/18862/1545296917881-26fabeb9-71fa-4f3e-8a7a-fc317d3389f4.png)

更多原理和设计细节，请参阅 [Seata Wiki 页面](https://github.com/apache/incubator-seata/wiki)。



### 历史背景

##### 阿里巴巴

- **TXC**：淘宝事务构建器。2014 年阿里中间件团队启动，用于应对从单体架构转向微服务带来的分布式事务问题。

- **GTS**：全局事务服务。2016 年 TXC 在阿里云上线，更名为 GTS。

- **Fescar**：2019 年开始基于 TXC/GTS 启动开源项目 Fescar，与社区合作。

##### 蚂蚁金服

- **XTS**：扩展事务服务。自 2007 年开始开发的分布式事务中间件，被广泛应用于蚂蚁金服，解决跨库跨服务数据一致性问题。

- **DTX**：分布式事务扩展。2013 年起在蚂蚁金服云发布，命名为 DTX。
##### Seata 社区

- **Seata**：简单可扩展的自主事务架构。蚂蚁金服加入 Fescar，使其成为一个更加中立开放的社区，Fescar 更名为 Seata。

## 如何运行？
```go

go get seata.apache.org/seata-go@2.0.0 

```
如果你想了解如何使用和集成 seata-go，请参考 [apache/seata-go-samples](https://github.com/apache/incubator-seata-go-samples)
## 如何查找最新版本
访问 Seata-Go 的 GitHub Releases 页面

打开：https://github.com/seata/seata-go/releases

最新的 tag / release 就是目前的最新稳定版本。
## 文档

你可以访问 Seata 官方网站获取完整文档：[Seata 官网](https://seata.apache.org/zh-cn/docs/overview/what-is-seata)
## 问题报告

若有问题，请遵循 [模板](./.github/ISSUE_TEMPLATE/BUG_REPORT_TEMPLATE.md) 报告问题。

## 安全

请勿使用公开的问题跟踪器，详情请参阅我们的 [安全政策](https://github.com/apache/incubator-seata/blob/2.x/SECURITY.md)

## 参与贡献
Seata-go 当前处于建设阶段，欢迎业界同仁加入我们，共同推进 Seata-go 建设！如果你想为 Seata-go 贡献代码，请参考 [代码贡献规范](./CONTRIBUTING_CN.md)，也可以加入我们的社区钉钉群：33069364 一起交流！

## 联系方式

* 邮件列表：  

  * dev@seata.apache.org - 用于开发/用户讨论   [订阅](mailto:dev-subscribe@seata.apache.org)、[取消订阅](mailto:dev-unsubscribe@seata.apache.org)、[归档](https://lists.apache.org/list.html?dev@seata.apache.org)

* 在线交流：  

|                                                             钉钉                                                              |                                                            微信公众号                                                             |                                                          QQ                                                           |                                                        微信助手                                                         |
|:---------------------------------------------------------------------------------------------------------------------------:|:----------------------------------------------------------------------------------------------------------------------------:|:---------------------------------------------------------------------------------------------------------------------:|:-------------------------------------------------------------------------------------------------------------------:|
| <img src="https://seata.apache.org/zh-cn/assets/images/dingtalk-group-67f42c9466fb2268b6927bb16b549d6c.jpg"  width="150" /> | <img src="https://seata.apache.org/zh-cn/assets/images/wechat-official-467d10305f5449e6b2096e65d23a9d02.jpg"  width="150" /> | <img src="https://seata.apache.org/zh-cn/assets/images/qq-group-8d8a89699cdb9ba8818364069475ba96.jpg"  width="150" /> | <img src="https://seata.apache.org/zh-cn/assets/images/wechat-f8a87a96973942b826e32d1aed9bc8d9.jpg"  width="150" /> |


## Seata 生态系统

* [Seata 官网](https://github.com/apache/incubator-seata.github.io)- Seata官方网站

* [Seata](https://github.com/apache/incubator-seata) - Seata 客户端和服务端

* [Seata GoLang](https://github.com/apache/incubator-seata-go) - Seata Go 客户端和服务端

* [Seata 示例](https://github.com/apache/incubator-seata-samples)- Seata 示例

* [Seata Go 示例](https://github.com/apache/incubator-seata-go-samples)- Seata GoLang 示例

* [Seata K8s 集成](https://github.com/apache/incubator-seata-k8s)- Seata 与 K8S 集成

* [Seata 命令行工具](https://github.com/apache/incubator-seata-ctl)- Seata CLI 工具

## 贡献者

Seata-go感谢所有贡献者的付出。[[贡献者列表](https://github.com/apache/incubator-seata-go/graphs/contributors)]
## 许可证

Seata-go 使用 Apache 2.0 协议，详情请查看 [LICENSE 文件](https://github.com/apache/incubator-seata-go/blob/master/LICENSE)
