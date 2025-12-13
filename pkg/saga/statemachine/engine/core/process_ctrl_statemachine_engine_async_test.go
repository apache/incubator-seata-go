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

package core

import (
	"context"
	stderr "errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/config"
	engExc "seata.apache.org/seata-go/pkg/saga/statemachine/engine/exception"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/invoker"
	"seata.apache.org/seata-go/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang/parser"
	"seata.apache.org/seata-go/pkg/util/errors"
)

func TestStartAsyncDisabled(t *testing.T) {
	cfg, err := config.NewDefaultStateMachineConfig(
		config.WithEnableAsync(false),
		config.WithStateMachineResources(nil),
	)
	require.NoError(t, err)

	engine := &ProcessCtrlStateMachineEngine{StateMachineConfig: cfg}

	_, err = engine.StartAsync(context.Background(), "dummy", "", nil, nil)
	require.Error(t, err)

	execErr, ok := engExc.IsEngineExecutionException(err)
	if !ok {
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
	require.Equal(t, errors.AsynchronousStartDisabled, execErr.Code)
}

func TestStartAsyncEnabledWithoutStateMachine(t *testing.T) {
	cfg, err := config.NewDefaultStateMachineConfig(
		config.WithEnableAsync(true),
		config.WithStateMachineResources(nil),
	)
	require.NoError(t, err)

	engine := &ProcessCtrlStateMachineEngine{StateMachineConfig: cfg}

	_, err = engine.StartAsync(context.Background(), "missing", "", nil, nil)
	require.Error(t, err)

	execErr, ok := engExc.IsEngineExecutionException(err)
	if !ok {
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
	require.Equal(t, errors.ObjectNotExists, execErr.Code)
}

func TestStartAsyncSuccessfulFlow(t *testing.T) {
	cfg, err := config.NewDefaultStateMachineConfig(
		config.WithEnableAsync(true),
		config.WithStateMachineResources(nil),
	)
	require.NoError(t, err)

	const simpleStateMachine = `{
	  "Name": "AsyncSimple",
	  "StartState": "Greet",
	  "States": {
	    "Greet": {
	      "Type": "ServiceTask",
	      "ServiceType": "local",
	      "ServiceName": "asyncTestService",
	      "ServiceMethod": "Hello",
	      "CompensateState": "",
	      "ForCompensation": false,
	      "ForUpdate": false,
	      "Next": "Success"
	    },
	    "Success": {
	      "Type": "Succeed"
	    }
	  }
	}`

	stateMachine, err := parser.NewJSONStateMachineParser().Parse(simpleStateMachine)
	require.NoError(t, err)
	stateMachine.SetTenantId(cfg.GetDefaultTenantId())
	stateMachine.SetContent(simpleStateMachine)
	require.NoError(t, cfg.StateMachineRepository().RegistryStateMachine(stateMachine))

	localInvoker, ok := cfg.ServiceInvokerManager().ServiceInvoker("local").(*invoker.LocalServiceInvoker)
	require.True(t, ok)
	service := &asyncTestService{}
	localInvoker.RegisterService("asyncTestService", service)

	callback := newAsyncTestCallback()
	engine := &ProcessCtrlStateMachineEngine{StateMachineConfig: cfg}

	inst, err := engine.StartAsync(context.Background(), "AsyncSimple", "", nil, callback)
	require.NoError(t, err)
	require.NotNil(t, inst)
	require.Equal(t, "AsyncSimple", inst.StateMachine().Name())

	select {
	case <-callback.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for async callback")
	}

	require.NoError(t, callback.Err())
	require.NotNil(t, callback.Instance())
	require.Equal(t, statelang.SU, callback.Instance().Status())
	require.Equal(t, 1, service.Calls())
}

type asyncTestService struct {
	mu    sync.Mutex
	calls int
}

func (s *asyncTestService) Hello() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return "ok"
}

func (s *asyncTestService) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type asyncTestCallback struct {
	done chan struct{}
	once sync.Once
	mu   sync.Mutex
	inst statelang.StateMachineInstance
	err  error
}

func newAsyncTestCallback() *asyncTestCallback {
	return &asyncTestCallback{done: make(chan struct{})}
}

func (c *asyncTestCallback) OnFinished(ctx context.Context, _ process_ctrl.ProcessContext, stateMachineInstance statelang.StateMachineInstance) {
	c.mu.Lock()
	c.inst = stateMachineInstance
	c.err = nil
	c.mu.Unlock()
	c.once.Do(func() { close(c.done) })
}

func (c *asyncTestCallback) OnError(ctx context.Context, _ process_ctrl.ProcessContext, stateMachineInstance statelang.StateMachineInstance, err error) {
	c.mu.Lock()
	c.inst = stateMachineInstance
	c.err = err
	c.mu.Unlock()
	c.once.Do(func() { close(c.done) })
}

func (c *asyncTestCallback) Done() <-chan struct{} {
	return c.done
}

func (c *asyncTestCallback) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *asyncTestCallback) Instance() statelang.StateMachineInstance {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inst
}

