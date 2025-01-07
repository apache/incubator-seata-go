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
	"testing"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

// struct's method test
type mockFuncImpl struct {
	invokeCount int
}

func (m *mockFuncImpl) SayHelloRight(word string) (string, error) {
	m.invokeCount++
	fmt.Println("invoke right")
	return word, nil
}

func (m *mockFuncImpl) SayHelloRightLater(word string, delay int) (string, error) {
	m.invokeCount++
	if delay == m.invokeCount {
		fmt.Println("invoke right")
		return word, nil
	}
	fmt.Println("invoke fail")
	return "", errors.New("invoke failed")
}

func TestFuncInvokerInvokeSucceed(t *testing.T) {
	tests := []struct {
		name      string
		input     []any
		taskState state.ServiceTaskState
		expected  string
		expectErr bool
	}{
		{
			name:      "Invoke Struct Succeed",
			input:     []any{"hello"},
			taskState: newFuncHelloServiceTaskState(),
			expected:  "hello",
			expectErr: false,
		},
		{
			name:      "Invoke Struct In Retry",
			input:     []any{"hello", 2},
			taskState: newFuncHelloServiceTaskStateWithRetry(),
			expected:  "hello",
			expectErr: false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := newFuncServiceInvoker()
			values, err := invoker.Invoke(ctx, tt.input, tt.taskState)

			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if values == nil || len(values) == 0 {
				t.Fatal("no value in values")
			}

			if resultString, ok := values[0].Interface().(string); ok {
				if resultString != tt.expected {
					t.Errorf("expect %s, but got %s", tt.expected, resultString)
				}
			} else {
				t.Errorf("expected string, but got %v", values[0].Interface())
			}

			if resultError, ok := values[1].Interface().(error); ok {
				if resultError != nil {
					t.Errorf("expect nil, but got %s", resultError)
				}
			}
		})
	}
}

func TestFuncInvokerInvokeFailed(t *testing.T) {
	tests := []struct {
		name      string
		input     []any
		taskState state.ServiceTaskState
		expected  string
		expectErr bool
	}{
		{
			name:      "Invoke Struct Failed In Retry",
			input:     []any{"hello", 5},
			taskState: newFuncHelloServiceTaskStateWithRetry(),
			expected:  "",
			expectErr: true,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := newFuncServiceInvoker()
			_, err := invoker.Invoke(ctx, tt.input, tt.taskState)

			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func newFuncServiceInvoker() ServiceInvoker {
	mockFuncInvoker := NewFuncInvoker()
	mockFuncService := &mockFuncImpl{}
	mockService := NewFuncService("hello", mockFuncService)
	mockFuncInvoker.RegisterService("hello", mockService)
	return mockFuncInvoker
}

func newFuncHelloServiceTaskState() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("hello")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("hello")
	serviceTaskStateImpl.SetServiceType("func")
	serviceTaskStateImpl.SetServiceMethod("SayHelloRight")
	return serviceTaskStateImpl
}

func newFuncHelloServiceTaskStateWithRetry() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("hello")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("hello")
	serviceTaskStateImpl.SetServiceType("func")
	serviceTaskStateImpl.SetServiceMethod("SayHelloRightLater")

	retryImpl := &state.RetryImpl{}
	retryImpl.SetExceptions([]string{"fail"})
	retryImpl.SetIntervalSecond(1)
	retryImpl.SetMaxAttempt(3)
	retryImpl.SetBackoffRate(0.9)
	serviceTaskStateImpl.SetRetry([]state.Retry{retryImpl})
	return serviceTaskStateImpl
}
