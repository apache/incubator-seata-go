/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package invoker

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang/state"
)

type ServiceInvokerManager interface {
	ServiceInvoker(serviceType string) ServiceInvoker
	PutServiceInvoker(serviceType string, invoker ServiceInvoker)
}

type ServiceInvoker interface {
	Invoke(ctx context.Context, input []any, service state.ServiceTaskState) (output []reflect.Value, err error)
	Close(ctx context.Context) error
}

// ScriptInvoker 脚本调用器接口
type ScriptInvoker interface {
	Type() string
	Invoke(ctx context.Context, script string, params map[string]interface{}) (interface{}, error)
	Close(ctx context.Context) error
}

// ScriptInvokerManager 脚本调用器管理器
type ScriptInvokerManager interface {
	GetInvoker(scriptType string) (ScriptInvoker, error)
	RegisterInvoker(invoker ScriptInvoker) error
}

type ScriptInvokerManagerImpl struct {
	invokers map[string]ScriptInvoker
	mutex    sync.RWMutex
}

func NewScriptInvokerManager() ScriptInvokerManager {
	return &ScriptInvokerManagerImpl{invokers: make(map[string]ScriptInvoker)}
}

func (manager *ScriptInvokerManagerImpl) GetInvoker(scriptType string) (ScriptInvoker, error) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	invoker := manager.invokers[scriptType]
	if invoker == nil {
		return nil, fmt.Errorf("script invoker %s not found", scriptType)
	}
	return invoker, nil
}

func (manager *ScriptInvokerManagerImpl) RegisterInvoker(invoker ScriptInvoker) error {
	if invoker == nil {
		return fmt.Errorf("script invoker is nil")
	}
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.invokers[invoker.Type()] = invoker
	return nil
}

type ServiceInvokerManagerImpl struct {
	invokers map[string]ServiceInvoker
	mutex    sync.Mutex
}

func NewServiceInvokerManagerImpl() *ServiceInvokerManagerImpl {
	return &ServiceInvokerManagerImpl{
		invokers: make(map[string]ServiceInvoker),
	}
}

func (manager *ServiceInvokerManagerImpl) ServiceInvoker(serviceType string) ServiceInvoker {
	return manager.invokers[serviceType]
}

func (manager *ServiceInvokerManagerImpl) PutServiceInvoker(serviceType string, invoker ServiceInvoker) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.invokers[serviceType] = invoker
}
