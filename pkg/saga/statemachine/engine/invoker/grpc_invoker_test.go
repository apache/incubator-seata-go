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
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
	"time"

	pb "github.com/seata/seata-go/testdata/saga/engine/invoker/grpc"
)

type MockGPRCClientImpl struct {
	GPRCClientImpl
}

type mockClientImpl struct {
	invokeCount int
}

func (m *mockClientImpl) SayHelloRight(ctx context.Context, word string) (string, error) {
	m.invokeCount++
	fmt.Println("invoke right")
	return word, nil
}

func (m *mockClientImpl) SayHelloRightLater(ctx context.Context, word string, delay int) (string, error) {
	m.invokeCount++
	if delay == m.invokeCount {
		fmt.Println("invoke right")
		return word, nil
	}
	fmt.Println("invoke fail")
	return "", errors.New("invoke failed")
}

func TestGRPCInvokerInvokeSucceedWithOutRetry(t *testing.T) {
	ctx := context.Background()
	invoker := newGRPCServiceInvoker()
	values, err := invoker.Invoke(ctx, []any{"hello"}, newHelloServiceTaskState())
	if err != nil {
		t.Error(err)
		return
	}
	if values == nil || len(values) == 0 {
		t.Error("no value in values")
		return
	}
	if values[0].Interface().(string) != "hello" {
		t.Errorf("expect hello, but got %v", values[0].Interface())
	}
	if _, ok := values[1].Interface().(error); ok {
		t.Errorf("expect nil, but got %v", values[1].Interface())
	}
}

func TestGRPCInvokerInvokeSucceedInRetry(t *testing.T) {
	ctx := context.Background()
	invoker := newGRPCServiceInvoker()
	values, err := invoker.Invoke(ctx, []any{"hello", 2}, newHelloServiceTaskStateWithRetry())
	if err != nil {
		t.Error(err)
		return
	}
	if values == nil || len(values) == 0 {
		t.Error("no value in values")
		return
	}
	if values[0].Interface().(string) != "hello" {
		t.Errorf("expect hello, but got %v", values[0].Interface())
	}
	if _, ok := values[1].Interface().(error); ok {
		t.Errorf("expect nil, but got %v", values[1].Interface())
	}
}

func TestGRPCInvokerInvokeFailedInRetry(t *testing.T) {
	ctx := context.Background()
	invoker := newGRPCServiceInvoker()
	_, err := invoker.Invoke(ctx, []any{"hello", 5}, newHelloServiceTaskStateWithRetry())
	if err != nil {
		assert.Error(t, err)
	}
}

func TestGRPCInvokerInvokeE2E(t *testing.T) {
	go func() {
		pb.StartProductServer()
	}()
	time.Sleep(3000 * time.Millisecond)
	conn, err := grpc.Dial("localhost:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}
	c := pb.NewProductInfoClient(conn)
	grpcClient := NewGRPCClient("product", c, conn)

	invoker := NewGRPCInvoker()
	invoker.RegisterClient("product", grpcClient)
	ctx := context.Background()
	values, err := invoker.Invoke(ctx, []any{&pb.Product{Id: "123"}}, newProductServiceTaskState())
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(values)
	err = invoker.Close(ctx)
	if err != nil {
		t.Error(err)
		return
	}
}

func newGRPCServiceInvoker() ServiceInvoker {
	mockGRPCInvoker := NewGRPCInvoker()
	mockGRPCClient := &mockClientImpl{}
	mockClient := NewGRPCClient("hello", mockGRPCClient, &grpc.ClientConn{})
	mockGRPCInvoker.RegisterClient("hello", mockClient)
	return mockGRPCInvoker
}

func newProductServiceTaskState() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("product")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("product")
	serviceTaskStateImpl.SetServiceType("GRPC")
	serviceTaskStateImpl.SetServiceMethod("AddProduct")

	retryImpl := &state.RetryImpl{}
	retryImpl.SetExceptions([]string{"fail"})
	retryImpl.SetIntervalSecond(1)
	retryImpl.SetMaxAttempt(3)
	retryImpl.SetBackoffRate(0.9)
	serviceTaskStateImpl.SetRetry([]state.Retry{retryImpl})
	return serviceTaskStateImpl
}

func newHelloServiceTaskState() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("hello")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("hello")
	serviceTaskStateImpl.SetServiceType("GRPC")
	serviceTaskStateImpl.SetServiceMethod("SayHelloRight")
	return serviceTaskStateImpl
}

func newHelloServiceTaskStateWithRetry() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("hello")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("hello")
	serviceTaskStateImpl.SetServiceType("GRPC")
	serviceTaskStateImpl.SetServiceMethod("SayHelloRightLater")

	retryImpl := &state.RetryImpl{}
	retryImpl.SetExceptions([]string{"fail"})
	retryImpl.SetIntervalSecond(1)
	retryImpl.SetMaxAttempt(3)
	retryImpl.SetBackoffRate(0.9)
	serviceTaskStateImpl.SetRetry([]state.Retry{retryImpl})
	return serviceTaskStateImpl
}
