
# seata-go: 简单的可扩展自主事务架构(Go版本)

[![Build Status](https://github.com/seata/seata/workflows/build/badge.svg?branch=develop)](https://github.com/seata/seata/actions)
[![license](https://img.shields.io/github/license/seata/seata.svg)](https://www.apache.org/licenses/LICENSE-2.0.html)

[English 🇺🇸](./README.md)

## 什么是 seata-go？

Seata是一个非常成熟的分布式事务框架，在Java领域是事实上的分布式事务技术标准平台。Seata-go 是 seata 多语言生态中的Go语言实现版本，实现了 Java 和 Go 之间的互通，让 Go 开发者也能使用 seata-go 来实现分布式事务。请访问[Seata 官网](https://seata.io/zh-cn/)查看快速开始和文档。

Seata-go 的原理和 Seata-java 保持一致，都是由 TM、RM 和 TC 组成，其中 TC 的功能复用 Java 的，TM和RM功能后面会和 Seata-java对齐，整体流程如下：

![](https://user-images.githubusercontent.com/68344696/145942191-7a2d469f-94c8-4cd2-8c7e-46ad75683636.png)

## 待办事项

- [x] TCC
- [ ] XA
- [ ] AT
- [ ] SAGA
- [ ] TM
- [x] RPC 通信
- [ ] 事务防悬挂
- [ ] 空补偿
- [ ] 配置中心
- [ ] 注册中心
- [ ] Metric监控
- [x] Sample例子


## 如何运行项目？

1. 首先下载 [**Seata Java**](https://github.com/seata/seata/tree/v1.5.2) 的源码，启动 TC 服务即可，具体流程参考 **[Seata部署指南](https://seata.io/zh-cn/docs/ops/deploy-guide-beginner.ht)**文档
2. 执行根目录下的 samples/ 下的 main 函数即可


## 如何给Seata-go贡献代码？

Seata-go 目前正在建设阶段，欢迎行业同仁入群参与其中，与我们一起推动 seata-go 的建设！如果你想给 seata-go 贡献代码，可以参考 **[代码贡献规范](./CONTRIBUTING.md)** 文档来了解社区的规范，也可以加入我们的社区钉钉群：33069364，一起沟通交流！

## 协议

Seata-go 使用 Apache 许可证2.0版本，请参阅 LICENSE 文件了解更多。