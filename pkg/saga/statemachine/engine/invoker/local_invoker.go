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
	argTotal := methodType.NumIn()
	if argTotal == 0 {
		return nil, nil
	}

	argCount := argTotal - 1 // skip receiver
	if argCount <= 0 {
		return nil, nil
	}

	params := make([]reflect.Value, argCount)
	for i := 0; i < argCount; i++ {
		paramType := methodType.In(i + 1)
		if i >= len(input) {
			params[i] = reflect.Zero(paramType)
			continue
		}

		converted, err := l.convertParam(input[i], paramType)
		if err != nil {
			return nil, fmt.Errorf("parameter %d conversion error: %w", i, err)
		}

		val := reflect.ValueOf(converted)
		if !val.IsValid() {
			params[i] = reflect.Zero(paramType)
			continue
		}
		if val.Type().AssignableTo(paramType) {
			params[i] = val
			continue
		}
		if val.Type().ConvertibleTo(paramType) {
			params[i] = val.Convert(paramType)
			continue
		}
		params[i] = reflect.Zero(paramType)
	}

	return params, nil
}

func (l *LocalServiceInvoker) convertParam(value any, targetType reflect.Type) (any, error) {
	if targetType.Kind() == reflect.Ptr {
		elemType := targetType.Elem()
		instance := reflect.New(elemType).Interface()
		jsonData, err := l.jsonParser.Marshal(value)
		if err != nil {
			return nil, err
		}
		if err := l.jsonParser.Unmarshal(jsonData, instance); err != nil {
			return nil, err
		}
		return instance, nil
	}

	if targetType.Kind() == reflect.Struct {
		instance := reflect.New(targetType).Interface()
		jsonData, err := l.jsonParser.Marshal(value)
		if err != nil {
			return nil, err
		}
		if err := l.jsonParser.Unmarshal(jsonData, instance); err != nil {
			return nil, err
		}
		return reflect.ValueOf(instance).Elem().Interface(), nil
	}

	if targetType.Kind() == reflect.Int && reflect.TypeOf(value).Kind() == reflect.Float64 {
		return int(value.(float64)), nil
	} else if targetType == reflect.TypeOf("") && reflect.TypeOf(value).Kind() == reflect.Int {
		return fmt.Sprintf("%d", value), nil
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
