
# seata saga

未来计划有三种使用方式

- 基于状态机引擎的 json
   link: statemachine_engine#Start
- stream builder
    stateMachine.serviceTask().build().Start
- 二阶段方式saga，类似tcc使用

上面1、2是以来[statemachine](statemachine)，状态机引擎实现的，3相对比较独立。


状态机的实现在：saga-statemachine包中
其中[statelang](statemachine%2Fstatelang)是状态机语言的解析，目前实现的是json解析方式，状态机语言可以参考：
https://seata.io/docs/user/mode/saga

状态机json执行的入口类是：[statemachine_engine.go](statemachine%2Fengine%2Fstatemachine_engine.go)



