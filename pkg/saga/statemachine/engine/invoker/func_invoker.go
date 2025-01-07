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
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/seata/seata-go/pkg/util/log"
)

type FuncInvoker struct {
	ServicesMapLock sync.Mutex
	servicesMap     map[string]FuncService
}

func NewFuncInvoker() *FuncInvoker {
	return &FuncInvoker{
		servicesMap: make(map[string]FuncService),
	}
}

func (f *FuncInvoker) RegisterService(serviceName string, service FuncService) {
	f.ServicesMapLock.Lock()
	defer f.ServicesMapLock.Unlock()
	f.servicesMap[serviceName] = service
}

func (f *FuncInvoker) GetService(serviceName string) FuncService {
	f.ServicesMapLock.Lock()
	defer f.ServicesMapLock.Unlock()
	return f.servicesMap[serviceName]
}

func (f *FuncInvoker) Invoke(ctx context.Context, input []any, service state.ServiceTaskState) (output []reflect.Value, err error) {
	serviceTaskStateImpl := service.(*state.ServiceTaskStateImpl)
	FuncService := f.GetService(serviceTaskStateImpl.ServiceName())
	if FuncService == nil {
		return nil, errors.New("no func service " + serviceTaskStateImpl.ServiceName() + " for service task state")
	}

	if serviceTaskStateImpl.IsAsync() {
		go func() {
			_, err := FuncService.CallMethod(serviceTaskStateImpl, input)
			if err != nil {
				log.Errorf("invoke Service[%s].%s failed, err is %s", serviceTaskStateImpl.ServiceName(), serviceTaskStateImpl.ServiceMethod(), err.Error())
			}
		}()
		return nil, nil
	}

	return FuncService.CallMethod(serviceTaskStateImpl, input)
}

func (f *FuncInvoker) Close(ctx context.Context) error {
	return nil
}

type FuncService interface {
	CallMethod(ServiceTaskStateImpl *state.ServiceTaskStateImpl, input []any) ([]reflect.Value, error)
}

type FuncServiceImpl struct {
	serviceName string
	methodLock  sync.Mutex
	method      any
}

func NewFuncService(serviceName string, method any) *FuncServiceImpl {
	return &FuncServiceImpl{
		serviceName: serviceName,
		method:      method,
	}
}

func (f *FuncServiceImpl) getMethod(serviceTaskStateImpl *state.ServiceTaskStateImpl) (*reflect.Value, error) {
	method := serviceTaskStateImpl.Method()
	if method == nil {
		return f.initMethod(serviceTaskStateImpl)
	}
	return method, nil
}

func (f *FuncServiceImpl) prepareArguments(input []any) []reflect.Value {
	args := make([]reflect.Value, len(input))
	for i, arg := range input {
		args[i] = reflect.ValueOf(arg)
	}
	return args
}

func (f *FuncServiceImpl) CallMethod(serviceTaskStateImpl *state.ServiceTaskStateImpl, input []any) ([]reflect.Value, error) {
	method, err := f.getMethod(serviceTaskStateImpl)
	if err != nil {
		return nil, err
	}

	args := f.prepareArguments(input)

	retryCountMap := make(map[state.Retry]int)
	for {
		res, err, shouldRetry := f.invokeMethod(method, args, serviceTaskStateImpl, retryCountMap)

		if !shouldRetry {
			if err != nil {
				return nil, errors.New("invoke service[" + serviceTaskStateImpl.ServiceName() + "]." + serviceTaskStateImpl.ServiceMethod() + " failed, err is " + err.Error())
			}
			return res, nil
		}
	}
}

func (f *FuncServiceImpl) initMethod(serviceTaskStateImpl *state.ServiceTaskStateImpl) (*reflect.Value, error) {
	methodName := serviceTaskStateImpl.ServiceMethod()
	f.methodLock.Lock()
	defer f.methodLock.Unlock()
	methodValue := reflect.ValueOf(f.method)
	if methodValue.IsZero() {
		return nil, errors.New("invalid method when func call, serviceName: " + f.serviceName)
	}

	if methodValue.Kind() == reflect.Func {
		serviceTaskStateImpl.SetMethod(&methodValue)
		return &methodValue, nil
	}

	method := methodValue.MethodByName(methodName)
	if method.IsZero() {
		return nil, errors.New("invalid method name when func call, serviceName: " + f.serviceName + ", methodName: " + methodName)
	}
	serviceTaskStateImpl.SetMethod(&method)
	return &method, nil
}

func (f *FuncServiceImpl) invokeMethod(method *reflect.Value, args []reflect.Value, serviceTaskStateImpl *state.ServiceTaskStateImpl, retryCountMap map[state.Retry]int) ([]reflect.Value, error, bool) {
	var res []reflect.Value
	var resErr error
	var shouldRetry bool

	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("%v", r)
			retry := f.matchRetry(serviceTaskStateImpl, errStr)
			resErr = errors.New(errStr)
			if retry != nil {
				shouldRetry = f.needRetry(serviceTaskStateImpl, retryCountMap, retry, resErr)
			}
		}
	}()

	outs := method.Call(args)
	if err, ok := outs[len(outs)-1].Interface().(error); ok {
		resErr = err
		errStr := err.Error()
		retry := f.matchRetry(serviceTaskStateImpl, errStr)
		if retry != nil {
			shouldRetry = f.needRetry(serviceTaskStateImpl, retryCountMap, retry, resErr)
		}
		return nil, resErr, shouldRetry
	}

	res = outs
	return res, nil, false
}

func (f *FuncServiceImpl) matchRetry(impl *state.ServiceTaskStateImpl, str string) state.Retry {
	if impl.Retry() != nil {
		for _, retry := range impl.Retry() {
			if retry.Exceptions() != nil {
				for _, exception := range retry.Exceptions() {
					if strings.Contains(str, exception) {
						return retry
					}
				}
			}
		}
	}
	return nil
}

func (f *FuncServiceImpl) needRetry(impl *state.ServiceTaskStateImpl, countMap map[state.Retry]int, retry state.Retry, err error) bool {
	attempt, exist := countMap[retry]
	if !exist {
		countMap[retry] = 0
	}

	if attempt >= retry.MaxAttempt() {
		return false
	}

	interval := retry.IntervalSecond()
	backoffRate := retry.BackoffRate()
	curInterval := int64(interval * 1000)
	if attempt != 0 {
		curInterval = int64(interval * backoffRate * float64(attempt) * 1000)
	}

	log.Warnf("invoke service[%s.%s] failed, will retry after %s millis, current retry count: %s, current err: %s",
		impl.ServiceName(), impl.ServiceMethod(), curInterval, attempt, err)

	time.Sleep(time.Duration(curInterval) * time.Millisecond)
	countMap[retry] = attempt + 1
	return true
}