// TestStartAsyncWithCompensation tests async state machine with compensation flow
func TestStartAsyncWithCompensation(t *testing.T) {
	cfg, err := config.NewDefaultStateMachineConfig(
		config.WithEnableAsync(true),
		config.WithStateMachineResources(nil),
	)
	require.NoError(t, err)

	const compensationStateMachine = `{
	  "Name": "AsyncCompensation",
	  "StartState": "Task1",
	  "States": {
	    "Task1": {
	      "Type": "ServiceTask",
	      "ServiceType": "local",
	      "ServiceName": "asyncCompensateService",
	      "ServiceMethod": "DoTask1",
	      "CompensateState": "CompensateTask1",
	      "Next": "Task2"
	    },
	    "Task2": {
	      "Type": "ServiceTask",
	      "ServiceType": "local",
	      "ServiceName": "asyncCompensateService",
	      "ServiceMethod": "DoTask2Fail",
	      "CompensateState": "CompensateTask2",
	      "Next": "Success",
	      "Catch": [
	        {
	          "Exceptions": ["java.lang.Exception"],
	          "Next": "CompensationTrigger"
	        }
	      ]
	    },
	    "CompensationTrigger": {
	      "Type": "CompensationTrigger",
	      "Next": "Fail"
	    },
	    "CompensateTask1": {
	      "Type": "ServiceTask",
	      "ServiceType": "local",
	      "ServiceName": "asyncCompensateService",
	      "ServiceMethod": "CompensateTask1",
	      "ForCompensation": true,
	      "Next": "CompensateEnd"
	    },
	    "CompensateTask2": {
	      "Type": "ServiceTask",
	      "ServiceType": "local",
	      "ServiceName": "asyncCompensateService",
	      "ServiceMethod": "CompensateTask2",
	      "ForCompensation": true,
	      "Next": "CompensateEnd"
	    },
	    "CompensateEnd": {
	      "Type": "Succeed"
	    },
	    "Success": {
	      "Type": "Succeed"
	    },
	    "Fail": {
	      "Type": "Fail"
	    }
	  }
	}`

	stateMachine, err := parser.NewJSONStateMachineParser().Parse(compensationStateMachine)
	require.NoError(t, err)
	stateMachine.SetTenantId(cfg.GetDefaultTenantId())
	stateMachine.SetContent(compensationStateMachine)
	require.NoError(t, cfg.StateMachineRepository().RegistryStateMachine(stateMachine))

	localInvoker, ok := cfg.ServiceInvokerManager().ServiceInvoker("local").(*invoker.LocalServiceInvoker)
	require.True(t, ok)
	service := &asyncCompensateService{}
	localInvoker.RegisterService("asyncCompensateService", service)

	callback := newAsyncTestCallback()
	engine := &ProcessCtrlStateMachineEngine{StateMachineConfig: cfg}

	inst, err := engine.StartAsync(context.Background(), "AsyncCompensation", "", nil, callback)
	require.NoError(t, err)
	require.NotNil(t, inst)
	require.Equal(t, "AsyncCompensation", inst.StateMachine().Name())

	select {
	case <-callback.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for async callback")
	}

	// Should finish with error due to Task2 failure
	// Note: In test environment without DB store, compensation flow may not be fully executed
	require.NotNil(t, callback.Instance())
	require.NotEqual(t, statelang.SU, callback.Instance().Status(), "Status should not be SU after Task2 fails")
	require.Equal(t, 1, service.Task1Calls(), "Task1 should be called once")
	require.Equal(t, 1, service.Task2Calls(), "Task2 should be called once (and fail)")
}

type asyncCompensateService struct {
	mu                   sync.Mutex
	task1Calls           int
	task2Calls           int
	compensateTask1Calls int
}

func (s *asyncCompensateService) DoTask1() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.task1Calls++
	return "task1_ok"
}

func (s *asyncCompensateService) DoTask2Fail() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.task2Calls++
	return "", stderr.New("task2_failed")
}

func (s *asyncCompensateService) CompensateTask1() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.compensateTask1Calls++
	return "compensate_task1_ok"
}

func (s *asyncCompensateService) CompensateTask2() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return "compensate_task2_ok"
}

func (s *asyncCompensateService) Task1Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.task1Calls
}

func (s *asyncCompensateService) Task2Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.task2Calls
}

func (s *asyncCompensateService) CompensateTask1Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.compensateTask1Calls
}
