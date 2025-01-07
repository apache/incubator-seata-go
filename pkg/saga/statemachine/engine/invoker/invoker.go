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
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"reflect"
	"sync"
)

type ScriptInvokerManager interface {
}

type ScriptInvoker interface {
}

type ServiceInvokerManager interface {
	ServiceInvoker(serviceType string) ServiceInvoker
	PutServiceInvoker(serviceType string, invoker ServiceInvoker)
}

type ServiceInvoker interface {
	Invoke(ctx context.Context, input []any, service state.ServiceTaskState) (output []reflect.Value, err error)
	Close(ctx context.Context) error
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
