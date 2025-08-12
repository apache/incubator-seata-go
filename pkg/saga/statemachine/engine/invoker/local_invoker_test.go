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
	"testing"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type MockLocalService struct {
	invokeCount int
}

func (m *MockLocalService) GetServiceName() string {
	return "MockLocalService"
}

func (m *MockLocalService) Add(a, b int) int {
	m.invokeCount++
	return a + b
}

func (m *MockLocalService) Multiply(f float64, i int) float64 {
	m.invokeCount++
	return f * float64(i)
}

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (m *MockLocalService) GetUserName(user User) string {
	m.invokeCount++
	return user.Name
}

func (m *MockLocalService) ErrorMethod() error {
	return errors.New("expected error")
}

func TestLocalInvoker_ServiceNotRegistered(t *testing.T) {
	invoker := NewLocalServiceInvoker()
	ctx := context.Background()
	taskState := newLocalServiceTaskState("unregisteredService", "AnyMethod")

	_, err := invoker.Invoke(ctx, []any{}, taskState)
	if err == nil {
		t.Error("expected error when service not registered, but got nil")
	}
	if err.Error() != "service unregisteredService not registered" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLocalInvoker_MethodNotFound(t *testing.T) {
	invoker := NewLocalServiceInvoker()
	service := &MockLocalService{}
	invoker.RegisterService("mockService", service)

	ctx := context.Background()
	taskState := newLocalServiceTaskState("mockService", "NonExistentMethod")

	_, err := invoker.Invoke(ctx, []any{}, taskState)
	if err == nil {
		t.Error("expected error when method not found, but got nil")
	}
	if err.Error() != "method NonExistentMethod not found in service mockService" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLocalInvoker_InvokeSuccess(t *testing.T) {
	tests := []struct {
		name        string
		service     interface{}
		serviceName string
		methodName  string
		input       []any
		expected    interface{}
	}{
		{
			name:        "test basic method call",
			service:     &MockLocalService{},
			serviceName: "mockService",
			methodName:  "GetServiceName",
			input:       []any{},
			expected:    "MockLocalService",
		},
		{
			name:        "test method with parameters",
			service:     &MockLocalService{},
			serviceName: "mockService",
			methodName:  "Add",
			input:       []any{2, 3},
			expected:    5,
		},
		{
			name:        "test parameter type conversion",
			service:     &MockLocalService{},
			serviceName: "mockService",
			methodName:  "Multiply",
			input:       []any{2.5, 4},
			expected:    10.0,
		},
	}

	invoker := NewLocalServiceInvoker()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker.RegisterService(tt.serviceName, tt.service)
			taskState := newLocalServiceTaskState(tt.serviceName, tt.methodName)

			results, err := invoker.Invoke(ctx, tt.input, taskState)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("no results returned")
			}

			result := results[0].Interface()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLocalInvoker_StructParameterConversion(t *testing.T) {
	invoker := NewLocalServiceInvoker()
	service := &MockLocalService{}
	invoker.RegisterService("userService", service)

	ctx := context.Background()
	taskState := newLocalServiceTaskState("userService", "GetUserName")

	input := []any{map[string]interface{}{"name": "Alice", "age": 30}}
	results, err := invoker.Invoke(ctx, input, taskState)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("no results returned")
	}

	result := results[0].Interface()
	if result != "Alice" {
		t.Errorf("expected 'Alice', got %v", result)
	}
}

func TestLocalInvoker_MethodCaching(t *testing.T) {
	invoker := NewLocalServiceInvoker()
	service := &MockLocalService{}
	invoker.RegisterService("cacheTestService", service)

	ctx := context.Background()
	taskState := newLocalServiceTaskState("cacheTestService", "Add")

	_, err := invoker.Invoke(ctx, []any{1, 1}, taskState)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := invoker.Invoke(ctx, []any{2, 3}, taskState)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Interface() != 5 {
		t.Errorf("expected 5, got %v", results[0].Interface())
	}

	if service.invokeCount != 2 {
		t.Errorf("expected 2 invocations, got %d", service.invokeCount)
	}
}

func newLocalServiceTaskState(serviceName, methodName string) state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName(fmt.Sprintf("%s_%s", serviceName, methodName))
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName(serviceName)
	serviceTaskStateImpl.SetServiceType("local")
	serviceTaskStateImpl.SetServiceMethod(methodName)
	return serviceTaskStateImpl
}
