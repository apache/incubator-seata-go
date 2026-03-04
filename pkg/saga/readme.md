<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

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

下面简单说下engine中各个包的作用：
events：saga的是基于事件处理的，其中是event、eventBus的实现
expr：表达式声明、解析、执行
invoker：声明了serviceInvoker、scriptInvoker等接口、task调用管理、执行都在这个包中，例如httpInvoker
process_ctrl：状态机处理流程：上下文、执行、事件流转
sequence：分布式id
store：状态机存储接口、实现
status_decision：状态机状态决策



