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
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type JsonParser interface {
	Unmarshal(data []byte, v any) error
	Marshal(v any) ([]byte, error)
}

type DefaultJsonParser struct{}

func (p *DefaultJsonParser) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (p *DefaultJsonParser) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

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

type LocalServiceInvoker struct {
	serviceRegistry map[string]interface{}
	methodCache     map[string]*reflect.Method
	jsonParser      JsonParser
	mutex           sync.RWMutex
}

func NewLocalServiceInvoker() *LocalServiceInvoker {
	return &LocalServiceInvoker{
		serviceRegistry: make(map[string]interface{}),
		methodCache:     make(map[string]*reflect.Method),
		jsonParser:      &DefaultJsonParser{},
	}
}

func (l *LocalServiceInvoker) RegisterService(serviceName string, instance interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.serviceRegistry[serviceName] = instance
}

func (l *LocalServiceInvoker) Invoke(ctx context.Context, input []any, service state.ServiceTaskState) ([]reflect.Value, error) {
	serviceName := service.ServiceName()
	instance, exists := l.serviceRegistry[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not registered", serviceName)
	}

	methodName := service.ServiceMethod()
	method, err := l.getMethod(serviceName, methodName, service.ParameterTypes())
	if err != nil {
		return nil, err
	}

	params, err := l.resolveParameters(input, method.Type)
	if err != nil {
		return nil, err
	}

	return l.invokeMethod(instance, method, params), nil
}

func (l *LocalServiceInvoker) resolveMethod(key, serviceName, methodName string) (*reflect.Method, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if cachedMethod, ok := l.methodCache[key]; ok {
		return cachedMethod, nil
	}

	instance, exists := l.serviceRegistry[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	objType := reflect.TypeOf(instance)
	method, ok := objType.MethodByName(methodName)
	if !ok {
		return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
	}

	l.methodCache[key] = &method
	return &method, nil
}

func (l *LocalServiceInvoker) getMethod(serviceName, methodName string, paramTypes []string) (*reflect.Method, error) {
	key := fmt.Sprintf("%s.%s", serviceName, methodName)

	l.mutex.RLock()
	if method, ok := l.methodCache[key]; ok {
		l.mutex.RUnlock()
		return method, nil
	}
	l.mutex.RUnlock()

	return l.resolveMethod(key, serviceName, methodName)
}

func (l *LocalServiceInvoker) resolveParameters(input []any, methodType reflect.Type) ([]reflect.Value, error) {
	params := make([]reflect.Value, methodType.NumIn())
	for i := 0; i < methodType.NumIn(); i++ {
		paramType := methodType.In(i)
		if i >= len(input) {
			params[i] = reflect.Zero(paramType)
			continue
		}

		converted, err := l.convertParam(input[i], paramType)
		if err != nil {
			return nil, err
		}
		params[i] = reflect.ValueOf(converted)
	}
	return params, nil
}

func (l *LocalServiceInvoker) convertParam(value any, targetType reflect.Type) (any, error) {
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
		value = reflect.ValueOf(value).Interface()
	}

	if targetType.Kind() == reflect.Int && reflect.TypeOf(value).Kind() == reflect.Float64 {
		return int(value.(float64)), nil
	} else if targetType == reflect.TypeOf("") && reflect.TypeOf(value).Kind() == reflect.Int {
		return fmt.Sprintf("%d", value), nil
	}

	if targetType.Kind() == reflect.Struct {
		jsonData, err := l.jsonParser.Marshal(value)
		if err != nil {
			return nil, err
		}
		instance := reflect.New(targetType).Interface()
		if err := l.jsonParser.Unmarshal(jsonData, instance); err != nil {
			return nil, err
		}
		return instance, nil
	}

	return value, nil
}

func (l *LocalServiceInvoker) invokeMethod(instance interface{}, method *reflect.Method, params []reflect.Value) []reflect.Value {
	instanceValue := reflect.ValueOf(instance)
	if method.Func.IsValid() {
		allParams := append([]reflect.Value{instanceValue}, params...)
		return method.Func.Call(allParams)
	}
	return nil
}

func (l *LocalServiceInvoker) Close(ctx context.Context) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.serviceRegistry = nil
	l.methodCache = nil
	return nil
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
