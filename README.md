# seata-golang

### 一个朴素的想法
作为一个刚入 Golang 坑的普通微服务开发者来讲，很容易产生一个朴素的想法，希望 Golang 微服务也有分布式事务解决方案。我们注意到阿里开源了 Java 版的分布式事务解决方案 Seata，本项目尝试将 Java 版的 Seata 改写一个 Golang 的版本。
在 Seata 没有 Golang 版本 client sdk 的情况下，Golang 版本的 TC Server 使用了和 Java 版 Seata 一样的通信协议，方便调试。
希望有同样朴素想法的开发者加入我们一起完善 Golang 版本的分布式事务解决方案。本方案参考了 [dubbo-go](#https://github.com/apache/dubbo-go) 的实现。由于时间有限，且对 golang 的一些特性不甚了解，有些实现不太优雅，希望有更多开发者来参与并优化它。

### todo list
- [X] Memory Session Manager
- [X] DB Session Manager (only support mysql) 
- [ ] RAFT Session Manager  
- [X] Metrics Collector
- [X] TM
- [X] RM TCC
- [X] RM AT
- [ ] Client merged request
- [ ] Read config from Config Center
- [ ] Unit Test

### 运行 TC

+ 编译
```
cd ${projectpath}/tc/app/cmd
go build
```

+ 将编译好的程序移动到示例代码目录

```
mv cmd ${targetpath}/
cd ${targetpath}
```

+ 启动 TC

```
./cmd start -config ${projectpath}/tc/app/profiles/dev/config.yml
```
