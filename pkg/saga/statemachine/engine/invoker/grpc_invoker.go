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
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/seata/seata-go/pkg/util/log"
	"google.golang.org/grpc"
	"reflect"
	"strings"
	"sync"
	"time"
)

type GRPCInvoker struct {
	clients        map[string]GRPCClient
	clientsMapLock sync.Mutex
	needClose      bool
}

func NewGRPCInvoker() *GRPCInvoker {
	return &GRPCInvoker{
		clients: make(map[string]GRPCClient),
	}
}

func (g *GRPCInvoker) NeedClose() bool {
	return g.needClose
}

func (g *GRPCInvoker) SetNeedClose(needClose bool) {
	g.needClose = needClose
}

func (g *GRPCInvoker) RegisterClient(serviceName string, client GRPCClient) {
	g.clientsMapLock.Lock()
	defer g.clientsMapLock.Unlock()

	g.clients[serviceName] = client
}

func (g *GRPCInvoker) GetClient(serviceName string) GRPCClient {
	g.clientsMapLock.Lock()
	defer g.clientsMapLock.Unlock()

	if client, ok := g.clients[serviceName]; ok {
		return client
	}

	return nil
}

func (g *GRPCInvoker) Invoke(ctx context.Context, input []any, service state.ServiceTaskState) (output []reflect.Value, err error) {
	serviceTaskStateImpl := service.(*state.ServiceTaskStateImpl)
	client := g.GetClient(serviceTaskStateImpl.ServiceName())
	if client == nil {
		return nil, errors.New(fmt.Sprintf("no grpc client %s for service task state", serviceTaskStateImpl.ServiceName()))
	}

	// context is the first arg in grpc client method
	input = append([]any{ctx}, input...)
	if serviceTaskStateImpl.IsAsync() {
		go func() {
			_, err := client.CallMethod(serviceTaskStateImpl, input)
			if err != nil {
				log.Errorf("invoke Service[%s].%s failed, err is %s", serviceTaskStateImpl.ServiceName(),
					serviceTaskStateImpl.ServiceMethod(), err)
			}
		}()
		return nil, nil
	} else {
		return client.CallMethod(serviceTaskStateImpl, input)
	}
}

func (g *GRPCInvoker) Close(ctx context.Context) error {
	if g.needClose {
		for _, client := range g.clients {
			err := client.CloseConnection()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type GRPCClient interface {
	CallMethod(serviceTaskStateImpl *state.ServiceTaskStateImpl, input []any) ([]reflect.Value, error)

	CloseConnection() error
}

type GPRCClientImpl struct {
	serviceName string
	client      any
	connection  *grpc.ClientConn
	methodLock  sync.Mutex
}

func NewGRPCClient(serviceName string, client any, connection *grpc.ClientConn) *GPRCClientImpl {
	return &GPRCClientImpl{
		serviceName: serviceName,
		client:      client,
		connection:  connection,
	}
}

func (g *GPRCClientImpl) CallMethod(serviceTaskStateImpl *state.ServiceTaskStateImpl, input []any) ([]reflect.Value, error) {

	if serviceTaskStateImpl.Method() == nil {
		err := g.initMethod(serviceTaskStateImpl)
		if err != nil {
			return nil, err
		}
	}
	method := serviceTaskStateImpl.Method()

	args := make([]reflect.Value, 0, len(input))
	for _, arg := range input {
		args = append(args, reflect.ValueOf(arg))
	}

	retryCountMap := make(map[state.Retry]int)
	for {
		res, err, shouldRetry := func() (res []reflect.Value, resErr error, shouldRetry bool) {
			defer func() {
				// err may happen in the method invoke (panic) and method return, we try to find err and use it to decide retry by
				// whether contains exception or not
				if r := recover(); r != nil {
					errStr := fmt.Sprintf("%v", r)
					retry := g.matchRetry(serviceTaskStateImpl, errStr)
					res = nil
					resErr = errors.New(errStr)

					if retry == nil {
						return
					}
					shouldRetry = g.needRetry(serviceTaskStateImpl, retryCountMap, retry, resErr)
					return
				}
			}()

			outs := method.Call(args)
			// err is the last arg in grpc client method
			if err, ok := outs[len(outs)-1].Interface().(error); ok {
				errStr := err.Error()
				retry := g.matchRetry(serviceTaskStateImpl, errStr)
				res = nil
				resErr = err

				if retry == nil {
					return
				}
				shouldRetry = g.needRetry(serviceTaskStateImpl, retryCountMap, retry, resErr)
				return
			}

			// invoke success
			res = outs
			resErr = nil
			shouldRetry = false
			return
		}()

		if !shouldRetry {
			if err != nil {
				return nil, errors.New(fmt.Sprintf("invoke Service[%s] failed, not satisfy retry config, the last err is %s",
					serviceTaskStateImpl.ServiceName(), err))
			}
			return res, nil
		}
	}
}

func (g *GPRCClientImpl) CloseConnection() error {
	if g.connection != nil {
		err := g.connection.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *GPRCClientImpl) initMethod(serviceTaskStateImpl *state.ServiceTaskStateImpl) error {
	methodName := serviceTaskStateImpl.ServiceMethod()
	g.methodLock.Lock()
	defer g.methodLock.Unlock()
	clientValue := reflect.ValueOf(g.client)
	if clientValue.IsZero() {
		return errors.New(fmt.Sprintf("invalid client value when grpc client call, serviceName: %s", g.serviceName))
	}
	method := clientValue.MethodByName(methodName)
	if method.IsZero() {
		return errors.New(fmt.Sprintf("invalid client method when grpc client call, serviceName: %s, serviceMethod: %s",
			g.serviceName, methodName))
	}
	serviceTaskStateImpl.SetMethod(&method)
	return nil
}

func (g *GPRCClientImpl) matchRetry(impl *state.ServiceTaskStateImpl, str string) state.Retry {
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

func (g *GPRCClientImpl) needRetry(impl *state.ServiceTaskStateImpl, countMap map[state.Retry]int, retry state.Retry, err error) bool {
	attempt, exist := countMap[retry]
	if !exist {
		countMap[retry] = 0
	}

	if attempt >= retry.MaxAttempt() {
		return false
	}

	intervalSecond := retry.IntervalSecond()
	backoffRate := retry.BackoffRate()
	var currentInterval int64
	if attempt == 0 {
		currentInterval = int64(intervalSecond * 1000)
	} else {
		currentInterval = int64(intervalSecond * backoffRate * float64(attempt) * 1000)
	}

	log.Warnf("invoke service[%s.%s] failed, will retry after %s millis, current retry count: %s, current err: %s",
		impl.ServiceName(), impl.ServiceMethod(), currentInterval, attempt, err)

	time.Sleep(time.Duration(currentInterval) * time.Millisecond)
	countMap[retry] = attempt + 1
	return true
}
